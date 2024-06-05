package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type OptionsScreen struct {
	Menu *MenuDrawer

	FpsItemId               MenuItemId
	DownScrollItemId        MenuItemId
	LoadAudioDuringGpItemId MenuItemId

	InputId InputGroupId
}

func NewOptionsScreen() *OptionsScreen {
	op := new(OptionsScreen)

	op.InputId = NewInputGroupId()

	op.Menu = NewMenuDrawer()

	optionsDeco := NewMenuItem()
	optionsDeco.Name = "Options"
	optionsDeco.Type = MenuItemDeco
	optionsDeco.Color = Color255(0xE3, 0x9C, 0x02, 0xFF)
	optionsDeco.FadeIfUnselected = false
	optionsDeco.SizeRegular = MenuItemDefaults.SizeRegular * 1.7
	optionsDeco.SizeSelected = MenuItemDefaults.SizeSelected * 1.7
	op.Menu.AddItems(optionsDeco)

	backItem := NewMenuItem()
	backItem.Name = "Back To Menu"
	backItem.Type = MenuItemTrigger
	backItem.TriggerCallback = func() {
		// TODO : options screen doesn't save settings
		// if it's quit by user
		err := SaveSettings()
		if err != nil {
			ErrorLogger.Println(err)
			DisplayAlert("failed to save settings")
		}
		ShowTransition(BlackPixel, func() {
			SetNextScreen(TheSelectScreen)
			HideTransition()
		})
	}
	op.Menu.AddItems(backItem)

	fpsItem := NewMenuItem()
	fpsItem.Name = "Target FPS"
	fpsItem.Type = MenuItemNumber
	fpsItem.NValue = float32(TheOptions.TargetFPS)
	fpsItem.NValueMin = 30
	fpsItem.NValueMax = 500
	fpsItem.NValueInterval = 10
	fpsItem.NValueFmtString = "%1.f"
	fpsItem.NumberCallback = func(nValue float32) {
		TheOptions.TargetFPS = int32(nValue)
	}
	op.FpsItemId = fpsItem.Id
	op.Menu.AddItems(fpsItem)

	downScrollItem := NewMenuItem()
	downScrollItem.Name = "Down Scroll"
	downScrollItem.Type = MenuItemToggle
	downScrollItem.ToggleCallback = func(bValue bool) {
		TheOptions.DownScroll = bValue
	}
	op.DownScrollItemId = downScrollItem.Id
	op.Menu.AddItems(downScrollItem)

	loadAudioDuringGpItem := NewMenuItem()
	loadAudioDuringGpItem.Name = "Load Audio During Game Play"
	loadAudioDuringGpItem.Type = MenuItemToggle
	loadAudioDuringGpItem.ToggleCallback = func(bValue bool) {
		TheOptions.LoadAudioDuringGamePlay = bValue
	}
	loadAudioDuringGpItem.SizeSelected = 75
	loadAudioDuringGpItem.SelectedLeftMargin = 5
	op.LoadAudioDuringGpItemId = loadAudioDuringGpItem.Id
	op.Menu.AddItems(loadAudioDuringGpItem)

	// create key control options
	{
		deco := NewMenuItem()
		deco.Name = "Controls"
		deco.Type = MenuItemDeco
		deco.SizeRegular = MenuItemDefaults.SizeRegular * 1.4
		deco.SizeSelected = MenuItemDefaults.SizeSelected * 1.4
		deco.Color = Color255(0xFC, 0x9F, 0x7C, 0xFF)
		deco.FadeIfUnselected = false
		op.Menu.AddItems(deco)

		createKeyOp := func(name string, keys []int32) *MenuItem {
			keyItem := NewMenuItem()
			keyItem.Name = name
			keyItem.Type = MenuItemKey
			keyItem.SizeRegular = 65
			keyItem.SizeSelected = 70
			keyItem.BottomMargin = 15
			keyItem.NameMinWidth = 270
			keyItem.SelectedLeftMargin = 5
			keyItem.AddKeys(keys...)

			op.Menu.AddItems(keyItem)

			return keyItem
		}

		// direction key
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			item := createKeyOp(
				// NOTE : I know Title is deprecated but I don't care
				fmt.Sprintf("%v :", strings.Title(NoteDirStrs[dir])),
				NoteKeys(dir),
			)

			item.KeyCallback = func(index int, prevKey int32, newKey int32) {
				if prevKey == newKey {
					return
				}

				// check for duplicate
				isDuplicate := false
				var duplicateOf FnfKey

				for binding := FnfKey(0); binding < FnfKeySize; binding++ {
					if TheKM[binding] == newKey {
						isDuplicate = true
						duplicateOf = binding
					}
				}

				if isDuplicate {
					DisplayOptionsPopup(
						fmt.Sprintf("Sorry\n\"%v\" key is already assigned to \"%v\"", GetKeyName(newKey), KeyHumanName[duplicateOf]),
						[]string{}, nil)
				} else {
					SetNoteKeys(dir, index, newKey)
					item.KeyValues[index] = newKey
				}
			}
		}

		debugKeys := []FnfKey{
			ToggleDebugKey,
			ToggleLogNoteEvent,
			ReloadAssetsKey,
		}

		for key := SelectKey; key < FnfKeySize; key++ {
			// leave out debug keys from menu
			if slices.Contains(debugKeys, key) {
				continue
			}

			// NOTE : I know Title is deprecated but I don't care
			name := fmt.Sprintf("%v :", strings.Title(KeyHumanName[key]))
			item := createKeyOp(name, []int32{TheKM[key]})

			item.KeyCallback = func(index int, prevKey int32, newKey int32) {
				if prevKey == newKey {
					return
				}

				// check for duplicate
				isDuplicate := false
				var duplicateOf FnfKey

				for binding := FnfKey(0); binding < FnfKeySize; binding++ {
					if TheKM[binding] == newKey {
						isDuplicate = true
						duplicateOf = binding
					}
				}

				if isDuplicate {
					DisplayOptionsPopup(
						fmt.Sprintf("Sorry\n\"%v\" key is already assigned to \"%v\"", GetKeyName(newKey), KeyHumanName[duplicateOf]),
						[]string{}, nil)
				} else {
					TheKM[key] = newKey
					item.KeyValues[index] = newKey
				}
			}

			switch key {
			case AudioSpeedUpKey, AudioSpeedDownKey:
				item.NameMinWidth = 290
			case ZoomInKey, ZoomOutKey:
				item.NameMinWidth = 460
			}
		}
	}

	return op
}

func (op *OptionsScreen) Update(deltaTime time.Duration) {
	op.Menu.Update(deltaTime)

	if AreKeysPressed(op.InputId, TheKM[EscapeKey]) {
		// TODO : options screen doesn't save settings
		// if it's quit by user
		err := SaveSettings()
		if err != nil {
			ErrorLogger.Println(err)
			DisplayAlert("failed to save settings")
		}
		ShowTransition(BlackPixel, func() {
			SetNextScreen(TheSelectScreen)
			HideTransition()
		})
	}
}

func (op *OptionsScreen) Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	op.Menu.Draw()
}

func (op *OptionsScreen) BeforeScreenTransition() {
	op.Menu.ResetAnimation()
	op.Menu.SelectedIndex = 1

	op.Menu.SetItemNvalue(op.FpsItemId, float32(TheOptions.TargetFPS))
	op.Menu.SetItemBValue(op.DownScrollItemId, TheOptions.DownScroll)
	op.Menu.SetItemBValue(op.LoadAudioDuringGpItemId, TheOptions.LoadAudioDuringGamePlay)
}

func (op *OptionsScreen) Free() {
	// pass
}
