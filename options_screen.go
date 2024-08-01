package fnf

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type OptionsScreen struct {
	Menu *MenuDrawer

	setItemValuesToOptions func()

	InputId InputGroupId

	HelpMessages map[MenuItemId][]RichTextElement

	HelpMessageOpacity float32

	HitSoundPlayer *VaryingSpeedPlayer
}

func NewOptionsScreen() *OptionsScreen {
	op := new(OptionsScreen)

	op.InputId = NewInputGroupId()

	op.Menu = NewMenuDrawer()

	op.HelpMessages = make(map[MenuItemId][]RichTextElement)

	op.HitSoundPlayer = NewVaryingSpeedPlayer(0, 0)
	op.HitSoundPlayer.LoadDecodedAudio(HitSoundAudio)

	addHelpMessage := func(id MenuItemId, marginTop, marginRight, width float32, richText string) {
		factory := NewRichTextFactory(width)
		factory.SetStyle(RichTextStyle{
			FontSize: 30,
			Font:     SdfFontClear,
			Fill:     FnfColor{0, 0, 0, 255},
		})

		factory.PrintRichText(richText)
		elements := factory.Elements(TextAlignLeft, 0, 35)

		for i := range elements {
			elements[i].Bound.X += SCREEN_WIDTH - (width + marginRight)
			elements[i].Bound.Y += marginTop
		}

		op.HelpMessages[id] = elements
	}

	optionsDeco := NewMenuItem()
	optionsDeco.Name = "Options"
	optionsDeco.Type = MenuItemDeco
	optionsDeco.Color = FnfColor{0xE3, 0x9C, 0x02, 0xFF}
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
	op.Menu.AddItems(fpsItem)

	volumeItem := NewMenuItem()
	volumeItem.Name = "Volume"
	volumeItem.Type = MenuItemNumber
	volumeItem.NValue = float32(TheOptions.Volume)
	volumeItem.NValueMin = 0
	volumeItem.NValueMax = 10
	volumeItem.NValueInterval = 1
	volumeItem.NValueFmtString = "%1.f"
	volumeItem.NumberCallback = func(nValue float32) {
		TheOptions.Volume = float64(nValue) / 10
	}
	op.Menu.AddItems(volumeItem)

	downScrollItem := NewMenuItem()
	downScrollItem.Name = "Down Scroll"
	downScrollItem.Type = MenuItemToggle
	downScrollItem.ToggleCallback = func(bValue bool) {
		TheOptions.DownScroll = bValue
	}
	op.Menu.AddItems(downScrollItem)

	middleScrollItem := NewMenuItem()
	middleScrollItem.Name = "Middle Scroll"
	middleScrollItem.Type = MenuItemToggle
	middleScrollItem.ToggleCallback = func(bValue bool) {
		TheOptions.MiddleScroll = bValue
	}
	op.Menu.AddItems(middleScrollItem)

	ghostTapping := NewMenuItem()
	ghostTapping.Name = "Ghost Tapping"
	ghostTapping.Type = MenuItemToggle
	ghostTapping.ToggleCallback = func(bValue bool) {
		TheOptions.GhostTapping = bValue
	}
	op.Menu.AddItems(ghostTapping)

	hitSoundItem := NewMenuItem()
	hitSoundItem.Name = "Hit Sound"
	hitSoundItem.Type = MenuItemNumber
	hitSoundItem.NValue = float32(TheOptions.HitSoundVolume)
	hitSoundItem.NValueMin = 0
	hitSoundItem.NValueMax = 10
	hitSoundItem.NValueInterval = 1
	hitSoundItem.NValueFmtString = "%1.f"
	hitSoundItem.NumberCallback = func(nValue float32) {
		volume := float64(nValue) / 10

		TheOptions.HitSoundVolume = volume
		op.HitSoundPlayer.SetVolume(volume)

		if volume > 0.001 { // just in case
			op.HitSoundPlayer.Rewind()
			op.HitSoundPlayer.Play()
		}
	}
	op.Menu.AddItems(hitSoundItem)

	loadAudioDuringGpItem := NewMenuItem()
	loadAudioDuringGpItem.Name = "Load Audio During Game Play"
	loadAudioDuringGpItem.Type = MenuItemToggle
	loadAudioDuringGpItem.ToggleCallback = func(bValue bool) {
		TheOptions.LoadAudioDuringGamePlay = bValue
	}
	loadAudioDuringGpItem.SizeSelected = 75
	loadAudioDuringGpItem.SelectedLeftMargin = 5
	op.Menu.AddItems(loadAudioDuringGpItem)

	addHelpMessage(loadAudioDuringGpItem.Id,
		50, 50, 460,
		`Load audio during game paly. May cause some issues and definitely not recommended if you use slow PC.`,
	)

	var ratingItems [HitRatingSize]MenuItemId

	// add rating options
	{
		deco := NewMenuItem()
		deco.Name = "Hit Window Size"
		deco.Type = MenuItemDeco
		deco.SizeRegular = MenuItemDefaults.SizeRegular * 1.4
		deco.SizeSelected = MenuItemDefaults.SizeSelected * 1.4
		deco.Color = FnfColor{0xFC, 0x9F, 0x7C, 0xFF}
		deco.FadeIfUnselected = false
		op.Menu.AddItems(deco)

		ratingOptNames := [HitRatingSize]string{
			"Bad Hit Window",
			"Good Hit Window",
			"Sick! Hit Window",
		}

		ratingOptOrder := [HitRatingSize]FnfHitRating{
			HitRatingSick,
			HitRatingGood,
			HitRatingBad,
		}

		for _, rating := range ratingOptOrder {
			ratingOpt := NewMenuItem()
			ratingOpt.Name = ratingOptNames[rating]
			ratingOpt.Type = MenuItemNumber
			ratingOpt.NValue = float32(TheOptions.HitWindows[rating])
			ratingOpt.NValueMin = 15
			ratingOpt.NValueMax = 2000
			ratingOpt.NValueInterval = 1
			ratingOpt.NValueFmtString = "%1.f"
			ratingOpt.NumberCallback = func(nValue float32) {
				TheOptions.HitWindows[rating] = time.Duration(nValue) * time.Millisecond
			}
			op.Menu.AddItems(ratingOpt)

			ratingItems[rating] = ratingOpt.Id
		}
	}

	op.setItemValuesToOptions = func() {
		op.Menu.SetItemNvalue(fpsItem.Id, f32(TheOptions.TargetFPS))
		op.Menu.SetItemNvalue(volumeItem.Id, f32(TheOptions.Volume)*10)
		op.Menu.SetItemBValue(downScrollItem.Id, TheOptions.DownScroll)
		op.Menu.SetItemBValue(middleScrollItem.Id, TheOptions.MiddleScroll)
		op.Menu.SetItemBValue(loadAudioDuringGpItem.Id, TheOptions.LoadAudioDuringGamePlay)
		op.Menu.SetItemBValue(ghostTapping.Id, TheOptions.GhostTapping)
		op.Menu.SetItemNvalue(hitSoundItem.Id, float32(TheOptions.HitSoundVolume)*10)

		for r := FnfHitRating(0); r < HitRatingSize; r++ {
			op.Menu.SetItemNvalue(ratingItems[r], f32(TheOptions.HitWindows[r])/f32(time.Millisecond))
		}
	}

	// create key control options
	{
		deco := NewMenuItem()
		deco.Name = "Controls"
		deco.Type = MenuItemDeco
		deco.SizeRegular = MenuItemDefaults.SizeRegular * 1.4
		deco.SizeSelected = MenuItemDefaults.SizeSelected * 1.4
		deco.Color = FnfColor{0xFC, 0x9F, 0x7C, 0xFF}
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

		displaySorryMsg := func(newKey int32, duplicateOf FnfBinding) {
			defaultStyle := PopupDefaultRichTextStyle()
			defaultStyleStr := RichTextStyleToStr(defaultStyle)

			highLightStyle := RichTextStyle{
				FontSize: defaultStyle.FontSize * 1.1,
				Font:     SdfFontBold,

				Fill:   FnfColor{255, 255, 255, 255},
				Stroke: FnfColor{0, 0, 0, 255},

				StrokeWidth: 8,
			}
			highLightStyleStr := RichTextStyleToStr(highLightStyle)

			DisplayOptionsPopup(
				fmt.Sprintf("Sorry!\n%s\"%s\"%s key is already assigned to %s\"%s\"",
					highLightStyleStr,
					EscapeRichText(GetKeyName(newKey)),
					defaultStyleStr,
					highLightStyleStr,
					EscapeRichText(KeyHumanName[duplicateOf]),
				),
				true, []string{}, nil,
			)
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
				var duplicateOf FnfBinding

				for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
					if TheKM[binding] == newKey {
						isDuplicate = true
						duplicateOf = binding
					}
				}

				if isDuplicate {
					displaySorryMsg(newKey, duplicateOf)
				} else {
					SetNoteKeys(dir, index, newKey)
					item.KeyValues[index] = newKey
				}
			}
		}

		debugKeys := []FnfBinding{
			ToggleDebugMsg,
			ToggleLogNoteEvent,
			ToggleDebugGraphics,
			ReloadAssetsKey,
		}

		for key := SelectKey; key < FnfBindingSize; key++ {
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
				var duplicateOf FnfBinding

				for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
					if TheKM[binding] == newKey {
						isDuplicate = true
						duplicateOf = binding
					}
				}

				if isDuplicate {
					displaySorryMsg(newKey, duplicateOf)
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

	id := op.Menu.GetSelectedId()

	if _, ok := op.HelpMessages[id]; ok {
		op.HelpMessageOpacity += f32(deltaTime) / f32(time.Millisecond*150)
	} else {
		op.HelpMessageOpacity = 0
	}

	op.HelpMessageOpacity = Clamp(op.HelpMessageOpacity, 0, 1)
}

func (op *OptionsScreen) Draw() {
	DrawPatternBackground(MenuScreenSimpleBg, 0, 0, ToRlColor(FnfColor{255, 255, 255, 255}))

	op.Menu.Draw()

	// draw help messages
	id := op.Menu.GetSelectedId()

	if elements, ok := op.HelpMessages[id]; ok {
		DrawTextElements(elements, 0, 0,
			Col01(1, 1, 1, op.HelpMessageOpacity))
	}
}

func (op *OptionsScreen) BeforeScreenTransition() {
	op.Menu.BeforeScreenTransition()
	op.Menu.SelectItemAt(0, false) // select first item

	if op.setItemValuesToOptions != nil {
		op.setItemValuesToOptions()
	}
}

func (op *OptionsScreen) BeforeScreenEnd() {
}

func (op *OptionsScreen) Free() {
	// pass
}
