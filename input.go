package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"sync"
	"time"
)

var isInputDisabled bool
var inputDisabledCheckMutex sync.Mutex

func IsInputDisabled() bool {
	inputDisabledCheckMutex.Lock()
	defer inputDisabledCheckMutex.Unlock()
	return isInputDisabled
}

func DisableInput() {
	inputDisabledCheckMutex.Lock()
	defer inputDisabledCheckMutex.Unlock()
	isInputDisabled = true
}

func EnableInput() {
	inputDisabledCheckMutex.Lock()
	defer inputDisabledCheckMutex.Unlock()
	isInputDisabled = false
}

func MouseX() float32 {
	screenRect := GetScreenRect()

	mx := f32(rl.GetMouseX())
	mx -= screenRect.X

	return mx / screenRect.Width * SCREEN_WIDTH
}

func MouseY() float32 {
	screenRect := GetScreenRect()

	my := f32(rl.GetMouseY())
	my -= screenRect.Y

	return my / screenRect.Height * SCREEN_HEIGHT
}

func MouseV() rl.Vector2 {
	return rl.Vector2{
		X: MouseX(),
		Y: MouseY(),
	}
}

// ========================================
// key map
// ========================================

var (
	NoteKeysLeft  = []int32{rl.KeyA, rl.KeyLeft}
	NoteKeysDown  = []int32{rl.KeyS, rl.KeyDown}
	NoteKeysUp    = []int32{rl.KeySemicolon, rl.KeyUp}
	NoteKeysRight = []int32{rl.KeyApostrophe, rl.KeyRight}
)

var NoteKeys = [NoteDirSize][]int32{
	NoteKeysLeft,
	NoteKeysDown,
	NoteKeysUp,
	NoteKeysRight,
}

var SelectKey int32 = rl.KeyEnter
var PauseKey int32 = rl.KeySpace
var EscapeKey int32 = rl.KeyEscape

var (
	NoteScrollUpKey   int32 = rl.KeyPageUp
	NoteScrollDownKey int32 = rl.KeyPageDown
)

var (
	AudioSpeedUpKey   int32 = rl.KeyEqual
	AudioSpeedDownKey int32 = rl.KeyMinus
)

var SongResetKey int32 = rl.KeyR

var (
	ToggleDebugKey     int32 = rl.KeyF1
	ToggleLogNoteEvent int32 = rl.KeyF2
	ReloadAssetsKey    int32 = rl.KeyF5
)

var (
	ZoomOutKey int32 = rl.KeyLeftBracket
	ZoomInKey  int32 = rl.KeyRightBracket
)

// ========================================
// end of key map
// ========================================

func AreKeysPressed(keys ...int32) bool {
	if IsInputDisabled() {
		return false
	}

	for _, key := range keys {
		if rl.IsKeyPressed(key) {
			return true
		}
	}

	return false
}

func AreKeysDown(keys ...int32) bool {
	if IsInputDisabled() {
		return false
	}

	for _, key := range keys {
		if rl.IsKeyDown(key) {
			return true
		}
	}

	return false
}

func AreKeysUp(keys ...int32) bool {
	if IsInputDisabled() {
		return true
	}

	return !AreKeysDown(keys...)
}

func AreKeysReleased(keys ...int32) bool {
	if IsInputDisabled() {
		// NOTE : retruning false because I think key being released
		// feels like something that would only happen if input is enabled
		return false
	}

	for _, key := range keys {
		if rl.IsKeyReleased(key) {
			return true
		}
	}

	return false
}

var keyRepeatMap = make(map[int32]time.Duration)

func HandleKeyRepeat(firstRate, repeatRate time.Duration, keys ...int32) bool {
	minKey := int32(math.MaxInt32)

	for _, key := range keys {
		minKey = min(key, minKey)
	}

	if !AreKeysDown(keys...) {
		keyRepeatMap[minKey] = 0
		return false
	}

	if AreKeysPressed(keys...) {
		keyRepeatMap[minKey] = GlobalTimerNow() + firstRate
		return true
	}

	time, ok := keyRepeatMap[minKey]

	if !ok {
		keyRepeatMap[minKey] = GlobalTimerNow() + firstRate
		return true
	} else {
		now := GlobalTimerNow()
		if now-time > repeatRate {
			keyRepeatMap[minKey] = now
			return true
		}
	}

	return false
}
