package fnf

import (
	"time"
)

type PlayerState struct {
	HoldingNotes [NoteDirSize][]FnfNote

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

func (ps *PlayerState) IsHoldingAnyNote(dir NoteDir) bool {
	return len(ps.HoldingNotes[dir]) > 0
}

func (ps *PlayerState) IsHoldingNote(note FnfNote) bool {
	for _, holdingNote := range ps.HoldingNotes[note.Direction] {
		if holdingNote.Equals(note) {
			return true
		}
	}
	return false
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
	song FnfSong,
	pStates [FnfPlayerSize]PlayerState,
	wasKeyPressed [FnfPlayerSize][NoteDirSize]bool,
	isKeyPressed [FnfPlayerSize][NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	audioDuration time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	botPlay bool,
	noteIndexStart int,
) ([FnfPlayerSize]PlayerState, []NoteEvent, int) {
	notes := song.Notes

	newNoteIndexStart := noteIndexStart

	var noteEvents []NoteEvent

	avgPos := (audioPos + prevAudioPos) / 2

	if isPlayingAudio {
		var isKeyJustPressed [FnfPlayerSize][NoteDirSize]bool
		var isKeyJustReleased [FnfPlayerSize][NoteDirSize]bool

		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			for dir := range NoteDirSize {
				if !wasKeyPressed[player][dir] && isKeyPressed[player][dir] {
					isKeyJustPressed[player][dir] = true
				}

				if wasKeyPressed[player][dir] && !isKeyPressed[player][dir] {
					isKeyJustReleased[player][dir] = true
				}
			}
		}

		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			pStates[player].IsKeyJustPressed = isKeyJustPressed[player]
			pStates[player].IsKeyJustReleased = isKeyJustReleased[player]
		}

		//clear note miss state
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			for dir := range NoteDirSize {
				pStates[player].DidMissNote[dir] = false
			}
		}

		// declare convinience functions

		didHitNote := [FnfPlayerSize][NoteDirSize]bool{}
		hitNote := [FnfPlayerSize][NoteDirSize]FnfNote{}

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

			pStates[note.Player].HoldingNotes[note.Direction] = append(pStates[note.Player].HoldingNotes[note.Direction], note)
		}

		onNoteMiss := func(note FnfNote, event *NoteEvent) {
			event.SetMiss()

			pStates[note.Player].DidMissNote[note.Direction] = true
			pStates[note.Player].NoteMissAt[note.Direction] = GlobalTimerNow()
		}

		// we check if user pressed any key
		// and if so mark all as bad hit (it will be overidden as not bad later)
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
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
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			for dir := range NoteDirSize {
				if !isKeyPressed[player][dir] && pStates[player].IsHoldingAnyNote(dir) {
					for _, note := range pStates[player].HoldingNotes[dir] {
						notes[note.Index].HoldReleaseAt = audioPos

						event := NoteEvent{
							Time:  avgPos,
							Index: note.Index,
						}
						event.SetRelease()

						noteEvents = append(noteEvents, event)
					}

					pStates[player].HoldingNotes[dir] = pStates[player].HoldingNotes[dir][:0]
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
				missed := !pStates[np].IsHoldingNote(note)
				missed = missed && note.StartPassedWindow(audioPos, hitWindow)
				missed = missed && note.IsAudioPositionInDuration(audioPos, hitWindow)

				bpm := song.GetBpmAt(note.StartsAt)
				stepTime := StepsToTime(1, bpm)

				if note.IsHit {
					missed = missed && note.End()-note.HoldReleaseAt > stepTime
				}

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
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			for dir := range NoteDirSize {
				// release notes that are being held
				if pStates[player].IsHoldingAnyNote(dir) {
					for _, note := range pStates[player].HoldingNotes[dir] {
						notes[note.Index].HoldReleaseAt = audioPos

						event := NoteEvent{
							Time:  avgPos,
							Index: note.Index,
						}
						event.SetRelease()

						noteEvents = append(noteEvents, event)
					}

					pStates[player].HoldingNotes[dir] = pStates[player].HoldingNotes[dir][:0]
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
	wasKeyPressed [FnfPlayerSize][NoteDirSize]bool,
	inputId InputGroupId,
	notes []FnfNote,
	noteIndexStart int,
	isPlayingAudio bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isBotPlay bool,
	opponentMode bool,
	hitWindow time.Duration,
) [FnfPlayerSize][NoteDirSize]bool {

	keyPressState := GetBotKeyPresseState(
		wasKeyPressed,
		notes,
		noteIndexStart,
		isPlayingAudio,
		prevAudioPos,
		audioPos,
		isBotPlay,
		opponentMode,
		hitWindow,
	)

	if !isBotPlay {
		for dir, keys := range NoteKeysArr() {
			if AreKeysDown(inputId, keys...) {
				keyPressState[mainPlayer(opponentMode)][dir] = true
			}
		}
	}

	return keyPressState
}

func isNoteForBot(note FnfNote, isBotPlay bool, opponentMode bool) bool {
	if isBotPlay {
		return true
	}

	otherP := otherPlayer(opponentMode)

	return note.Player == otherP
}

func GetBotKeyPresseState(
	wasKeyPressed [FnfPlayerSize][NoteDirSize]bool,
	notes []FnfNote,
	noteIndexStart int,
	isPlayingAudio bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isBotPlay bool,
	opponentMode bool,
	hitWindow time.Duration,
) [FnfPlayerSize][NoteDirSize]bool {

	var keyPressed [FnfPlayerSize][NoteDirSize]bool

	const tinyWindow = time.Millisecond * 10

	for ; noteIndexStart < len(notes); noteIndexStart++ {
		note := notes[noteIndexStart]

		if isNoteForBot(note, isBotPlay, opponentMode) {
			if note.IsSustain() {
				shouldHit := note.IsAudioPositionInDuration(audioPos, tinyWindow)
				shouldHit = shouldHit || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow)
				shouldHit = shouldHit || (!note.IsHit &&
					note.StartsAt <= (audioPos+tinyWindow/2) &&
					note.IsAudioPositionInDuration(audioPos, HitWindow()))

				if shouldHit {
					keyPressed[note.Player][note.Direction] = true
				}
			} else {
				if !note.IsHit {
					shouldHit := note.StartsAt <= (audioPos+tinyWindow/2) && note.IsStartInWindow(audioPos, HitWindow())
					shouldHit = shouldHit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
					if shouldHit {
						if wasKeyPressed[note.Player][note.Direction] {
							keyPressed[note.Player][note.Direction] = false
						} else {
							keyPressed[note.Player][note.Direction] = true
						}
					}
				}
			}
		}

		if note.StartsAt > audioPos+tinyWindow {
			break
		}
	}

	return keyPressed
}
