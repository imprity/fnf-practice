package main

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type PlayerState struct {
	HoldingNote   [NoteDirSize]FnfNote
	IsHoldingNote [NoteDirSize]bool

	// animation infos
	IsHoldingKey    [NoteDirSize]bool
	IsHoldingBadKey [NoteDirSize]bool

	KeyPressedAt  [NoteDirSize]time.Duration
	KeyReleasedAt [NoteDirSize]time.Duration
	DidReleaseBadKey [NoteDirSize]bool

	NoteMissAt [NoteDirSize]time.Duration
	DidMissNote [NoteDirSize]bool
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
	pStates [2]PlayerState,
	wasKeyPressed [2][NoteDirSize]bool,
	isKeyPressed [2][NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos     time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	botPlay bool,
	noteIndexStart int,
) ([2]PlayerState, int) {
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
				pStates[player].DidMissNote[dir] = false
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
			pStates[note.Player].IsHoldingBadKey[note.Direction] = false
			didHitNote[note.Player][note.Direction] = true
			hitNote[note.Player][note.Direction] = note
		}

		onNoteHold := func(note FnfNote) {
			onNoteHit(note)

			pStates[note.Player].HoldingNote[note.Direction] = note
			pStates[note.Player].IsHoldingNote[note.Direction] = true
		}

		// we check if user pressed any key
		// and if so mark all as bad hit (it will be overidden as not bad later)
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if isKeyPressed[player][dir] && !pStates[player].IsHoldingKey[dir] {
					pStates[player].IsHoldingKey[dir] = true
					pStates[player].KeyPressedAt[dir] = GlobalTimerNow()

					pStates[player].IsHoldingBadKey[dir] = true
				} else if !isKeyPressed[player][dir] {
					if pStates[player].IsHoldingKey[dir] {
						pStates[player].KeyReleasedAt[dir] = GlobalTimerNow()

						if pStates[player].IsHoldingBadKey[dir]{
							pStates[player].DidReleaseBadKey[dir] = true
						}else{
							pStates[player].DidReleaseBadKey[dir] = false
						}
					}

					pStates[player].IsHoldingKey[dir] = false
					pStates[player].IsHoldingBadKey[dir] = false
				}
			}
		}

		// update any notes that were held but now no longer being held
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				if !isKeyPressed[player][dir] && pStates[player].IsHoldingNote[dir] {
					note := pStates[player].HoldingNote[dir]
					notes[note.Index].HoldReleaseAt = audioPos

					pStates[player].IsHoldingNote[dir] = false
				}
			}
		}

		newNoteIndexSet := false

		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]

			np := note.Player
			nd := note.Direction

			//check if user hit note
			if isKeyJustPressed[np][nd]{
				var hit bool

				if note.IsSustain(){
					hit = note.IsAudioPositionInDuration(audioPos, hitWindow)
					hit = hit || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow)
				}else{
					hit = !note.IsHit
					hit = hit && note.IsInWindow(audioPos, hitWindow)
					hit = hit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
				}

				hitElse := (didHitNote[np][nd] && !hitNote[np][nd].IsSustain())
				hit = hit && (!hitElse || (hitElse && hitNote[np][nd].IsOverlapped(note)))

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
				if isKeyPressed[np][nd]{
					onNoteHold(note)
				}
			}

			//check if user missed note
			if note.IsSustain(){
				missed := !pStates[np].IsHoldingNote[nd]
				missed = missed || (pStates[np].IsHoldingNote[nd] && !pStates[np].HoldingNote[nd].Equals(note))
				missed = missed && note.StartPassedHitWindow(audioPos, hitWindow)
				missed = missed && note.IsAudioPositionInDuration(audioPos, hitWindow)
				missed = missed && note.HoldReleaseAt < note.StartsAt + note.Duration
				if missed {
					pStates[np].DidMissNote[nd] = true
					pStates[np].NoteMissAt[nd] = GlobalTimerNow()
				}
			}else if !note.IsHit{
				wasInHitWindow := false
				isInHitWindow := false

				wasInHitWindow = note.IsInWindow(prevAudioPos, hitWindow)
				isInHitWindow = note.IsInWindow(audioPos, hitWindow)

				if wasInHitWindow && !isInHitWindow{
					pStates[np].DidMissNote[nd] = true
					pStates[np].NoteMissAt[nd] = GlobalTimerNow()
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

	return pStates, newNoteIndexStart
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
