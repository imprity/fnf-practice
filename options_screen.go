package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

type OptionsScreen struct {
	Menu *MenuDrawer

	FpsItemId int64
}

func NewOptionsScreen() *OptionsScreen {
	op := new(OptionsScreen)
	op.Menu = NewMenuDrawer()

	optionsDeco := NewMenuItem()
	optionsDeco.Name = "Options"
	optionsDeco.Type = MenuItemDeco
	optionsDeco.Color = Color255(0xE3, 0x9C, 0x02, 0xFF)
	optionsDeco.FadeIfUnselected = false
	optionsDeco.SizeRegular = MenuItemSizeRegularDefault * 1.7
	optionsDeco.SizeSelected = MenuItemSizeSelectedDefault * 1.7
	op.Menu.Items = append(op.Menu.Items, optionsDeco)

	backItem := NewMenuItem()
	backItem.Name = "Back To Menu"
	backItem.Type = MenuItemTrigger
	backItem.OnValueChange = func(bValue bool, _ float32, _ string) {
		if !bValue {
			return
		}

		// TODO : options screen doesn't save settings
		// if it's quit by user
		// TODO : do not ignore error
		SaveSettings()
		ShowTransition(BlackPixel, func() {
			DisableInput()
			SetNextScreen(TheSelectScreen)
			EnableInput()
			HideTransition()
		})
	}
	op.Menu.Items = append(op.Menu.Items, backItem)

	fpsItem := NewMenuItem()
	fpsItem.Name = "Target FPS"
	fpsItem.Type = MenuItemNumber
	fpsItem.NValue = float32(TargetFPS)
	fpsItem.NValueMin = 30
	fpsItem.NValueMax = 500
	fpsItem.NValueInterval = 10
	fpsItem.NValueFmtString = "%1.f"
	fpsItem.OnValueChange = func(_ bool, nValue float32, _ string) {
		TargetFPS = int32(nValue)
	}
	op.FpsItemId = fpsItem.Id
	op.Menu.Items = append(op.Menu.Items, fpsItem)

	return op
}

func (op *OptionsScreen) Update(deltaTime time.Duration) {
	op.Menu.Update(deltaTime)

	if AreKeysPressed(EscapeKey) {
		// TODO : options screen doesn't save settings
		// if it's quit by user
		// TODO : do not ignore error
		SaveSettings()
		ShowTransition(BlackPixel, func() {
			DisableInput()
			SetNextScreen(TheSelectScreen)
			EnableInput()
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

	fpsItem := op.Menu.GetItemById(op.FpsItemId)
	fpsItem.NValue = float32(TargetFPS)

	EnableInput()
}
