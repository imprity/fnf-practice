package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"slices"
)

var KeyNameMap = map[int32]string{
	rl.KeySpace:        "Space",
	rl.KeyEscape:       "Escape",
	rl.KeyEnter:        "Enter",
	rl.KeyTab:          "Tab",
	rl.KeyBackspace:    "Backspace",
	rl.KeyInsert:       "Insert",
	rl.KeyDelete:       "Delete",
	rl.KeyRight:        "Right",
	rl.KeyLeft:         "Left",
	rl.KeyDown:         "Down",
	rl.KeyUp:           "Up",
	rl.KeyPageUp:       "Page Up",
	rl.KeyPageDown:     "Page Down",
	rl.KeyHome:         "Home",
	rl.KeyEnd:          "End",
	rl.KeyCapsLock:     "Caps Lock",
	rl.KeyScrollLock:   "ScrollLock",
	rl.KeyNumLock:      "NumLock",
	rl.KeyPrintScreen:  "Print Screen",
	rl.KeyPause:        "Pause",
	rl.KeyF1:           "F1",
	rl.KeyF2:           "F2",
	rl.KeyF3:           "F3",
	rl.KeyF4:           "F4",
	rl.KeyF5:           "F5",
	rl.KeyF6:           "F6",
	rl.KeyF7:           "F7",
	rl.KeyF8:           "F8",
	rl.KeyF9:           "F9",
	rl.KeyF10:          "F10",
	rl.KeyF11:          "F11",
	rl.KeyF12:          "F12",
	rl.KeyLeftShift:    "Left Shift",
	rl.KeyLeftControl:  "Left Control",
	rl.KeyLeftAlt:      "Left Alt",
	rl.KeyLeftSuper:    "Left Super",
	rl.KeyRightShift:   "Right Shift",
	rl.KeyRightControl: "Right Control",
	rl.KeyRightAlt:     "Right Alt",
	rl.KeyRightSuper:   "Right Super",
	rl.KeyKbMenu:       "KbMenu",
	rl.KeyLeftBracket:  "[",
	rl.KeyBackSlash:    "\\",
	rl.KeyRightBracket: "]",
	rl.KeyGrave:        "`",

	// Keyboard Number Pad Keys
	rl.KeyKp0:        "Keypad 0",
	rl.KeyKp1:        "Keypad 1",
	rl.KeyKp2:        "Keypad 2",
	rl.KeyKp3:        "Keypad 3",
	rl.KeyKp4:        "Keypad 4",
	rl.KeyKp5:        "Keypad 5",
	rl.KeyKp6:        "Keypad 6",
	rl.KeyKp7:        "Keypad 7",
	rl.KeyKp8:        "Keypad 8",
	rl.KeyKp9:        "Keypad 9",
	rl.KeyKpDecimal:  "Keypad .",
	rl.KeyKpDivide:   "Keypad /",
	rl.KeyKpMultiply: "Keypad *",
	rl.KeyKpSubtract: "Keypad -",
	rl.KeyKpAdd:      "Keypad +",
	rl.KeyKpEnter:    "Keypad Enter",
	rl.KeyKpEqual:    "Keypad =",

	// Keyboard Alpha Numeric Keys
	rl.KeyApostrophe: "'",
	rl.KeyComma:      ",",
	rl.KeyMinus:      "-",
	rl.KeyPeriod:     ".",
	rl.KeySlash:      "/",
	rl.KeyZero:       "0",
	rl.KeyOne:        "1",
	rl.KeyTwo:        "2",
	rl.KeyThree:      "3",
	rl.KeyFour:       "4",
	rl.KeyFive:       "5",
	rl.KeySix:        "6",
	rl.KeySeven:      "7",
	rl.KeyEight:      "8",
	rl.KeyNine:       "9",
	rl.KeySemicolon:  ";",
	rl.KeyEqual:      "=",
	rl.KeyA:          "A",
	rl.KeyB:          "B",
	rl.KeyC:          "C",
	rl.KeyD:          "D",
	rl.KeyE:          "E",
	rl.KeyF:          "F",
	rl.KeyG:          "G",
	rl.KeyH:          "H",
	rl.KeyI:          "I",
	rl.KeyJ:          "J",
	rl.KeyK:          "K",
	rl.KeyL:          "L",
	rl.KeyM:          "M",
	rl.KeyN:          "N",
	rl.KeyO:          "O",
	rl.KeyP:          "P",
	rl.KeyQ:          "Q",
	rl.KeyR:          "R",
	rl.KeyS:          "S",
	rl.KeyT:          "T",
	rl.KeyU:          "U",
	rl.KeyV:          "V",
	rl.KeyW:          "W",
	rl.KeyX:          "X",
	rl.KeyY:          "Y",
	rl.KeyZ:          "Z",
}

func GetKeyName(key int32) string {
	keyName, ok := KeyNameMap[key]

	if !ok {
		return "?"
	}

	return keyName
}

var keyList []int32

func init() {
	for k := range KeyNameMap {
		keyList = append(keyList, k)
	}

	slices.Sort(keyList)
}

func ListOfKeys() []int32 {
	return keyList
}
