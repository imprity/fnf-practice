package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"strings"
	"time"
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

		createKeyOp := func(name string, keys []int32, cb func(index int, prevKey int32, newKey int32)) *MenuItem {
			keyItem := NewMenuItem()
			keyItem.Name = name
			keyItem.Type = MenuItemKey
			keyItem.SizeRegular = 65
			keyItem.SizeSelected = 70
			keyItem.BottomMargin = 15
			keyItem.NameMinWidth = 270
			keyItem.SelectedLeftMargin = 5
			keyItem.AddKeys(keys...)
			keyItem.KeyCallback = cb

			op.Menu.AddItems(keyItem)

			return keyItem
		}

		// direction key
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			d := dir
			createKeyOp(
				// NOTE : I know Title is deprecated but range of input is only 4
				// up down left right
				// So I don't it really matters...
				fmt.Sprintf("%v :", strings.Title(NoteDirStrs[d])),
				NoteKeys(d),
				func(index int, _ int32, newKey int32) {
					SetNoteKeys(dir, index, newKey)
				},
			)
		}

		createKeyOp("Select :", []int32{TheKM[SelectKey]}, func(index int, _ int32, newKey int32) {
			TheKM[SelectKey] = newKey
		})
		createKeyOp("Pause :", []int32{TheKM[PauseKey]}, func(index int, _ int32, newKey int32) {
			TheKM[PauseKey] = newKey
		})
		createKeyOp("Escape :", []int32{TheKM[EscapeKey]}, func(index int, _ int32, newKey int32) {
			TheKM[EscapeKey] = newKey
		})
		createKeyOp("Scroll Up :", []int32{TheKM[NoteScrollUpKey]}, func(index int, _ int32, newKey int32) {
			TheKM[NoteScrollUpKey] = newKey
		})
		createKeyOp("Scroll Down :", []int32{TheKM[NoteScrollDownKey]}, func(index int, _ int32, newKey int32) {
			TheKM[NoteScrollDownKey] = newKey
		})
		speedUp := createKeyOp("Speed Up :", []int32{TheKM[AudioSpeedUpKey]}, func(index int, _ int32, newKey int32) {
			TheKM[AudioSpeedUpKey] = newKey
		})
		speedDown := createKeyOp("Speed Down :", []int32{TheKM[AudioSpeedDownKey]}, func(index int, _ int32, newKey int32) {
			TheKM[AudioSpeedDownKey] = newKey
		})
		createKeyOp("Reset :", []int32{TheKM[SongResetKey]}, func(index int, _ int32, newKey int32) {
			TheKM[SongResetKey] = newKey
		})
		createKeyOp("Bookmark :", []int32{TheKM[SetBookMarkKey]}, func(index int, _ int32, newKey int32) {
			TheKM[SetBookMarkKey] = newKey
		})
		createKeyOp("Jump To Bookmark :", []int32{TheKM[JumpToBookMarkKey]}, func(index int, _ int32, newKey int32) {
			TheKM[JumpToBookMarkKey] = newKey
		})
		spacingUp := createKeyOp("Note Spacing Up :", []int32{TheKM[ZoomInKey]}, func(index int, _ int32, newKey int32) {
			TheKM[ZoomInKey] = newKey
		})
		spacingDown := createKeyOp("Note Spacing Down :", []int32{TheKM[ZoomOutKey]}, func(index int, _ int32, newKey int32) {
			TheKM[ZoomOutKey] = newKey
		})

		// additional settings for specific items
		speedUp.NameMinWidth, speedDown.NameMinWidth = 290, 290

		spacingUp.NameMinWidth, spacingDown.NameMinWidth = 460, 460
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
