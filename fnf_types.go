package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"kitty"
)

type NoteDir int

const (
	NoteDirLeft NoteDir = iota
	NoteDirDown
	NoteDirUp
	NoteDirRight
	NoteDirSize

	NoteDirAny = -1
)

const (
	NoteKeyLeft  = ebiten.KeyA
	NoteKeyDown  = ebiten.KeyS
	NoteKeyUp    = ebiten.KeySemicolon
	NotekeyRight = ebiten.KeyQuote
)

var NoteKeys = [NoteDirSize]ebiten.Key{
	NoteKeyLeft,
	NoteKeyDown,
	NoteKeyUp,
	NotekeyRight,
}

type FnfNote struct {
	Player    int
	StartsAt  time.Duration
	Duration  time.Duration
	Direction NoteDir
	Index     int

	// variables that change during gameplay
	IsHit  bool
	IsMiss bool

	HoldReleaseAt time.Duration
}

func (n FnfNote) Equal(otherN FnfNote) bool {
	return n.Index == otherN.Index
}

func (n FnfNote) IsOverlapped(otherN FnfNote) bool {
	if n.Player != otherN.Player {
		return false
	}
	if n.Direction != otherN.Direction {
		return false
	}

	return kitty.AbsI(n.StartsAt-otherN.StartsAt) < time.Millisecond*2
}

func (n FnfNote) IsInWindow(audioPos, windowSize time.Duration) bool {
	start := audioPos - windowSize/2
	end := audioPos + windowSize/2
	return start <= n.StartsAt && n.StartsAt <= end
}

func (n FnfNote) IsAudioPositionInDuration(audioPos, windowSize time.Duration) bool {
	start := n.StartsAt - windowSize/2
	end := n.StartsAt + n.Duration + windowSize/2

	return start <= audioPos && audioPos <= end
}

const PlayerAny = -1
const IsHitAny = -1

type NoteFilter struct {
	Player    int
	IsHit     int
	Direction NoteDir
}

var NoteFilterAny = NoteFilter{
	Player:    PlayerAny,
	IsHit:     IsHitAny,
	Direction: NoteDirAny,
}

func NoteMatchesFilter(note FnfNote, filter NoteFilter) bool {
	if filter.Player >= 0 {
		if !(note.Player == filter.Player) {
			return false
		}
	}

	if filter.IsHit >= 0 {
		if !(filter.IsHit == BoolToInt(note.IsHit)) {
			return false
		}
	}

	if filter.Direction >= 0 {
		if !(filter.Direction == note.Direction) {
			return false
		}
	}

	return true
}

// TODO : This function can be faster, make it faster
func FindNextNote(notes []FnfNote, after time.Duration, filter NoteFilter) (FnfNote, bool) {
	for _, note := range notes {
		if note.StartsAt > after {
			if NoteMatchesFilter(note, filter) {
				return note, true
			}
		}
	}

	return FnfNote{}, false
}

// TODO : This function can be faster, make it faster
func FindPrevNoteIndex(notes []FnfNote, before time.Duration, filter NoteFilter) (FnfNote, bool) {
	for i := len(notes) - 1; i >= 0; i-- {
		note := notes[i]
		if note.StartsAt <= before {
			if NoteMatchesFilter(note, filter) {
				return note, true
			}
		}
	}

	return FnfNote{}, false
}

type FnfSong struct {
	Notes       []FnfNote
	NotesEndsAt time.Duration
	Speed       float64
}

func (fs FnfSong) Copy() FnfSong{
	copy := FnfSong{}

	copy.Notes = make([]FnfNote, len(fs.Notes))

	for i := range len(fs.Notes){
		copy.Notes[i] = fs.Notes[i]
	}

	copy.NotesEndsAt = fs.NotesEndsAt
	copy.Speed = fs.Speed

	return copy
}
