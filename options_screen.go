package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
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
	optionsDeco.SizeRegular = MenuItemSizeRegularDefault * 1.7
	optionsDeco.SizeSelected = MenuItemSizeSelectedDefault * 1.7
	op.Menu.AddItems(optionsDeco)

	backItem := NewMenuItem()
	backItem.Name = "Back To Menu"
	backItem.Type = MenuItemTrigger
	backItem.OnValueChange = func(bValue bool, _ float32, _ string) {
		if !bValue {
			return
		}

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
	fpsItem.OnValueChange = func(_ bool, nValue float32, _ string) {
		TheOptions.TargetFPS = int32(nValue)
	}
	op.FpsItemId = fpsItem.Id
	op.Menu.AddItems(fpsItem)

	downScrollItem := NewMenuItem()
	downScrollItem.Name = "Down Scroll"
	downScrollItem.Type = MenuItemToggle
	downScrollItem.OnValueChange = func(bValue bool, _ float32, _ string) {
		TheOptions.DownScroll = bValue
	}
	op.DownScrollItemId = downScrollItem.Id
	op.Menu.AddItems(downScrollItem)

	loadAudioDuringGpItem := NewMenuItem()
	loadAudioDuringGpItem.Name = "Load Audio During Game Play"
	loadAudioDuringGpItem.Type = MenuItemToggle
	loadAudioDuringGpItem.OnValueChange = func(bValue bool, _ float32, _ string) {
		TheOptions.LoadAudioDuringGamePlay = bValue
	}
	op.LoadAudioDuringGpItemId = loadAudioDuringGpItem.Id
	op.Menu.AddItems(loadAudioDuringGpItem)

	return op
}

func (op *OptionsScreen) Update(deltaTime time.Duration) {
	op.Menu.Update(deltaTime)

	if AreKeysPressed(op.InputId, EscapeKey) {
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
