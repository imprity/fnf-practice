package main

import (
	"time"
	//rl "github.com/gen2brain/raylib-go/raylib"
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

var NoteDirStrs = [NoteDirSize]string{
	"left",
	"down",
	"up",
	"right",
}

type FnfNote struct {
	Player    int
	Direction NoteDir

	StartsAt time.Duration
	Duration time.Duration
	Index    int

	// variables that change during gameplay
	IsHit bool

	HoldReleaseAt time.Duration
}

func (n FnfNote) Equals(otherN FnfNote) bool {
	return n.Index == otherN.Index
}

func (n FnfNote) End() time.Duration {
	return n.StartsAt + n.Duration
}

func (n FnfNote) IsOverlapped(otherN FnfNote) bool {
	if n.Player != otherN.Player {
		return false
	}
	if n.Direction != otherN.Direction {
		return false
	}

	return AbsI(n.StartsAt-otherN.StartsAt) < time.Millisecond*2
}

func (n FnfNote) IsSustain() bool {
	// NOTE : I'm check if it's bigger than 1 millisecond rather than 0
	// because original fnf stores time in flating point number
	// and I'm scared of them
	return n.Duration >= time.Microsecond*500
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

func (n FnfNote) NotReachedHitWindow(audioPos, windowSize time.Duration) bool {
	return n.StartsAt > audioPos+windowSize/2
}

func (n FnfNote) StartPassedHitWindow(audioPos, windowSize time.Duration) bool {
	return n.StartsAt < audioPos-windowSize/2
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
	SongName    string
	Notes       []FnfNote
	NotesEndsAt time.Duration
	Speed       float64
	NeedsVoices bool
}

func (fs FnfSong) Copy() FnfSong {
	copy := FnfSong{}

	copy.Notes = make([]FnfNote, len(fs.Notes))

	for i := range len(fs.Notes) {
		copy.Notes[i] = fs.Notes[i]
	}

	copy.NotesEndsAt = fs.NotesEndsAt
	copy.Speed = fs.Speed
	copy.NeedsVoices = fs.NeedsVoices

	return copy
}

type FnfDifficulty int

const (
	DifficultyEasy FnfDifficulty = iota
	DifficultyNormal
	DifficultyHard
	DifficultySize
)

var DifficultyStrs [DifficultySize]string = [DifficultySize]string{
	"easy",
	"normal",
	"hard",
}

type FnfPathGroup struct {
	SongName string

	Songs     [DifficultySize]FnfSong
	SongPaths [DifficultySize]string
	HasSong   [DifficultySize]bool

	InstPath  string
	VoicePath string
}

type PathGroupCollection struct {
	PathGroups []FnfPathGroup
	BasePath   string
}

type FnfHitRating int

const (
	HitRatingBad FnfHitRating = iota
	HitRatingGood
	HitRatingSick
	HitRatingSize
)
