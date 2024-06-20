package main

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

	ToggleDebugKey
	ToggleLogNoteEvent
	ReloadAssetsKey

	FnfBindingSize
)

var TheKM [FnfBindingSize]int32
var KeyHumanName [FnfBindingSize]string

func init() {
	// set default key bindings
	TheKM[NoteKeyLeft0] = rl.KeyLeft
	TheKM[NoteKeyLeft1] = rl.KeyA

	TheKM[NoteKeyDown0] = rl.KeyDown
	TheKM[NoteKeyDown1] = rl.KeyS

	TheKM[NoteKeyUp0] = rl.KeyUp
	TheKM[NoteKeyUp1] = rl.KeyW

	TheKM[NoteKeyRight0] = rl.KeyRight
	TheKM[NoteKeyRight1] = rl.KeyD

	TheKM[SelectKey] = rl.KeyEnter
	TheKM[PauseKey] = rl.KeySpace
	TheKM[EscapeKey] = rl.KeyEscape

	TheKM[NoteScrollUpKey] = rl.KeyPageUp
	TheKM[NoteScrollDownKey] = rl.KeyPageDown

	TheKM[AudioSpeedUpKey] = rl.KeyEqual
	TheKM[AudioSpeedDownKey] = rl.KeyMinus

	TheKM[SongResetKey] = rl.KeyR

	TheKM[SetBookMarkKey] = rl.KeyB
	TheKM[JumpToBookMarkKey] = rl.KeyBackspace

	TheKM[ZoomOutKey] = rl.KeyLeftBracket
	TheKM[ZoomInKey] = rl.KeyRightBracket

	TheKM[ToggleDebugKey] = rl.KeyF1
	TheKM[ToggleLogNoteEvent] = rl.KeyF2
	TheKM[ReloadAssetsKey] = rl.KeyF5

	for i := range FnfBindingSize {
		if TheKM[i] == 0 {
			ErrorLogger.Fatalf("default key binding for \"%v\" is omitted", i.String())
		}
	}

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

	KeyHumanName[ToggleDebugKey] = "toggle debug message"
	KeyHumanName[ToggleLogNoteEvent] = "togglee note event"
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
		ErrorLogger.Fatal("invalid direction %v", dir)
		return []int32{}
	}
}

func NoteDirAndIndexToBinding(dir NoteDir, index int) FnfBinding {
	if !(0 <= dir && dir < NoteDirSize) {
		ErrorLogger.Fatal("invalid dir \"%v\"", dir)
	}

	if !(0 <= index && index <= 1) {
		ErrorLogger.Fatal("invalid index \"%v\"", index)
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

	Volume float64

	DownScroll bool

	HitWindows [HitRatingSize]time.Duration

	LoadAudioDuringGamePlay bool
}

var DefaultOptions Options

var TheOptions Options

func init() {
	// set default option values
	DefaultOptions.TargetFPS = 60
	DefaultOptions.Volume = 1.0

	DefaultOptions.HitWindows[HitRatingBad] = time.Millisecond * 135
	DefaultOptions.HitWindows[HitRatingGood] = time.Millisecond * 90
	DefaultOptions.HitWindows[HitRatingSick] = time.Millisecond * 45

	DefaultOptions.DownScroll = false
	DefaultOptions.LoadAudioDuringGamePlay = false

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
