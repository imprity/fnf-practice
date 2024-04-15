package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
	"math"
)

var (
	NoteKeysLeft  = []int32{rl.KeyA,          rl.KeyLeft}
	NoteKeysDown  = []int32{rl.KeyS,          rl.KeyDown}
	NoteKeysUp    = []int32{rl.KeySemicolon,  rl.KeyUp}
	NotekeysRight = []int32{rl.KeyApostrophe, rl.KeyRight}
)

var NoteKeys = [NoteDirSize][]int32{
	NoteKeysLeft,
	NoteKeysDown,
	NoteKeysUp,
	NotekeysRight,
}

// ========================================
// key map
// ========================================
var SelectKey int32 = rl.KeyEnter
var PauseKey int32 = rl.KeySpace
var EscapeKey int32 = rl.KeyEscape

var AudioSpeedUpKey int32 = rl.KeyEqual
var AudioSpeedDownKey int32 = rl.KeyMinus

var SongResetKey int32 = rl.KeyR

var ToggleDebugKey int32 = rl.KeyF1
var ReloadAssetsKey int32 = rl.KeyF5

// TODO : remove these keys
var ZoomOutKey int32 = rl.KeyLeftBracket
var ZoomInKey int32 = rl.KeyRightBracket

var DifficultyUpKey int32 = rl.KeyW
var DifficultyDownKey int32 = rl.KeyQ

var ToggleBotPlayKey int32 = rl.KeyB
// ========================================
// end of key map
// ========================================

var IsInputDisabled bool

func AreKeysPressed(keys ...int32) bool{
	if IsInputDisabled{
		return false
	}

	for _, key := range keys{
		if rl.IsKeyPressed(key){
			return true
		}
	}

	return false
}

func AreKeysDown(keys ...int32) bool{
	if IsInputDisabled{
		return false
	}

	for _, key := range keys{
		if rl.IsKeyDown(key){
			return true
		}
	}

	return false
}

func AreKeysUp(keys ...int32) bool{
	if IsInputDisabled{
		return true
	}

	return !AreKeysDown(keys...)
}

func AreKeysReleased(keys ...int32) bool{
	if IsInputDisabled{
		// NOTE : retruning false because I think key being released
		// feels like something that would only happen if input is enabled
		return false
	}

	for _, key := range keys{
		if rl.IsKeyReleased(key){
			return true
		}
	}

	return false
}

var keyRepeatMap = make(map[int32]time.Duration)

func HandleKeyRepeat(firstRate, repeatRate time.Duration, keys ...int32) bool {
	minKey := int32(math.MaxInt32 )

	for _, key := range keys{
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
