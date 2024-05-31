package main

import (
	"time"
)

type PlayerState struct {
	HoldingNote   [NoteDirSize]FnfNote
	IsHoldingNote [NoteDirSize]bool

	IsHoldingKey    [NoteDirSize]bool
	IsHoldingBadKey [NoteDirSize]bool

	IsKeyJustPressed  [NoteDirSize]bool
	IsKeyJustReleased [NoteDirSize]bool

	// animation infos

	// since these are for animations and stuff,
	// time is in real time (i.e. time since app started)
	KeyPressedAt     [NoteDirSize]time.Duration
	KeyReleasedAt    [NoteDirSize]time.Duration
	DidReleaseBadKey [NoteDirSize]bool

	// this is also in real time
	NoteMissAt  [NoteDirSize]time.Duration
	DidMissNote [NoteDirSize]bool
}

type NoteEvent struct {
	// EventBit can have 6 different values
	//
	// none : 0000
	//
	// hit               : 0001
	// hit and first hit : 0011
	//
	// release           : 1000
	// miss              : 0100
	// release and miss  : 1100
	//
	// this is done this way because I was afraid that I might set conflicting state
	// but I'm not sure if it's a good appoach
	EventBit int

	// this is in audio time (i.e. audio position when this event happened)
	Time time.Duration

	Index int
}

func (ne *NoteEvent) ClearHit() {
	ne.EventBit = ne.EventBit & 0b1100
}

func (ne *NoteEvent) ClearRelease() {
	ne.EventBit = ne.EventBit & 0b0011
}

func (ne *NoteEvent) SetHit() {
	ne.ClearRelease()
	ne.EventBit = ne.EventBit | 0b0001
}

func (ne *NoteEvent) SetFirstHit() {
	ne.ClearRelease()
	ne.EventBit = 0b0011
}

func (ne *NoteEvent) SetRelease() {
	ne.ClearHit()
	ne.EventBit = ne.EventBit | 0b1000
}

func (ne *NoteEvent) SetMiss() {
	ne.ClearHit()
	ne.EventBit = ne.EventBit | 0b0100
}

func (ne *NoteEvent) IsHit() bool {
	return ne.EventBit&0b0001 > 0
}

func (ne *NoteEvent) IsFirstHit() bool {
	return ne.EventBit == 0b0011
}

func (ne *NoteEvent) IsRelease() bool {
	return ne.EventBit&0b1000 > 0
}

func (ne *NoteEvent) IsMiss() bool {
	return ne.EventBit&0b0100 > 0
}

func (ne *NoteEvent) IsNone() bool {
	return ne.EventBit == 0
}

func (ne *NoteEvent) SameKind(other NoteEvent) bool {
	return ne.EventBit == other.EventBit
}

func NoteStartTunneled(
	note FnfNote,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	hitWindow time.Duration,
) bool {
	tunneled := note.StartPassedWindow(audioPos, hitWindow)
	tunneled = tunneled && note.NotReachedWindow(prevAudioPos, hitWindow)
	return tunneled
}

func SustainNoteTunneled(
	note FnfNote,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	hitWindow time.Duration,
) bool {
	tunneled := note.PassedWindow(audioPos, hitWindow)
	tunneled = tunneled && note.NotReachedWindow(prevAudioPos, hitWindow)
	return tunneled
}

func UpdateNotesAndStates(
	notes []FnfNote,
	pStates [2]PlayerState,
	wasKeyPressed [2][NoteDirSize]bool,
	isKeyPressed [2][NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	audioDuration time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	botPlay bool,
	noteIndexStart int,
) ([2]PlayerState, []NoteEvent, int) {
	newNoteIndexStart := noteIndexStart

	var noteEvents []NoteEvent

	avgPos := (audioPos + prevAudioPos) / 2

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

		for player := 0; player <= 1; player++ {
			pStates[player].IsKeyJustPressed = isKeyJustPressed[player]
			pStates[player].IsKeyJustReleased = isKeyJustReleased[player]
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

		onNoteHit := func(note FnfNote, event *NoteEvent) {
			if !notes[note.Index].IsHit {
				event.SetFirstHit()
			} else {
				event.SetHit()
			}

			notes[note.Index].IsHit = true
			pStates[note.Player].IsHoldingBadKey[note.Direction] = false
			didHitNote[note.Player][note.Direction] = true
			hitNote[note.Player][note.Direction] = note
		}

		onNoteHold := func(note FnfNote, event *NoteEvent) {
			onNoteHit(note, event)

			pStates[note.Player].HoldingNote[note.Direction] = note
			pStates[note.Player].IsHoldingNote[note.Direction] = true
		}

		onNoteMiss := func(note FnfNote, event *NoteEvent) {
			event.SetMiss()

			pStates[note.Player].DidMissNote[note.Direction] = true
			pStates[note.Player].NoteMissAt[note.Direction] = GlobalTimerNow()
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

						if pStates[player].IsHoldingBadKey[dir] {
							pStates[player].DidReleaseBadKey[dir] = true
						} else {
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

					event := NoteEvent{
						Time:  avgPos,
						Index: note.Index,
					}
					event.SetRelease()

					noteEvents = append(noteEvents, event)
				}
			}
		}

		newNoteIndexSet := false

		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]

			np := note.Player
			nd := note.Direction

			event := NoteEvent{
				Time:  avgPos,
				Index: note.Index,
			}

			//check if user hit note
			if isKeyJustPressed[np][nd] {
				var hit bool

				if note.IsSustain() {
					hit = note.IsAudioPositionInDuration(audioPos, hitWindow)
					hit = hit || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow)
				} else {
					hit = !note.IsHit
					hit = hit && note.IsStartInWindow(audioPos, hitWindow)
					hit = hit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
				}

				hitElse := (didHitNote[np][nd] && !hitNote[np][nd].IsSustain())
				hit = hit && (!hitElse || (hitElse && hitNote[np][nd].IsOverlapped(note)))

				if hit {
					if note.IsSustain() {
						onNoteHold(note, &event)
					} else {
						onNoteHit(note, &event)
					}
				}
			}

			// if sustain note passed hit window and key is pressed
			// just treat it as good enough
			if note.IsSustain() &&
				!note.IsHit &&
				note.StartPassedWindow(audioPos, hitWindow) &&
				note.IsAudioPositionInDuration(audioPos, hitWindow) {
				if isKeyPressed[np][nd] {
					onNoteHold(note, &event)
				}
			}

			//check if user missed note
			if note.IsSustain() {
				missed := !pStates[np].IsHoldingNote[nd]
				missed = missed || (pStates[np].IsHoldingNote[nd] && !pStates[np].HoldingNote[nd].Equals(note))
				missed = missed && note.StartPassedWindow(audioPos, hitWindow)
				missed = missed && note.IsAudioPositionInDuration(audioPos, hitWindow)
				missed = missed && note.HoldReleaseAt < note.StartsAt+note.Duration
				if missed {
					onNoteMiss(note, &event)
				}
			} else if !note.IsHit {
				wasInHitWindow := false
				isInHitWindow := false

				wasInHitWindow = note.IsStartInWindow(prevAudioPos, hitWindow)
				isInHitWindow = note.IsStartInWindow(audioPos, hitWindow)

				if wasInHitWindow && !isInHitWindow {
					onNoteMiss(note, &event)
				}
			}

			if !event.IsNone() {
				noteEvents = append(noteEvents, event)
			}

			if !newNoteIndexSet &&
				(note.IsStartInWindow(audioPos, hitWindow) ||
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

	if !isPlayingAudio && AbsI(audioDuration-audioPos) < time.Millisecond { // when song is done
		for player := 0; player <= 1; player++ {
			for dir := range NoteDirSize {
				// release notes that are being held
				if pStates[player].IsHoldingNote[dir] {
					note := pStates[player].HoldingNote[dir]
					notes[note.Index].HoldReleaseAt = audioPos

					pStates[player].IsHoldingNote[dir] = false

					event := NoteEvent{
						Time:  avgPos,
						Index: note.Index,
					}
					event.SetRelease()

					noteEvents = append(noteEvents, event)
				}

				// update key release stuff
				if pStates[player].IsHoldingKey[dir] {
					pStates[player].IsHoldingKey[dir] = false
					pStates[player].IsKeyJustReleased[dir] = true
					pStates[player].KeyReleasedAt[dir] = GlobalTimerNow()

					if pStates[player].IsHoldingBadKey[dir] {
						pStates[player].IsHoldingBadKey[dir] = false
						pStates[player].DidReleaseBadKey[dir] = true
					}
				}
			}
		}
	}

	return pStates, noteEvents, newNoteIndexStart
}

func GetKeyPressState(
	inputId InputGroupId,
	notes []FnfNote,
	noteIndexStart int,
	isPlayingAudio bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isBotPlay bool,
	hitWindow time.Duration,
) [2][NoteDirSize]bool {

	keyPressState := GetBotKeyPresseState(
		notes, noteIndexStart, isPlayingAudio, prevAudioPos, audioPos, isBotPlay, hitWindow)

	if !isBotPlay {
		for dir, keys := range NoteKeys {
			if AreKeysDown(inputId, keys...) {
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
	isPlayingAudio bool,
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
			if !note.IsHit && !note.IsSustain() {
				shouldHit := note.StartsAt <= audioPos && note.StartsAt >= prevAudioPos
				shouldHit = shouldHit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
				if shouldHit {
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
