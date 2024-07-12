package fnf

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
	rl.KeyLeftShift:    "L_Shift",
	rl.KeyLeftControl:  "L_Control",
	rl.KeyLeftAlt:      "L_Alt",
	rl.KeyLeftSuper:    "L_Super",
	rl.KeyRightShift:   "R_Shift",
	rl.KeyRightControl: "R_Control",
	rl.KeyRightAlt:     "R_Alt",
	rl.KeyRightSuper:   "R_Super",
	rl.KeyKbMenu:       "KbMenu",
	rl.KeyLeftBracket:  "[",
	rl.KeyBackSlash:    "\\",
	rl.KeyRightBracket: "]",
	rl.KeyGrave:        "`",

	// Keyboard Number Pad Keys
	rl.KeyKp0:        "#0",
	rl.KeyKp1:        "#1",
	rl.KeyKp2:        "#2",
	rl.KeyKp3:        "#3",
	rl.KeyKp4:        "#4",
	rl.KeyKp5:        "#5",
	rl.KeyKp6:        "#6",
	rl.KeyKp7:        "#7",
	rl.KeyKp8:        "#8",
	rl.KeyKp9:        "#9",
	rl.KeyKpDecimal:  "#.",
	rl.KeyKpDivide:   "#/",
	rl.KeyKpMultiply: "#*",
	rl.KeyKpSubtract: "#-",
	rl.KeyKpAdd:      "#+",
	rl.KeyKpEnter:    "#Enter",
	rl.KeyKpEqual:    "#=",

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
