package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type KeyMap struct {
	// Should be noted that slices in NoteKeys should have two values.
	// No more, no less.
	// Only reason it's not a fixed size array is because of inconvenience when passing it to AreKeys fucntions
	NoteKeys [NoteDirSize][]int32

	SelectKey int32
	PauseKey  int32
	EscapeKey int32

	NoteScrollUpKey   int32
	NoteScrollDownKey int32

	AudioSpeedUpKey   int32
	AudioSpeedDownKey int32

	SongResetKey int32

	SetBookMarkKey    int32
	JumpToBookMarkKey int32

	ToggleDebugKey     int32
	ToggleLogNoteEvent int32
	ReloadAssetsKey    int32

	ZoomOutKey int32
	ZoomInKey  int32
}

var TheKM KeyMap = KeyMap{
	NoteKeys: [NoteDirSize][]int32{
		{rl.KeyA, rl.KeyLeft},
		{rl.KeyS, rl.KeyDown},
		{rl.KeySemicolon, rl.KeyUp},
		{rl.KeyApostrophe, rl.KeyRight},
	},

	SelectKey: rl.KeyEnter,
	PauseKey:  rl.KeySpace,
	EscapeKey: rl.KeyEscape,

	NoteScrollUpKey:   rl.KeyPageUp,
	NoteScrollDownKey: rl.KeyPageDown,

	AudioSpeedUpKey:   rl.KeyEqual,
	AudioSpeedDownKey: rl.KeyMinus,

	SongResetKey: rl.KeyR,

	SetBookMarkKey:    rl.KeyB,
	JumpToBookMarkKey: rl.KeyBackspace,

	ToggleDebugKey:     rl.KeyF1,
	ToggleLogNoteEvent: rl.KeyF2,
	ReloadAssetsKey:    rl.KeyF5,

	ZoomOutKey: rl.KeyLeftBracket,
	ZoomInKey:  rl.KeyRightBracket,
}

func NoteKeys(dir NoteDir) []int32 {
	return TheKM.NoteKeys[dir]
}

func NoteKeysArr() [NoteDirSize][]int32 {
	return TheKM.NoteKeys
}

type Options struct {
	TargetFPS               int32
	DownScroll              bool
	LoadAudioDuringGamePlay bool
}

var TheOptions Options = Options{
	TargetFPS:               60,
	DownScroll:              false,
	LoadAudioDuringGamePlay: false,
}
