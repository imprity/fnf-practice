package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

var _ = rl.KeyA

type MenuTestScreen struct {
	Menu *MenuDrawer

	InputId InputGroupId
}

func NewMenuTestScreen() *MenuTestScreen {
	mt := new(MenuTestScreen)
	mt.Menu = NewMenuDrawer()

	testDeco := NewMenuItem()
	testDeco.Name = "TestMenu"
	testDeco.Type = MenuItemDeco
	testDeco.FadeIfUnselected = false
	testDeco.Color = Color255(255, 195, 130, 255)
	testDeco.SizeRegular = testDeco.SizeRegular * 1.5
	mt.Menu.AddItems(testDeco)

	testTrigger := NewMenuItem()
	testTrigger.Name = "trigger item"
	testTrigger.Type = MenuItemTrigger
	testTrigger.TriggerCallback = func() {
		FnfLogger.Println("item triggered")
	}
	mt.Menu.AddItems(testTrigger)

	testToggle := NewMenuItem()
	testToggle.Name = "toggle item"
	testToggle.Type = MenuItemToggle
	testToggle.ToggleStyleCheckBox = false
	testToggle.ToggleCallback = func(bValue bool) {
		FnfLogger.Printf("toggle %v\n", bValue)
	}
	mt.Menu.AddItems(testToggle)

	testCheckbox := NewMenuItem()
	testCheckbox.Name = "checkbox item"
	testCheckbox.Type = MenuItemToggle
	testCheckbox.ToggleStyleCheckBox = true
	testCheckbox.ToggleCallback = func(bValue bool) {
		FnfLogger.Printf("checkbox %v\n", bValue)
	}
	mt.Menu.AddItems(testCheckbox)

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
	mt.Menu.AddItems(testNumber)

	testList := NewMenuItem()
	testList.Name = "list item"
	testList.Type = MenuItemList
	testList.List = []string{"apple", "banana", "kiwi"}
	testList.ListCallback = func(selected int, list []string) {
		FnfLogger.Printf("list selected : %v", list[selected])
	}
	mt.Menu.AddItems(testList)

	testKey := NewMenuItem()
	testKey.Name = "key item"
	testKey.Type = MenuItemKey
	testKey.AddKeys(rl.KeyA)
	testKey.KeyCallback = func(index int, prevKey, newKey int32) {
		FnfLogger.Printf("%vth key changed from %v to %v", index, GetKeyName(prevKey), GetKeyName(newKey))
	}
	mt.Menu.AddItems(testKey)

	testKeyMany := NewMenuItem()
	testKeyMany.Name = "key item many"
	testKeyMany.Type = MenuItemKey
	testKeyMany.AddKeys(rl.KeyLeft)
	testKeyMany.AddKeys(rl.KeyRight)
	testKeyMany.KeyCallback = func(index int, prevKey, newKey int32) {
		FnfLogger.Printf("%vth key changed from %v to %v", index, GetKeyName(prevKey), GetKeyName(newKey))
	}
	mt.Menu.AddItems(testKeyMany)

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
		mt.Menu.AddItems(testKeyMany)
	}

	return mt
}

func (mt *MenuTestScreen) Update(deltaTime time.Duration) {
	mt.Menu.Update(deltaTime)
}

func (mt *MenuTestScreen) Draw() {
	DrawPatternBackground(GameScreenBg, 0, 0, rl.Color{100, 100, 100, 255})

	mt.Menu.Draw()
}

func (mt *MenuTestScreen) BeforeScreenTransition() {
}

func (mt *MenuTestScreen) Free() {
	// pass
}
