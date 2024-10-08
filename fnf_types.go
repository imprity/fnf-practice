package fnf

import (
	"time"
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

type FnfPlayerNo int

const FnfPlayerSize FnfPlayerNo = 2

type FnfNote struct {
	Player    FnfPlayerNo
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

func (n FnfNote) IsStartInWindow(audioPos, windowSize time.Duration) bool {
	start := audioPos - windowSize/2
	end := audioPos + windowSize/2
	return start <= n.StartsAt && n.StartsAt <= end
}

func (n FnfNote) IsAudioPositionInDuration(audioPos, windowSize time.Duration) bool {
	start := n.StartsAt - windowSize/2
	end := n.StartsAt + n.Duration + windowSize/2

	return start <= audioPos && audioPos <= end
}

func (n FnfNote) NotReachedWindow(audioPos, windowSize time.Duration) bool {
	return n.StartsAt > audioPos+windowSize/2
}

func (n FnfNote) PassedWindow(audioPos, windowSize time.Duration) bool {
	return n.End() < audioPos-windowSize/2
}

func (n FnfNote) StartPassedWindow(audioPos, windowSize time.Duration) bool {
	return n.StartsAt < audioPos-windowSize/2
}

type FnfBpm struct {
	StartsAt time.Duration
	Bpm      float64
}

const DefaultBpm = 100

type FnfSong struct {
	SongName    string
	Notes       []FnfNote
	NotesEndsAt time.Duration
	Speed       float64
	NeedsVoices bool
	Bpms        []FnfBpm
}

func (fs FnfSong) Copy() FnfSong {
	copy := FnfSong{}

	copy.Notes = make([]FnfNote, len(fs.Notes))
	for i := range len(fs.Notes) {
		copy.Notes[i] = fs.Notes[i]
	}

	copy.Bpms = make([]FnfBpm, len(fs.Bpms))
	for i := range len(fs.Bpms) {
		copy.Bpms[i] = fs.Bpms[i]
	}

	copy.NotesEndsAt = fs.NotesEndsAt
	copy.Speed = fs.Speed
	copy.NeedsVoices = fs.NeedsVoices

	return copy
}

// Offset the song to a offset
// As name implies, it modifies the notes and bpm changes
// So use the clone if you want to keep the original values intact
func (fs *FnfSong) OffsetNotesAndBpmChanges(offset time.Duration) {
	for i := 0; i < len(fs.Notes); i++ {
		fs.Notes[i].StartsAt += offset
	}

	// we ignore the first bpm since it's were we store the first bpm
	for i := 1; i < len(fs.Bpms); i++ {
		fs.Bpms[i].StartsAt += offset
	}
}

func (fs FnfSong) GetBpmAt(at time.Duration) float64 {
	if at >= fs.Bpms[len(fs.Bpms)-1].StartsAt {
		return fs.Bpms[len(fs.Bpms)-1].Bpm
	}

	var bpm float64

	for i := 0; i+1 < len(fs.Bpms); i++ {
		bpm = fs.Bpms[i].Bpm

		if at < fs.Bpms[i+1].StartsAt {
			break
		}
	}

	return bpm
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

type FnfPathGroupId int64

type FnfPathGroup struct {
	SongName string

	SongPaths [DifficultySize]string
	HasSong   [DifficultySize]bool

	InstPath  string
	VoicePath string

	id FnfPathGroupId
}

var fnfPathGroupIdGenerator IdGenerator[FnfPathGroupId]

func NewFnfPathGroupId() FnfPathGroupId {
	return fnfPathGroupIdGenerator.NewId()
}

func (fp *FnfPathGroup) Id() FnfPathGroupId {
	return fp.id
}

type PathGroupCollectionId int64

type PathGroupCollection struct {
	PathGroups []FnfPathGroup
	BasePath   string

	id PathGroupCollectionId
}

var pathGroupCollectionIdGenerator IdGenerator[PathGroupCollectionId]

func NewPathGroupCollectionId() PathGroupCollectionId {
	return pathGroupCollectionIdGenerator.NewId()
}

func (pg *PathGroupCollection) Id() PathGroupCollectionId {
	return pg.id
}

type FnfHitRating int

const (
	HitRatingBad FnfHitRating = iota
	HitRatingGood
	HitRatingSick
	HitRatingSize
)

var RatingStrs [HitRatingSize]string = [HitRatingSize]string{
	"bad",
	"good",
	"sick",
}

func GetHitRating(noteStartsAt time.Duration, noteHitAt time.Duration) FnfHitRating {
	t := AbsI(noteStartsAt - noteHitAt)

	if t <= TheOptions.HitWindows[HitRatingSick] {
		return HitRatingSick
	} else if t <= TheOptions.HitWindows[HitRatingGood] {
		return HitRatingGood
	} else {
		return HitRatingBad
	}
}
