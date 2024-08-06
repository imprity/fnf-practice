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

func UpdateNotesAndStatesForHuman(
	song FnfSong,
	pState PlayerState,
	humanP FnfPlayerNo,
	wasKeyPressed [NoteDirSize]bool,
	isKeyPressed [NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	audioDuration time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	noteIndexStart int,
) (PlayerState, []NoteEvent) {
	notes := song.Notes

	var noteEvents []NoteEvent

	avgPos := (audioPos + prevAudioPos) / 2

	if isPlayingAudio {
		var isKeyJustPressed [NoteDirSize]bool
		var isKeyJustReleased [NoteDirSize]bool

		for dir := range NoteDirSize {
			if !wasKeyPressed[dir] && isKeyPressed[dir] {
				isKeyJustPressed[dir] = true
			}

			if wasKeyPressed[dir] && !isKeyPressed[dir] {
				isKeyJustReleased[dir] = true
			}
		}

		pState.IsKeyJustPressed = isKeyJustPressed
		pState.IsKeyJustReleased = isKeyJustReleased

		//clear note miss state
		for dir := range NoteDirSize {
			pState.DidMissNote[dir] = false
		}

		// declare convinience functions

		didHitNote := [NoteDirSize]bool{}
		hitNote := [NoteDirSize]FnfNote{}

		onNoteHit := func(note FnfNote, event *NoteEvent) {
			if !notes[note.Index].IsHit {
				event.SetFirstHit()
			} else {
				event.SetHit()
			}

			notes[note.Index].IsHit = true
			pState.IsHoldingBadKey[note.Direction] = false
			didHitNote[note.Direction] = true
			hitNote[note.Direction] = note
		}

		onNoteHold := func(note FnfNote, event *NoteEvent) {
			onNoteHit(note, event)

			pState.HoldingNotes[note.Direction] = append(pState.HoldingNotes[note.Direction], note)
		}

		onNoteMiss := func(note FnfNote, event *NoteEvent) {
			event.SetMiss()

			pState.DidMissNote[note.Direction] = true
			pState.NoteMissAt[note.Direction] = GlobalTimerNow()
		}

		// we check if user pressed any key
		// and if so mark all as bad hit (it will be overidden as not bad later)
		for dir := range NoteDirSize {
			if isKeyPressed[dir] && !pState.IsHoldingKey[dir] {
				pState.IsHoldingKey[dir] = true
				pState.KeyPressedAt[dir] = GlobalTimerNow()

				pState.IsHoldingBadKey[dir] = true
			} else if !isKeyPressed[dir] {
				if pState.IsHoldingKey[dir] {
					pState.KeyReleasedAt[dir] = GlobalTimerNow()

					if pState.IsHoldingBadKey[dir] {
						pState.DidReleaseBadKey[dir] = true
					} else {
						pState.DidReleaseBadKey[dir] = false
					}
				}

				pState.IsHoldingKey[dir] = false
				pState.IsHoldingBadKey[dir] = false
			}
		}

		// update any notes that were held but now no longer being held
		for dir := range NoteDirSize {
			if !isKeyPressed[dir] && pState.IsHoldingAnyNote(dir) {
				for _, note := range pState.HoldingNotes[dir] {
					notes[note.Index].HoldReleaseAt = audioPos

					event := NoteEvent{
						Time:  avgPos,
						Index: note.Index,
					}
					event.SetRelease()

					noteEvents = append(noteEvents, event)
				}

				pState.HoldingNotes[dir] = pState.HoldingNotes[dir][:0]
			}
		}

		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]

			if note.StartsAt > audioPos+hitWindow {
				break
			}

			if note.Player != humanP {
				continue
			}

			nd := note.Direction

			event := NoteEvent{
				Time:  avgPos,
				Index: note.Index,
			}

			//check if user hit note
			if isKeyJustPressed[nd] {
				var hit bool

				if note.IsSustain() {
					hit = note.IsAudioPositionInDuration(audioPos, hitWindow)
					hit = hit || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow)
				} else {
					hit = !note.IsHit
					hit = hit && note.IsStartInWindow(audioPos, hitWindow)
					hit = hit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
				}

				hitElse := (didHitNote[nd] && !hitNote[nd].IsSustain())
				hit = hit && (!hitElse || (hitElse && hitNote[nd].IsOverlapped(note)))

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
				if isKeyPressed[nd] {
					onNoteHold(note, &event)
				}
			}

			//check if user missed note
			if note.IsSustain() {
				missed := !pState.IsHoldingNote(note)
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
		}
	}

	if !isPlayingAudio && AbsI(audioDuration-audioPos) < time.Millisecond { // when song is done
		for dir := range NoteDirSize {
			// release notes that are being held
			if pState.IsHoldingAnyNote(dir) {
				for _, note := range pState.HoldingNotes[dir] {
					notes[note.Index].HoldReleaseAt = audioPos

					event := NoteEvent{
						Time:  avgPos,
						Index: note.Index,
					}
					event.SetRelease()

					noteEvents = append(noteEvents, event)
				}

				pState.HoldingNotes[dir] = pState.HoldingNotes[dir][:0]
			}

			// update key release stuff
			if pState.IsHoldingKey[dir] {
				pState.IsHoldingKey[dir] = false
				pState.IsKeyJustReleased[dir] = true
				pState.KeyReleasedAt[dir] = GlobalTimerNow()

				if pState.IsHoldingBadKey[dir] {
					pState.IsHoldingBadKey[dir] = false
					pState.DidReleaseBadKey[dir] = true
				}
			}
		}
	}

	return pState, noteEvents
}

func UpdateNotesAndStatesForBot(
	song FnfSong,
	pState PlayerState,
	botP FnfPlayerNo,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isPlayingAudio bool,
	hitWindow time.Duration,
	noteIndexStart int,
) (PlayerState, []NoteEvent) {
	notes := song.Notes

	var noteEvents []NoteEvent

	avgPos := (audioPos + prevAudioPos) / 2

	//clear note miss state
	for dir := range NoteDirSize {
		pState.DidMissNote[dir] = false
	}

	// release notes that are needs to be held
	for dir := range NoteDirSize {
		var newHoldingNotes []FnfNote

		for _, note := range pState.HoldingNotes[dir] {
			if note.End() < audioPos-time.Millisecond*10 {
				notes[note.Index].HoldReleaseAt = audioPos

				event := NoteEvent{
					Time:  avgPos,
					Index: note.Index,
				}
				event.SetRelease()

				noteEvents = append(noteEvents, event)
			} else {
				newHoldingNotes = append(newHoldingNotes, note)
			}
		}

		pState.HoldingNotes[dir] = newHoldingNotes
	}

	var pressKey [NoteDirSize]bool

	if isPlayingAudio {
		for ; noteIndexStart < len(notes); noteIndexStart++ {
			note := notes[noteIndexStart]

			if note.StartsAt > audioPos+hitWindow {
				break
			}

			if note.Player != botP {
				continue
			}

			event := NoteEvent{
				Time:  avgPos,
				Index: note.Index,
			}

			if note.IsSustain() {
				hit := note.IsAudioPositionInDuration(audioPos, 0) || SustainNoteTunneled(note, prevAudioPos, audioPos, 0)

				if hit {
					notes[note.Index].IsHit = true
					pressKey[note.Direction] = true
					if note.IsHit {
						event.SetHit()
					} else {
						event.SetFirstHit()
					}

					if !pState.IsHoldingNote(note) {
						pState.HoldingNotes[note.Direction] = append(pState.HoldingNotes[note.Direction], note)
					}
				}

			} else {
				if !note.IsHit {
					hit := note.StartsAt <= audioPos && note.IsStartInWindow(audioPos, hitWindow)
					hit = hit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)

					if hit {
						pressKey[note.Direction] = true
						notes[note.Index].IsHit = true
						event.SetFirstHit()
					}
				}
			}

			if !event.IsNone() {
				noteEvents = append(noteEvents, event)
			}
		}

		// update pstate
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			pState.IsKeyJustPressed[dir] = false
			pState.IsKeyJustReleased[dir] = false
			pState.DidReleaseBadKey[dir] = false
		}

		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			if pressKey[dir] {
				if !pState.IsHoldingKey[dir] {
					pState.IsKeyJustPressed[dir] = true
					pState.KeyPressedAt[dir] = GlobalTimerNow()
				}
			} else {
				if pState.IsHoldingKey[dir] {
					pState.IsKeyJustReleased[dir] = true
					pState.KeyReleasedAt[dir] = GlobalTimerNow()

					if pState.IsHoldingBadKey[dir] {
						pState.IsHoldingBadKey[dir] = false
						pState.DidReleaseBadKey[dir] = true
					}
				}
			}
		}

		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			pState.IsHoldingBadKey[dir] = false
		}

		pState.IsHoldingKey = pressKey
	}

	return pState, noteEvents
}

func CalculateNewNoteIndexStart(
	song FnfSong,
	audioPos time.Duration,
	hitWindow time.Duration,
	noteIndexStart int,
) int {
	notes := song.Notes
	hitWindow = max(hitWindow, time.Millisecond*5)

	newStart := noteIndexStart

	for ; newStart < len(notes); newStart++ {
		note := notes[newStart]

		if note.IsStartInWindow(audioPos, hitWindow) || note.IsAudioPositionInDuration(audioPos, hitWindow) {
			return newStart
		}
	}

	return noteIndexStart
}

func isNoteForBot(note FnfNote, isBotPlay bool, opponentMode bool) bool {
	if isBotPlay {
		return true
	}

	otherP := otherPlayer(opponentMode)

	return note.Player == otherP
}

func SimulateKeyPressForBot(
	song FnfSong,
	player FnfPlayerNo,
	wasKeyPressed [NoteDirSize]bool,
	prevAudioPos time.Duration,
	audioPos time.Duration,
	isBotPlay bool,
	opponentMode bool,
	isPlayingAudio bool,
	hitWindow time.Duration,
	noteIndexStart int,
) [NoteDirSize]bool {

	var keyPressed [NoteDirSize]bool

	const tinyWindow = time.Millisecond * 10

	notes := song.Notes

	for ; noteIndexStart < len(notes); noteIndexStart++ {
		note := notes[noteIndexStart]

		if note.Player == player {
			if note.IsSustain() {
				shouldHit := note.IsAudioPositionInDuration(audioPos, tinyWindow)
				shouldHit = shouldHit || SustainNoteTunneled(note, prevAudioPos, audioPos, hitWindow)
				shouldHit = shouldHit || (!note.IsHit &&
					note.StartsAt <= (audioPos+tinyWindow/2) &&
					note.IsAudioPositionInDuration(audioPos, hitWindow))

				if shouldHit {
					keyPressed[note.Direction] = true
				}
			} else {
				if !note.IsHit {
					shouldHit := note.StartsAt <= (audioPos+tinyWindow/2) && note.IsStartInWindow(audioPos, hitWindow)
					shouldHit = shouldHit || NoteStartTunneled(note, prevAudioPos, audioPos, hitWindow)
					if shouldHit {
						if wasKeyPressed[note.Direction] {
							keyPressed[note.Direction] = false
						} else {
							keyPressed[note.Direction] = true
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
