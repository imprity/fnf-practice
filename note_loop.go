package main

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	//"github.com/ebitengine/oto/v3"
	//"sync"
)

type GameEvent struct {
	AudioPosition time.Duration

	HoldingNote   [2][NoteDirSize]FnfNote
	IsHoldingNote [2][NoteDirSize]bool

	// animation infos
	IsHoldingKey    [2][NoteDirSize]bool
	IsHoldingBadKey [2][NoteDirSize]bool

	KeyPressedAt  [2][NoteDirSize]time.Duration
	KeyReleasedAt [2][NoteDirSize]time.Duration

	NoteMissAt [2][NoteDirSize]time.Duration
	DidMissNote [2][NoteDirSize]bool
}

func NoteStartTunneled(
	note FnfNote, 
	hitWindow time.Duration,
	prevAudioPos time.Duration, 
	audioPos time.Duration,
)bool{
	tunneled := note.StartPassedHitWindow(audioPos, hitWindow) 
	tunneled = tunneled && note.NotReachedHitWindow(prevAudioPos, hitWindow) 
	return tunneled
}

func SustainNoteTunneled(
	note FnfNote, 
	hitWindow time.Duration,
	prevAudioPos time.Duration, 
	audioPos time.Duration,
)bool{
	tunneld := note.StartsAt + note.Duration < prevAudioPos - hitWindow / 2
	tunneld = tunneld && note.NotReachedHitWindow(audioPos, hitWindow)
	return tunneld
}

func UpdateNotesAndEvents(
	notes []FnfNote,
	event GameEvent,
	wasKeyPressed [2][NoteDirSize]bool,
	isKeyPressed [2][NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos     time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	botPlay bool,
	noteIndexStart int,
) (GameEvent, int) {
	event.AudioPosition = audioPos

	newNoteIndexStart := noteIndexStart

	if isPlayingAudio {

		var isKeyJustPressed [2][NoteDirSize]bool
		var isKeyJustReleased [2][NoteDirSize]bool

		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if !wasKeyPressed[player][dir] && isKeyPressed[player][dir] {
					isKeyJustPressed[player][dir] = true
				}

				if wasKeyPressed[player][dir] && !isKeyPressed[player][dir] {
					isKeyJustReleased[player][dir] = true
				}

			}
		}

		//clear note miss event
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				event.DidMissNote[player][dir] = false
			}
		}

		// declare convinience functions

		didHitNote := [2][NoteDirSize]bool{}
		hitNote := [2][NoteDirSize]FnfNote{}

		onNoteHit := func(note FnfNote) {
			// DEBUG!!!!!!!!!!!!!!!!!!!!!
			if !notes[note.Index].IsHit && note.Player == 0 {
				diff := AbsI(note.StartsAt - audioPos)
				fmt.Printf("hit note, %v\n", diff)
			}
			// DEBUG!!!!!!!!!!!!!!!!!!!!!

			notes[note.Index].IsHit = true
			event.IsHoldingBadKey[note.Player][note.Direction] = false
			didHitNote[note.Player][note.Direction] = true
			hitNote[note.Player][note.Direction] = note
		}

		onNoteHold := func(note FnfNote) {
			onNoteHit(note)

			event.HoldingNote[note.Player][note.Direction] = note
			event.IsHoldingNote[note.Player][note.Direction] = true
		}

		// we check if user pressed any key
		// and if so mark all as bad hit (it will be overidden as not bad later)
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if isKeyPressed[player][dir] && !event.IsHoldingKey[player][dir] {
					event.IsHoldingKey[player][dir] = true
					event.KeyPressedAt[player][dir] = GlobalTimerNow()

					event.IsHoldingBadKey[player][dir] = true
				} else if !isKeyPressed[player][dir] {
					if event.IsHoldingKey[player][dir] {
						event.KeyReleasedAt[player][dir] = GlobalTimerNow()
					}

					event.IsHoldingKey[player][dir] = false
					event.IsHoldingBadKey[player][dir] = false
				}
			}
		}

		// update any notes that were held but now no longer being held
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if !isKeyPressed[player][dir] && event.IsHoldingNote[player][dir] {
					note := event.HoldingNote[player][dir]
					notes[note.Index].HoldReleaseAt = audioPos

					event.IsHoldingNote[player][dir] = false
				}
			}
		}

		newNoteIndexSet := false

		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]

			//check if user hit note
			if isKeyJustPressed[note.Player][note.Direction]{
				var hittable bool

				if note.Duration > 0{
					hittable = note.IsAudioPositionInDuration(audioPos, hitWindow) 
					hittable = hittable || SustainNoteTunneled(note, hitWindow, prevAudioPos, audioPos) 
				}else{
					hittable = !note.IsHit
					hittable = hittable && note.IsInWindow(audioPos, hitWindow) 
					hittable = hittable || NoteStartTunneled(note, hitWindow, prevAudioPos, audioPos) 
				}

				hitElse := (didHitNote[note.Player][note.Direction] && hitNote[note.Player][note.Direction].Duration <= 0)
				hittable = hittable && (!hitElse || (hitElse && hitNote[note.Player][note.Direction].IsOverlapped(note))) 

				if hittable{
					if note.Duration > 0 {
						onNoteHold(note)
					}else{
						onNoteHit(note)
					}
				}
			}

			//check if user missed note
			if note.Duration > 0{
				missed := !event.IsHoldingNote[note.Player][note.Direction]
				missed = missed || (event.IsHoldingNote[note.Player][note.Direction] && !event.HoldingNote[note.Player][note.Direction].Equals(note))
				missed = missed && note.StartPassedHitWindow(audioPos, hitWindow)
				missed = missed && note.IsAudioPositionInDuration(audioPos, hitWindow)
				missed = missed && note.HoldReleaseAt < note.StartsAt + note.Duration
				if missed {
					event.DidMissNote[note.Player][note.Direction] = true
					event.NoteMissAt[note.Player][note.Direction] = GlobalTimerNow()
				}
			}else if !note.IsHit{
				wasInHitWindow := false
				isInHitWindow := false

				wasInHitWindow = note.IsInWindow(prevAudioPos, hitWindow)
				isInHitWindow = note.IsInWindow(audioPos, hitWindow)

				if wasInHitWindow && !isInHitWindow{
					event.DidMissNote[note.Player][note.Direction] = true
					event.NoteMissAt[note.Player][note.Direction] = GlobalTimerNow()
				}
			}

			if !newNoteIndexSet &&
				(note.IsInWindow(audioPos, hitWindow) ||
					note.IsAudioPositionInDuration(audioPos, hitWindow)) {

				newNoteIndexSet = true
				newNoteIndexStart = note.Index
			}

			if note.StartsAt > audioPos+hitWindow {
				break
			}
		}
		noteIndexStart = newNoteIndexStart
	}

	return event, newNoteIndexStart
}

func GetKeyPressState(
	notes []FnfNote,
	noteIndexStart int,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isBotPlay bool,
	hitWindow time.Duration,
) [2][NoteDirSize]bool {

	keyPressState := GetBotKeyPresseState(
		notes, noteIndexStart, prevAudioPos, audioPos, isBotPlay, hitWindow)

	if !isBotPlay {
		for dir, key := range NoteKeys {
			if rl.IsKeyDown(key) {
				keyPressState[0][dir] = true
			}
		}
	}

	return keyPressState
}

func isNoteForBot(note FnfNote, isBotPlay bool) bool {
	if isBotPlay {
		return true
	}

	return note.Player >= 1
}

func GetBotKeyPresseState(
	notes []FnfNote,
	noteIndexStart int,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isBotPlay bool,
	hitWindow time.Duration,
) [2][NoteDirSize]bool {

	var keyPressed [2][NoteDirSize]bool

	const tinyWindow = time.Millisecond * 10
	//var tinyWindow = hitWindow

	for ; noteIndexStart < len(notes); noteIndexStart++ {
		note := notes[noteIndexStart]
		if isNoteForBot(note, isBotPlay) {
			if !note.IsHit && (note.IsInWindow(audioPos, tinyWindow) || NoteStartTunneled(note, prevAudioPos, audioPos, tinyWindow)){
				keyPressed[note.Player][note.Direction] = true
			} else if note.IsAudioPositionInDuration(audioPos, tinyWindow) || SustainNoteTunneled(note, prevAudioPos, audioPos, tinyWindow) {
				keyPressed[note.Player][note.Direction] = true
			}
		}
		if note.StartsAt > audioPos+tinyWindow {
			break
		}
	}

	return keyPressed
}
