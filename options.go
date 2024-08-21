package fnf

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

type FnfBinding int

const (
	NoteKeyLeft0 FnfBinding = iota
	NoteKeyLeft1

	NoteKeyDown0
	NoteKeyDown1

	NoteKeyUp0
	NoteKeyUp1

	NoteKeyRight0
	NoteKeyRight1

	SelectKey
	PauseKey
	EscapeKey

	NoteScrollUpKey
	NoteScrollDownKey

	AudioSpeedUpKey
	AudioSpeedDownKey

	SongResetKey

	SetBookMarkKey
	JumpToBookMarkKey

	ZoomOutKey
	ZoomInKey

	ToggleDebugMsg
	ToggleLogNoteEvent
	ToggleDebugGraphics
	ReloadAssetsKey

	FnfBindingSize
)

var (
	// default key map
	DefaultKM [FnfBindingSize]int32
	// key bindings game will use
	TheKM [FnfBindingSize]int32
)

var KeyHumanName [FnfBindingSize]string

func init() {
	// set default key bindings
	DefaultKM[NoteKeyLeft0] = rl.KeyLeft
	DefaultKM[NoteKeyLeft1] = rl.KeyA

	DefaultKM[NoteKeyDown0] = rl.KeyDown
	DefaultKM[NoteKeyDown1] = rl.KeyS

	DefaultKM[NoteKeyUp0] = rl.KeyUp
	DefaultKM[NoteKeyUp1] = rl.KeyW

	DefaultKM[NoteKeyRight0] = rl.KeyRight
	DefaultKM[NoteKeyRight1] = rl.KeyD

	DefaultKM[SelectKey] = rl.KeyEnter
	DefaultKM[PauseKey] = rl.KeySpace
	DefaultKM[EscapeKey] = rl.KeyEscape

	DefaultKM[NoteScrollUpKey] = rl.KeyPageUp
	DefaultKM[NoteScrollDownKey] = rl.KeyPageDown

	DefaultKM[AudioSpeedUpKey] = rl.KeyEqual
	DefaultKM[AudioSpeedDownKey] = rl.KeyMinus

	DefaultKM[SongResetKey] = rl.KeyR

	DefaultKM[SetBookMarkKey] = rl.KeyB
	DefaultKM[JumpToBookMarkKey] = rl.KeyBackspace

	DefaultKM[ZoomOutKey] = rl.KeyLeftBracket
	DefaultKM[ZoomInKey] = rl.KeyRightBracket

	DefaultKM[ToggleDebugMsg] = rl.KeyF1
	DefaultKM[ToggleLogNoteEvent] = rl.KeyF2
	DefaultKM[ToggleDebugGraphics] = rl.KeyF3
	DefaultKM[ReloadAssetsKey] = rl.KeyF5

	for i := range FnfBindingSize {
		if DefaultKM[i] == 0 {
			ErrorLogger.Fatalf("default key binding for \"%v\" is omitted", i.String())
		}
	}

	// set TheKM to DefaultKM
	TheKM = DefaultKM

	// assign names for humans
	KeyHumanName[NoteKeyLeft0] = "left 0"
	KeyHumanName[NoteKeyLeft1] = "left 1"

	KeyHumanName[NoteKeyDown0] = "down 0"
	KeyHumanName[NoteKeyDown1] = "down 1"

	KeyHumanName[NoteKeyUp0] = "up 0"
	KeyHumanName[NoteKeyUp1] = "up 1"

	KeyHumanName[NoteKeyRight0] = "right 0"
	KeyHumanName[NoteKeyRight1] = "right 1"

	KeyHumanName[SelectKey] = "select"
	KeyHumanName[PauseKey] = "pause"
	KeyHumanName[EscapeKey] = "escape"

	KeyHumanName[NoteScrollUpKey] = "scroll up"
	KeyHumanName[NoteScrollDownKey] = "scroll down"

	KeyHumanName[AudioSpeedUpKey] = "speed up"
	KeyHumanName[AudioSpeedDownKey] = "speed down"

	KeyHumanName[SongResetKey] = "reset"

	KeyHumanName[SetBookMarkKey] = "bookmark"
	KeyHumanName[JumpToBookMarkKey] = "jump to bookmark"

	KeyHumanName[ZoomOutKey] = "note spacing up"
	KeyHumanName[ZoomInKey] = "note spacing down"

	KeyHumanName[ToggleDebugMsg] = "toggle debug message"
	KeyHumanName[ToggleLogNoteEvent] = "toggle note event"
	KeyHumanName[ToggleDebugGraphics] = "toggle debug graphics"
	KeyHumanName[ReloadAssetsKey] = "reload assets"

	for i := range FnfBindingSize {
		if KeyHumanName[i] == "" {
			ErrorLogger.Fatalf("human name for \"%v\" is omitted", i.String())
		}
	}
}

func NoteKeys(dir NoteDir) []int32 {
	switch dir {
	case NoteDirLeft:
		return []int32{TheKM[NoteKeyLeft0], TheKM[NoteKeyLeft1]}
	case NoteDirDown:
		return []int32{TheKM[NoteKeyDown0], TheKM[NoteKeyDown1]}
	case NoteDirUp:
		return []int32{TheKM[NoteKeyUp0], TheKM[NoteKeyUp1]}
	case NoteDirRight:
		return []int32{TheKM[NoteKeyRight0], TheKM[NoteKeyRight1]}
	default:
		ErrorLogger.Fatalf("invalid direction %v", dir)
		return []int32{}
	}
}

func NoteDirAndIndexToBinding(dir NoteDir, index int) FnfBinding {
	if !(0 <= dir && dir < NoteDirSize) {
		ErrorLogger.Fatalf("invalid dir \"%v\"", dir)
	}

	if !(0 <= index && index <= 1) {
		ErrorLogger.Fatalf("invalid index \"%v\"", index)
	}

	switch dir {
	case NoteDirLeft:
		if index == 0 {
			return NoteKeyLeft0
		} else {
			return NoteKeyLeft1
		}
	case NoteDirDown:
		if index == 0 {
			return NoteKeyDown0
		} else {
			return NoteKeyDown1
		}
	case NoteDirUp:
		if index == 0 {
			return NoteKeyUp0
		} else {
			return NoteKeyUp1
		}
	case NoteDirRight:
		if index == 0 {
			return NoteKeyRight0
		} else {
			return NoteKeyRight1
		}
	}

	ErrorLogger.Fatal("UNREACHABLE")
	return 0
}

func SetNoteKeys(dir NoteDir, index int, key int32) {
	TheKM[NoteDirAndIndexToBinding(dir, index)] = key
}

func NoteKeysArr() [NoteDirSize][]int32 {
	return [NoteDirSize][]int32{
		{TheKM[NoteKeyLeft0], TheKM[NoteKeyLeft1]},
		{TheKM[NoteKeyDown0], TheKM[NoteKeyDown1]},
		{TheKM[NoteKeyUp0], TheKM[NoteKeyUp1]},
		{TheKM[NoteKeyRight0], TheKM[NoteKeyRight1]},
	}
}

type Options struct {
	TargetFPS int32

	DisplayFPS bool

	Volume float64

	DownScroll bool

	HitWindows [HitRatingSize]time.Duration

	LoadAudioDuringGamePlay bool

	GhostTapping bool

	MiddleScroll bool

	HitSoundVolume float64

	NoteSplash bool

	AudioOffset time.Duration
}

const AudioOffsetMax time.Duration = 500 * time.Millisecond

var DefaultOptions Options

var TheOptions Options

func init() {
	// set default option values
	DefaultOptions.TargetFPS = 60
	DefaultOptions.DisplayFPS = true

	DefaultOptions.Volume = 1.0

	DefaultOptions.HitWindows[HitRatingBad] = time.Millisecond * 135
	DefaultOptions.HitWindows[HitRatingGood] = time.Millisecond * 90
	DefaultOptions.HitWindows[HitRatingSick] = time.Millisecond * 45

	DefaultOptions.DownScroll = false

	DefaultOptions.LoadAudioDuringGamePlay = false

	DefaultOptions.GhostTapping = false

	DefaultOptions.MiddleScroll = false

	DefaultOptions.HitSoundVolume = 0

	DefaultOptions.NoteSplash = true

	DefaultOptions.AudioOffset = 0

	// set TheOptions to DefaultOptions
	TheOptions = DefaultOptions
}

func HitWindow() time.Duration {
	return max(
		TheOptions.HitWindows[HitRatingBad],
		TheOptions.HitWindows[HitRatingGood],
		TheOptions.HitWindows[HitRatingSick],
	) * 2
}
