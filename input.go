package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"time"
)

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

/*
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

var SetBookMarkKey int32 = rl.KeyB
var JumpToBookMarkKey int32 = rl.KeyBackspace

var (
	ToggleDebugKey     int32 = rl.KeyF1
	ToggleLogNoteEvent int32 = rl.KeyF2
	ReloadAssetsKey    int32 = rl.KeyF5
)

var (
	ZoomOutKey int32 = rl.KeyLeftBracket
	ZoomInKey  int32 = rl.KeyRightBracket
)
*/

// ========================================
// end of key map
// ========================================

type InputGroupId int64

var (
	isInputGroupEnabled map[InputGroupId]bool = make(map[InputGroupId]bool)

	inputGroupSoloEnabled InputGroupId
)

var inputAllDisabled bool

var inputGroupIdGenerator IdGenerator[InputGroupId]

func NewInputGroupId() InputGroupId {
	id := inputGroupIdGenerator.NewId()

	isInputGroupEnabled[id] = true

	return id
}

func IsInputDisabled(id InputGroupId) bool {
	if inputAllDisabled {
		return true
	}

	if inputGroupSoloEnabled <= 0 {
		// we don't have to check if id is in map
		// because map returns false when there's no key
		return !isInputGroupEnabled[id]
	} else {
		return inputGroupSoloEnabled != id
	}
}

func IsInputEnabled(id InputGroupId) bool {
	return !IsInputDisabled(id)
}

func DisableInput(id InputGroupId) {
	isInputGroupEnabled[id] = false
}

func EnableInput(id InputGroupId) {
	isInputGroupEnabled[id] = true
}

func DisableInputGlobal() {
	inputAllDisabled = true
}

func ClearGlobalInputDisable() {
	inputAllDisabled = false
}

func SetSoloInput(id InputGroupId) {
	inputGroupSoloEnabled = id
}

func IsInputSoloEnabled(id InputGroupId) bool {
	return id == inputGroupSoloEnabled
}

func ClearSoloInput() {
	inputGroupSoloEnabled = 0
}

func AreKeysDown(id InputGroupId, keys ...int32) bool {
	if IsInputDisabled(id) {
		return false
	}

	for _, key := range keys {
		if rl.IsKeyDown(key) {
			return true
		}
	}

	return false
}

func AreKeysPressed(id InputGroupId, keys ...int32) bool {
	if IsInputDisabled(id) {
		return false
	}

	for _, key := range keys {
		if rl.IsKeyPressed(key) {
			return true
		}
	}

	return false
}

func AreKeysUp(id InputGroupId, keys ...int32) bool {
	if IsInputDisabled(id) {
		return true
	}

	return !AreKeysDown(id, keys...)
}

func AreKeysReleased(id InputGroupId, keys ...int32) bool {
	if IsInputDisabled(id) {
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

func HandleKeyRepeat(
	id InputGroupId,
	firstRate, repeatRate time.Duration,
	keys ...int32) bool {

	minKey := int32(math.MaxInt32)

	for _, key := range keys {
		minKey = min(key, minKey)
	}

	if !AreKeysDown(id, keys...) {
		keyRepeatMap[minKey] = 0
		return false
	}

	if AreKeysPressed(id, keys...) {
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

func IsMouseButtonDown(id InputGroupId, button int32) bool {
	if IsInputDisabled(id) {
		return false
	}

	return rl.IsMouseButtonDown(button)
}

func IsMouseButtonPressed(id InputGroupId, button int32) bool {
	if IsInputDisabled(id) {
		return false
	}

	return rl.IsMouseButtonPressed(button)
}

func IsMouseButtonUp(id InputGroupId, button int32) bool {
	if IsInputDisabled(id) {
		return true
	}

	return rl.IsMouseButtonUp(button)
}

func IsMouseButtonReleased(id InputGroupId, button int32) bool {
	if IsInputDisabled(id) {
		// NOTE : retruning false because I think button being released
		// feels like something that would only happen if input is enabled
		return false
	}

	return rl.IsMouseButtonReleased(button)
}
