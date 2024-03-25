package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"

	"kitty"
	"fmt"
)

type LoopEventData struct{
	AudioPosition time.Duration

	HoldingNote   [2][NoteDirSize]FnfNote
	IsHoldingNote [2][NoteDirSize]bool

	// animation infos
	IsHoldingKey    [2][NoteDirSize]bool
	IsHoldingBadKey [2][NoteDirSize]bool

	KeyPressedAt  [2][NoteDirSize]time.Duration
	KeyReleasedAt [2][NoteDirSize]time.Duration

	NoteMissAt [2][NoteDirSize]time.Duration
}

type ReadChannel[T any] struct{
	RequestChannel chan bool
	DataChannel    chan T
}

func (rc ReadChannel[T]) RequestRead(){
    rc.RequestChannel <- true
}

func (rc ReadChannel[T]) Read() T{
    return <- rc.DataChannel
}

type LoopChannels struct{
	SetPlayAudio chan bool
	SetSpeed     chan float64
	SetAudioPosition chan time.Duration

	EventData ReadChannel[LoopEventData]
	UpdatedNotes ReadChannel[FnfNote]
}

func MakeLoopChannels(notesSize int64) LoopChannels{
	return LoopChannels{
		SetPlayAudio : make(chan bool),
		SetSpeed : make(chan float64),
		SetAudioPosition : make(chan time.Duration),
		EventData : ReadChannel[LoopEventData]{
			RequestChannel : make(chan bool),
			DataChannel : make(chan LoopEventData),
		},
		UpdatedNotes : ReadChannel[FnfNote]{
			RequestChannel : make(chan bool),
			DataChannel : make(chan FnfNote, notesSize),
		},
	}
}

type LoopInitData struct{
	Channels LoopChannels

	HitWindow time.Duration

	Song FnfSong

	AudioContext *audio.Context

	VoiceAudioBytes []byte
	InstAudioBytes []byte

	PlayVoice bool
}

func StartAudioGameLoop(initData LoopInitData) {
	notes := make([]FnfNote, len(initData.Song.Notes))

	copy(notes, initData.Song.Notes)

	playVoice := initData.PlayVoice

	hitWindow := initData.HitWindow
	channels := initData.Channels

	context := initData.AudioContext

	go func() {
		var instPlayer  *VaryingSpeedPlayer
		var voicePlayer *VaryingSpeedPlayer

		botPlay := false

		{
			var err error

			instPlayer, err = NewVaryingSpeedPlayer(context, initData.InstAudioBytes)
			if err != nil{
				ErrorLogger.Fatal(err)
			}
			if playVoice{
				voicePlayer, err = NewVaryingSpeedPlayer(context, initData.VoiceAudioBytes)
				if err != nil{
					ErrorLogger.Fatal(err)
				}
			}
		}

		var event LoopEventData

		var isKeyPressed [NoteDirSize]bool

		// update audio position
		event.AudioPosition = instPlayer.Position()
		audioPos := event.AudioPosition

		for {
			select{
			case play := <- channels.SetPlayAudio:
				if play{
					instPlayer.Play()
					if playVoice{
						voicePlayer.Play()
					}
				}else{
					instPlayer.Pause()
					if playVoice{
						voicePlayer.Pause()
					}
				}
			case speed := <- channels.SetSpeed :
				instPlayer.SetSpeed(speed)
				if playVoice{
					voicePlayer.SetSpeed(speed)
				}
			case position := <- channels.SetAudioPosition:
				instPlayer.SetPosition(position)
				if playVoice{
					voicePlayer.SetPosition(position)
				}

				for index, _ := range notes {
					notes[index].IsMiss = false
					notes[index].IsHit = false
					notes[index].HoldReleaseAt = 0
				}

				for dir := range NoteDirSize {
					for player := 0; player <= 1; player++ {
						event.IsHoldingNote[player][dir] = false
						event.IsHoldingKey[player][dir] = false
						event.IsHoldingBadKey[player][dir] = false

						event.NoteMissAt[player][dir] = 0
					}
				}
			case <- channels.EventData.RequestChannel:
				channels.EventData.DataChannel <- event
			case <- channels.UpdatedNotes.RequestChannel:
				for _, note := range notes{
					channels.UpdatedNotes.DataChannel <- note
				}
			default :
				//pass
			}

			instPlayer.Update()
			if playVoice {
				voicePlayer.Update()
			}

			// update audio position
			event.AudioPosition = instPlayer.Position()
			audioPos = event.AudioPosition

			if instPlayer.IsPlaying() {

				// update key state
				var isKeyJustPressed  [NoteDirSize]bool
				var isKeyJustReleased [NoteDirSize]bool

				for dir, key := range NoteKeys{
					if ebiten.IsKeyPressed(key){
						if !isKeyPressed[dir]{
							isKeyJustPressed[dir] = true
						}

						isKeyPressed[dir] = true
					}else{
						if isKeyPressed[dir]{
							isKeyJustReleased[dir] = true
						}

						isKeyPressed[dir] = false
					}
				}

				// declare convinience functions
				const tinyWindow = time.Millisecond * 10

				tinyWindowStart := audioPos - tinyWindow/2
				tinyWindowEnd := audioPos + tinyWindow/2

				windowStart := audioPos - hitWindow/2
				windowEnd := audioPos + hitWindow/2

				inWindow := func(note FnfNote) bool {
					return windowStart <= note.StartsAt && note.StartsAt <= windowEnd
				}

				inTinyWindow := func(note FnfNote) bool {
					return tinyWindowStart <= note.StartsAt && note.StartsAt <= tinyWindowEnd
				}

				didHitNote := [2][NoteDirSize]bool{}
				hitNote := [2][NoteDirSize]FnfNote{}

				onNoteHold := func(note FnfNote) {
					event.HoldingNote[note.Player][note.Direction] = note
					event.IsHoldingNote[note.Player][note.Direction] = true

					event.IsHoldingBadKey[note.Player][note.Direction] = false
					didHitNote[note.Player][note.Direction] = true
				}

				onNoteHit := func(note FnfNote) {
					// DEBUG!!!!!!!!!!!!!!!!!!!!!
					if !notes[note.Index].IsHit && note.Player == 0{
						diff := kitty.AbsI(note.StartsAt - audioPos)
						fmt.Printf("hit note, %v\n", diff)
					}
					// DEBUG!!!!!!!!!!!!!!!!!!!!!
					notes[note.Index].IsHit = true
					event.IsHoldingBadKey[note.Player][note.Direction] = false
					didHitNote[note.Player][note.Direction] = true
					hitNote[note.Player][note.Direction] = note
				}

				posAtNoteDuration := func(note FnfNote) bool {
					return (audioPos >= note.StartsAt-hitWindow/2 &&
						audioPos <= note.StartsAt+note.Duration+hitWindow/2)
				}

				// we check if user pressed any key
				// and if so mark all as bad hit (it will be overidden as not bad later)
				if !botPlay {
					for dir := range NoteDirSize{
						if isKeyPressed[dir] && !event.IsHoldingKey[0][dir] {
							event.IsHoldingKey[0][dir] = true
							event.KeyPressedAt[0][dir] = TimeSinceStart()

							event.IsHoldingBadKey[0][dir] = true
						} else if event.IsHoldingKey[0][dir] && !isKeyPressed[dir] {
							event.IsHoldingKey[0][dir] = false
							event.KeyReleasedAt[0][dir] = TimeSinceStart()

							event.IsHoldingBadKey[0][dir] = false
						}
					}
				}

				// update any notes that were held but now no longer being held
				if !botPlay {
					for dir := range NoteDirSize {
						if !isKeyPressed[dir] && event.IsHoldingNote[0][dir] {
							note := event.HoldingNote[0][dir]
							notes[note.Index].HoldReleaseAt = audioPos

							event.IsHoldingNote[0][dir] = false
						}
					}
				}

				for index, note := range notes {
					// hit the notes if note belongs to player 1 or we are in BotPlay mode
					if note.Player == 1 || botPlay {
						if inTinyWindow(note) {
							if !didHitNote[note.Player][note.Direction]{
								onNoteHit(note)
							}else if IsNoteOverlapped(hitNote[note.Player][note.Direction], note){
								onNoteHit(note)
							}

							event.IsHoldingKey[note.Player][note.Direction] = true
							event.KeyPressedAt[note.Player][note.Direction] = TimeSinceStart()

							if note.Duration > 0 {
								onNoteHold(note)
							}

						} else if !note.IsHit && !note.IsMiss && note.StartsAt < audioPos-tinyWindow*2 {
							// TODO : THIS SHOULD NOT HAPPEN!!!!!!!!!!!!!!!!
							// WE ARE CHECKING EVERY FRAME TO SEE IF WE HIT ANY NOTES!!!!!!
							// AND IT SOME HOW MISSES !!!!!!!!!!!
							// EVEN IF WE ARE ONLY CHECKING NOTES BETWEEN CERTAIN WINDOW FRAMES
							// THIS IS NOT FUCKING ACCEPTABLE!!!!!!!
							t := kitty.AbsI(note.StartsAt-audioPos) - 5

							fmt.Printf("missed by %v\n", t)

							notes[index].IsMiss = true
							event.NoteMissAt[note.Player][note.Direction] = TimeSinceStart()
						}

						if note.Duration > 0 && posAtNoteDuration(note) {
							if event.IsHoldingNote[note.Player][note.Direction]{
								didHitNote[note.Player][note.Direction] = true
								hitNote[note.Player][note.Direction] = note
							}else{
								onNoteHit(note)
								onNoteHold(note)
							}
						}
					} else { // note IS player 0 and we are not in bot play

						//check if user missed note
						if !isKeyPressed[note.Direction] &&
							!note.IsMiss && !note.IsHit &&
							note.StartsAt < audioPos-hitWindow/2 {

							notes[index].IsMiss = true
							event.NoteMissAt[note.Player][note.Direction] = TimeSinceStart()
						}

						if note.Duration > 0 && posAtNoteDuration(note) {
							if isKeyJustPressed[note.Direction] {
								onNoteHit(note)
								onNoteHold(note)
							}
						}

						//check if user hit note
						if inWindow(note) && isKeyJustPressed[note.Direction] {
							if !didHitNote[note.Player][note.Direction]{
								onNoteHit(note)
							}else if IsNoteOverlapped(hitNote[note.Player][note.Direction], note){
								onNoteHit(note)
							}
						}

					}
				}

				pStart := 1
				if botPlay {
					pStart = 0
				}

				for dir := range NoteDirSize {
					for player := pStart; player <= 1; player++ {
						if !didHitNote[player][dir]{
							event.IsHoldingKey[player][dir] = false
							event.KeyReleasedAt[player][dir] = TimeSinceStart()

							if event.IsHoldingNote[player][dir] {
								note := event.HoldingNote[player][dir]
								notes[note.Index].HoldReleaseAt = audioPos
							}

							event.IsHoldingNote[player][dir] = false
						}
					}
				}
			}
		}
	}()
}
