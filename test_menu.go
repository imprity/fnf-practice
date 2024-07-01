package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var _ = rl.KeyA

/*
func init() {
	OverrideFirstScreen(func() Screen {
		return NewTestMenu()
	})
}
*/

func NewTestMenu() *MenuDrawer {
	menu := NewMenuDrawer()

	menu.DrawBackground = true

	menu.Background = MenuBackground{
		Texture: GameScreenBg,
		OffsetX: 0, OffsetY: 0,
		Tint: rl.White,
	}

	testDeco := NewMenuItem()
	testDeco.Name = "TestMenu"
	testDeco.Type = MenuItemDeco
	testDeco.FadeIfUnselected = false
	testDeco.Color = rl.Color{255, 195, 130, 255}
	testDeco.SizeRegular = testDeco.SizeRegular * 1.5
	menu.AddItems(testDeco)

	testTrigger := NewMenuItem()
	testTrigger.Name = "trigger item"
	testTrigger.Type = MenuItemTrigger
	testTrigger.TriggerCallback = func() {
		FnfLogger.Println("item triggered")
	}
	menu.AddItems(testTrigger)

	testToggle := NewMenuItem()
	testToggle.Name = "toggle item"
	testToggle.Type = MenuItemToggle
	testToggle.ToggleStyleCheckBox = false
	testToggle.ToggleCallback = func(bValue bool) {
		FnfLogger.Printf("toggle %v\n", bValue)
	}
	menu.AddItems(testToggle)

	testCheckbox := NewMenuItem()
	testCheckbox.Name = "checkbox item"
	testCheckbox.Type = MenuItemToggle
	testCheckbox.ToggleStyleCheckBox = true
	testCheckbox.ToggleCallback = func(bValue bool) {
		FnfLogger.Printf("checkbox %v\n", bValue)
	}
	menu.AddItems(testCheckbox)

	testNumber := NewMenuItem()
	testNumber.Name = "number item"
	testNumber.Type = MenuItemNumber
	testNumber.NValueMin = -10
	testNumber.NValueMax = 16
	testNumber.NValue = 0
	testNumber.NValueInterval = 3
	testNumber.NValueFmtString = "%.f"
	testNumber.NumberCallback = func(nValue float32) {
		FnfLogger.Printf("number : %v", nValue)
	}
	menu.AddItems(testNumber)

	testList := NewMenuItem()
	testList.Name = "list item"
	testList.Type = MenuItemList
	testList.List = []string{"apple", "banana", "kiwi"}
	testList.ListCallback = func(selected int, list []string) {
		FnfLogger.Printf("list selected : %v", list[selected])
	}
	menu.AddItems(testList)

	testKey := NewMenuItem()
	testKey.Name = "key item"
	testKey.Type = MenuItemKey
	testKey.AddKeys(rl.KeyA)
	testKey.KeyCallback = func(index int, prevKey, newKey int32) {
		FnfLogger.Printf("%vth key changed from %v to %v", index, GetKeyName(prevKey), GetKeyName(newKey))
	}
	menu.AddItems(testKey)

	testKeyMany := NewMenuItem()
	testKeyMany.Name = "key item many"
	testKeyMany.Type = MenuItemKey
	testKeyMany.AddKeys(rl.KeyLeft)
	testKeyMany.AddKeys(rl.KeyRight)
	testKeyMany.KeyCallback = func(index int, prevKey, newKey int32) {
		FnfLogger.Printf("%vth key changed from %v to %v", index, GetKeyName(prevKey), GetKeyName(newKey))
	}
	menu.AddItems(testKeyMany)

	{
		testKeyMany := NewMenuItem()
		testKeyMany.Name = "key item many2"
		testKeyMany.Type = MenuItemKey
		testKeyMany.AddKeys(rl.KeyLeft)
		testKeyMany.AddKeys(rl.KeyRight)
		testKeyMany.AddKeys(rl.KeyUp)
		testKeyMany.KeyCallback = func(index int, prevKey, newKey int32) {
			FnfLogger.Printf("%vth key changed from %v to %v", index, GetKeyName(prevKey), GetKeyName(newKey))
		}
		menu.AddItems(testKeyMany)
	}

	return menu
}
