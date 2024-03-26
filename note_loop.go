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
	SetPlayAudio     chan bool
	SetSpeed         chan float64
	SetAudioPosition chan time.Duration
	SetBotPlay       chan bool

	EventData ReadChannel[LoopEventData]
	UpdatedNotes ReadChannel[FnfNote]
}

func MakeLoopChannels(notesSize int64) LoopChannels{
	return LoopChannels{
		SetPlayAudio : make(chan bool),
		SetSpeed : make(chan float64),
		SetAudioPosition : make(chan time.Duration),
		SetBotPlay : make(chan bool),
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

	BotPlay bool
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

		botPlay := initData.BotPlay

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

		var isKeyPressed [2][NoteDirSize]bool

		// update audio position
		event.AudioPosition = instPlayer.Position()
		audioPos := event.AudioPosition

		var noteIndexStart int

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

				newNoteIndexSet := false

				// reset note state
				for index, note := range notes {
					notes[index].IsMiss = false
					notes[index].IsHit = false
					notes[index].HoldReleaseAt = 0
					if (
						!newNoteIndexSet &&
						(note.IsInWindow(audioPos, hitWindow) ||note.IsAudioPositionInDuration(audioPos, hitWindow))){
						newNoteIndexSet = true
						noteIndexStart = note.Index
						break
					}
				}

				// reset input state for bots
				botStart := 0
				if !botPlay{
					botStart = 1
				}

				for bot := botStart; bot <= 1; bot++{
					for dir := range NoteDirSize{
						isKeyPressed[bot][dir] = false
					}
				}

				// reset event state
				for player := 0; player <= 1; player++ {
					for dir := range NoteDirSize {
						event.IsHoldingNote[player][dir] = false
						event.IsHoldingKey[player][dir] = false
						event.IsHoldingBadKey[player][dir] = false

						event.NoteMissAt[player][dir] = 0
					}
				}
			case bot := <- channels.SetBotPlay :
				botPlay = bot
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

				var isKeyJustPressed  [2][NoteDirSize]bool
				var isKeyJustReleased [2][NoteDirSize]bool

				// update key state for bot
				keyPressState := GetBotKeyPresseState(notes, noteIndexStart, audioPos, botPlay)

				if !botPlay{
					for dir, key := range NoteKeys{
						keyPressState[0][dir] = ebiten.IsKeyPressed(key)
					}
				}

				for player := 0; player <=1; player ++{
					for dir := range NoteDirSize{
						if keyPressState[player][dir]{
							if !isKeyPressed[player][dir]{
								isKeyJustPressed[player][dir] = true
							}

							isKeyPressed[player][dir] = true
						}else{
							if isKeyPressed[player][dir]{
								isKeyJustReleased[player][dir] = true
							}

							isKeyPressed[player][dir] = false
						}
					}
				}

				// declare convinience functions

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

				// we check if user pressed any key
				// and if so mark all as bad hit (it will be overidden as not bad later)
				for player := 0; player <= 1; player++{
					for dir := range NoteDirSize{
						if isKeyPressed[player][dir] && !event.IsHoldingKey[player][dir] {
							event.IsHoldingKey[player][dir] = true
							event.KeyPressedAt[player][dir] = TimeSinceStart()

							event.IsHoldingBadKey[player][dir] = true
						} else if event.IsHoldingKey[player][dir] && !isKeyPressed[player][dir] {
							event.IsHoldingKey[player][dir] = false
							event.KeyReleasedAt[player][dir] = TimeSinceStart()

							event.IsHoldingBadKey[player][dir] = false
						}
					}
				}

				// update any notes that were held but now no longer being held
				for player := 0; player <= 1; player++{
					for dir := range NoteDirSize {
						if !isKeyPressed[player][dir] && event.IsHoldingNote[player][dir] {
							note := event.HoldingNote[player][dir]
							notes[note.Index].HoldReleaseAt = audioPos

							event.IsHoldingNote[player][dir] = false
						}
					}
				}

				newNoteIndexStart := noteIndexStart
				newNoteIndexSet := false

				for ;noteIndexStart <len(notes); noteIndexStart ++ {
					note := notes[noteIndexStart]
					//check if user missed note
					if !isKeyPressed[note.Player][note.Direction] &&
						!note.IsMiss && !note.IsHit &&
						note.StartsAt < audioPos-hitWindow/2 {

						notes[note.Index].IsMiss = true
						event.NoteMissAt[note.Player][note.Direction] = TimeSinceStart()
					}

					if note.Duration > 0 && note.IsAudioPositionInDuration(audioPos, hitWindow) {
						if isKeyJustPressed[note.Player][note.Direction] {
							onNoteHit(note)
							onNoteHold(note)
						}
					}

					//check if user hit note
					if note.IsInWindow(audioPos, hitWindow) && !note.IsHit && isKeyJustPressed[note.Player][note.Direction] {
						if !didHitNote[note.Player][note.Direction]{
							onNoteHit(note)
						}else if note.IsOverlapped(hitNote[note.Player][note.Direction]){
							onNoteHit(note)
						}
					}

					if !newNoteIndexSet &&
						(
							note.IsInWindow(audioPos, hitWindow) ||
							note.IsAudioPositionInDuration(audioPos, hitWindow)){

						newNoteIndexSet = true
						newNoteIndexStart = note.Index
					}

					if note.StartsAt > audioPos + hitWindow{
						break
					}
				}
				noteIndexStart = newNoteIndexStart
			}
		}
	}()
}

func isNoteForBot(note FnfNote, isBotPlay bool) bool{
	if isBotPlay{
		return true
	}

	return note.Player >= 1
}

func GetBotKeyPresseState(notes []FnfNote, noteIndexStart int, audioPos time.Duration, isBotPlay bool) [2][NoteDirSize] bool{
	var keyPressed [2][NoteDirSize] bool

	const tinyWindow = time.Millisecond * 10

	for ;noteIndexStart < len(notes); noteIndexStart++{
		note := notes[noteIndexStart]
		if isNoteForBot(note, isBotPlay){
			if !note.IsHit && note.IsInWindow(audioPos, tinyWindow){
				keyPressed[note.Player][note.Direction] = true
			} else if note.IsAudioPositionInDuration(audioPos, tinyWindow){
				keyPressed[note.Player][note.Direction] = true
			}
		}
		if note.StartsAt > audioPos + tinyWindow{
			break
		}
	}

	return keyPressed
}
