package main

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type PlayerState struct {
	HoldingNote   [2][NoteDirSize]FnfNote
	IsHoldingNote [2][NoteDirSize]bool

	// animation infos
	IsHoldingKey    [2][NoteDirSize]bool
	IsHoldingBadKey [2][NoteDirSize]bool

	KeyPressedAt  [2][NoteDirSize]time.Duration
	KeyReleasedAt [2][NoteDirSize]time.Duration
	DidReleaseBadKey [2][NoteDirSize]bool

	NoteMissAt [2][NoteDirSize]time.Duration
	DidMissNote [2][NoteDirSize]bool
}

func NoteStartTunneled(
	note FnfNote,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	hitWindow time.Duration,
)bool{
	tunneled := note.StartPassedHitWindow(audioPos, hitWindow)
	tunneled = tunneled && note.NotReachedHitWindow(prevAudioPos, hitWindow)
	return tunneled
}

func SustainNoteTunneled(
	note FnfNote,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	hitWindow time.Duration,
)bool{
	tunneled := note.StartsAt + note.Duration < prevAudioPos - hitWindow / 2
	tunneled = tunneled && note.NotReachedHitWindow(audioPos, hitWindow)
	return tunneled
}

func UpdateNotesAndStates(
	notes []FnfNote,
	pState PlayerState,
	wasKeyPressed [2][NoteDirSize]bool,
	isKeyPressed [2][NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos     time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	botPlay bool,
	noteIndexStart int,
) (PlayerState, int) {
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

		//clear note miss state
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				pState.DidMissNote[player][dir] = false
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
			pState.IsHoldingBadKey[note.Player][note.Direction] = false
			didHitNote[note.Player][note.Direction] = true
			hitNote[note.Player][note.Direction] = note
		}

		onNoteHold := func(note FnfNote) {
			onNoteHit(note)

			pState.HoldingNote[note.Player][note.Direction] = note
			pState.IsHoldingNote[note.Player][note.Direction] = true
		}

		// we check if user pressed any key
		// and if so mark all as bad hit (it will be overidden as not bad later)
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if isKeyPressed[player][dir] && !pState.IsHoldingKey[player][dir] {
					pState.IsHoldingKey[player][dir] = true
					pState.KeyPressedAt[player][dir] = GlobalTimerNow()

					pState.IsHoldingBadKey[player][dir] = true
				} else if !isKeyPressed[player][dir] {
					if pState.IsHoldingKey[player][dir] {
						pState.KeyReleasedAt[player][dir] = GlobalTimerNow()

						if pState.IsHoldingBadKey[player][dir]{
							pState.DidReleaseBadKey[player][dir] = true
						}else{
							pState.DidReleaseBadKey[player][dir] = false
						}
					}

					pState.IsHoldingKey[player][dir] = false
					pState.IsHoldingBadKey[player][dir] = false
				}
			}
		}

		// update any notes that were held but now no longer being held
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if !isKeyPressed[player][dir] && pState.IsHoldingNote[player][dir] {
					note := pState.HoldingNote[player][dir]
					notes[note.Index].HoldReleaseAt = audioPos

					pState.IsHoldingNote[player][dir] = false
				}
			}
		}

		newNoteIndexSet := false

		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]

			//check if user hit note
			if isKeyJustPressed[note.Player][note.Direction]{
				var hit bool

				if note.IsSustain(){
					hit = note.IsAudioPositionInDuration(audioPos, hitWindow)
					hit = hit || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow)
				}else{
					hit = !note.IsHit
					hit = hit && note.IsInWindow(audioPos, hitWindow)
					hit = hit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
				}

				hitElse := (didHitNote[note.Player][note.Direction] && !hitNote[note.Player][note.Direction].IsSustain())
				hit = hit && (!hitElse || (hitElse && hitNote[note.Player][note.Direction].IsOverlapped(note)))

				if hit{
					if note.IsSustain() {
						onNoteHold(note)
					}else{
						onNoteHit(note)
					}
				}
			}

			// if sustain note passed hit window and key is pressed
			// just treat it as good enough
			if (note.IsSustain() &&
				!note.IsHit &&
				note.StartPassedHitWindow(audioPos, hitWindow) &&
				note.IsAudioPositionInDuration(audioPos, hitWindow)){
				if isKeyPressed[note.Player][note.Direction]{
					onNoteHold(note)
				}
			}

			//check if user missed note
			if note.IsSustain(){
				missed := !pState.IsHoldingNote[note.Player][note.Direction]
				missed = missed || (pState.IsHoldingNote[note.Player][note.Direction] && !pState.HoldingNote[note.Player][note.Direction].Equals(note))
				missed = missed && note.StartPassedHitWindow(audioPos, hitWindow)
				missed = missed && note.IsAudioPositionInDuration(audioPos, hitWindow)
				missed = missed && note.HoldReleaseAt < note.StartsAt + note.Duration
				if missed {
					pState.DidMissNote[note.Player][note.Direction] = true
					pState.NoteMissAt[note.Player][note.Direction] = GlobalTimerNow()
				}
			}else if !note.IsHit{
				wasInHitWindow := false
				isInHitWindow := false

				wasInHitWindow = note.IsInWindow(prevAudioPos, hitWindow)
				isInHitWindow = note.IsInWindow(audioPos, hitWindow)

				if wasInHitWindow && !isInHitWindow{
					pState.DidMissNote[note.Player][note.Direction] = true
					pState.NoteMissAt[note.Player][note.Direction] = GlobalTimerNow()
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

	return pState, newNoteIndexStart
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

	for ; noteIndexStart < len(notes); noteIndexStart++ {
		note := notes[noteIndexStart]
		if isNoteForBot(note, isBotPlay) {
			if !note.IsHit && !note.IsSustain(){
				shouldHit := note.StartsAt <= audioPos && note.StartsAt >= prevAudioPos
				shouldHit = shouldHit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
				if shouldHit{
					keyPressed[note.Player][note.Direction] = true
				}
			} else if note.IsAudioPositionInDuration(audioPos, tinyWindow) || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow) {
				keyPressed[note.Player][note.Direction] = true
			}
		}
		if note.StartsAt > audioPos+tinyWindow {
			break
		}
	}

	return keyPressed
}
