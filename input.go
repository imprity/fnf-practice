package fnf

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"slices"
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

func MouseDelta() rl.Vector2 {
	screenRect := GetScreenRect()

	delta := rl.GetMouseDelta()

	delta.X = delta.X / screenRect.Width * SCREEN_WIDTH
	delta.Y = delta.Y / screenRect.Height * SCREEN_HEIGHT

	return delta
}

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

func AnyKeyPressed(id InputGroupId, except ...int32) (bool, int32) {
	if IsInputDisabled(id) {
		return false, rl.KeyNull
	}

	for _, key := range ListOfKeys() {
		if slices.Contains(except, key) {
			continue
		}

		if rl.IsKeyPressed(key) {
			return true, key
		}
	}

	return false, rl.KeyNull
}

func AnyKeyDown(id InputGroupId, except ...int32) (bool, int32) {
	if IsInputDisabled(id) {
		return false, rl.KeyNull
	}

	for _, key := range ListOfKeys() {
		if slices.Contains(except, key) {
			continue
		}

		if rl.IsKeyDown(key) {
			return true, key
		}
	}

	return false, rl.KeyNull
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
