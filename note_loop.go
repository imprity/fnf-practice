package main

import (
	"time"
	"fmt"

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
}

func UpdateNotesAndEvents(
	notes []FnfNote,
	event GameEvent,
	wasKeyPressed [2][NoteDirSize]bool,
	isKeyPressed  [2][NoteDirSize]bool,
	audioPos time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	botPlay bool,
	audioPosChanged bool,
	noteIndexStart int,
) GameEvent{
	event.AudioPosition = audioPos

	// things to do when position is arbitrarily changed
	if audioPosChanged {
		audioPosChanged = false

		newNoteIndexSet := false

		// reset note state
		for index, note := range notes {
			notes[index].IsMiss = false
			notes[index].IsHit = false
			notes[index].HoldReleaseAt = 0
			if !newNoteIndexSet &&
			(note.IsInWindow(audioPos, hitWindow) || note.IsAudioPositionInDuration(audioPos, hitWindow)) {
				newNoteIndexSet = true
				noteIndexStart = note.Index
			}
		}

		// if position is changed
		// for bots ignore the old input state set it all to not pressed
		botStart := 0
		if !botPlay {
			botStart = 1
		}

		for bot := botStart; bot <= 1; bot++ {
			for dir := range NoteDirSize {
				wasKeyPressed[bot][dir] = false
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
	}


	if isPlayingAudio{

		var isKeyJustPressed [2][NoteDirSize]bool
		var isKeyJustReleased [2][NoteDirSize]bool

		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if !wasKeyPressed[player][dir] && isKeyPressed[player][dir]{
					isKeyJustPressed[player][dir] = true
				}

				if wasKeyPressed[player][dir] && !isKeyPressed[player][dir]{
					isKeyJustReleased[player][dir] = true
				}

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

		newNoteIndexStart := noteIndexStart
		newNoteIndexSet := false

		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]
			//check if user missed note
			if !isKeyPressed[note.Player][note.Direction] &&
			!note.IsMiss && !note.IsHit &&
			note.StartsAt < audioPos-hitWindow/2 {

				notes[note.Index].IsMiss = true
				event.NoteMissAt[note.Player][note.Direction] = GlobalTimerNow()
			}

			if note.Duration > 0 && note.IsAudioPositionInDuration(audioPos, hitWindow) {
				if isKeyJustPressed[note.Player][note.Direction] {
					onNoteHold(note)
				}
			}

			//check if user hit note
			if note.IsInWindow(audioPos, hitWindow) && !note.IsHit && isKeyJustPressed[note.Player][note.Direction] {
				if !(didHitNote[note.Player][note.Direction] && hitNote[note.Player][note.Direction].Duration <= 0) {
					onNoteHit(note)
				} else if note.IsOverlapped(hitNote[note.Player][note.Direction]) {
					onNoteHit(note)
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

	return event
}

func GetKeyPressState(
	notes []FnfNote,
	noteIndexStart int,
	audioPos time.Duration,
	isBotPlay bool,
) [2][NoteDirSize]bool{

	keyPressState := GetBotKeyPresseState(notes, noteIndexStart, audioPos, isBotPlay)

	if !isBotPlay{
		for dir, key := range NoteKeys{
			if rl.IsKeyDown(key){
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
	audioPos time.Duration,
	isBotPlay bool,
) [2][NoteDirSize]bool {

	var keyPressed [2][NoteDirSize]bool

	const tinyWindow = time.Millisecond * 10

	for ; noteIndexStart < len(notes); noteIndexStart++ {
		note := notes[noteIndexStart]
		if isNoteForBot(note, isBotPlay) {
			if !note.IsHit && note.IsInWindow(audioPos, tinyWindow) {
				keyPressed[note.Player][note.Direction] = true
			} else if note.IsAudioPositionInDuration(audioPos, tinyWindow) {
				keyPressed[note.Player][note.Direction] = true
			}
		}
		if note.StartsAt > audioPos+tinyWindow {
			break
		}
	}

	return keyPressed
}
