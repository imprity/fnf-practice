package fnf

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// ============================
// BaseOptionsScreen stuff
// ============================
type BaseOptionsScreen struct {
	Menu *MenuDrawer

	InputId InputGroupId

	PrevScreen                 Screen
	ShowScreenTransitionEffect bool

	helpMessages map[MenuItemId][]RichTextElement

	helpMessageOpacity float32

	onMatchItemsToOption []func()

	selectFirstItem bool
}

func newBaseOptionsScreen() *BaseOptionsScreen {
	op := new(BaseOptionsScreen)
	op.Menu = NewMenuDrawer()
	op.InputId = NewInputGroupId()
	op.helpMessages = make(map[MenuItemId][]RichTextElement)

	return op
}

func (op *BaseOptionsScreen) HelpMessageDefaultStyle() RichTextStyle {
	return RichTextStyle{
		FontSize: 30,
		Font:     SdfFontClear,
		Fill:     FnfColor{0, 0, 0, 255},
	}
}

func (op *BaseOptionsScreen) addHelpMessageImpl(
	id MenuItemId,
	marginVertical float32,
	isBottomMargin bool,
	marginRight float32,
	width float32,
	richText string,
) {
	factory := NewRichTextFactory(width)
	factory.SetStyle(op.HelpMessageDefaultStyle())

	factory.PrintRichText(richText)
	elements := factory.Elements(TextAlignLeft, 0, 35)

	for i := range elements {
		elements[i].Bound.X += SCREEN_WIDTH - (width + marginRight)
	}

	if isBottomMargin {
		bound := ElementsBound(elements)
		for i := range elements {
			elements[i].Bound.Y += SCREEN_HEIGHT - bound.Height - marginVertical
		}
	} else {
		for i := range elements {
			elements[i].Bound.Y += marginVertical
		}
	}

	op.helpMessages[id] = elements
}

func (op *BaseOptionsScreen) AddHelpMessageTopRight(
	id MenuItemId,
	marginTop float32,
	marginRight float32,
	width float32,
	richText string,
) {
	op.addHelpMessageImpl(
		id, marginTop, false, marginRight, width, richText)
}

func (op *BaseOptionsScreen) AddHelpMessageBottomRight(
	id MenuItemId,
	marginBottom float32,
	marginRight float32,
	width float32,
	richText string,
) {
	op.addHelpMessageImpl(
		id, marginBottom, true, marginRight, width, richText)
}

func (op *BaseOptionsScreen) OnMatchItemsToOption(cb func()) {
	op.onMatchItemsToOption = append(op.onMatchItemsToOption, cb)
}

func (op *BaseOptionsScreen) MatchItemsToOption() {
	for _, cb := range op.onMatchItemsToOption {
		cb()
	}
}

func (op *BaseOptionsScreen) Update(deltaTime time.Duration) {
	op.Menu.Update(deltaTime)

	if AreKeysPressed(op.InputId, TheKM[EscapeKey]) && op.PrevScreen != nil {
		op.selectFirstItem = true
		if op.ShowScreenTransitionEffect {
			ShowTransition(BlackPixel, func() {
				SetNextScreen(op.PrevScreen)
				HideTransition()
			})
		} else {
			SetNextScreen(op.PrevScreen)
		}
	}

	id := op.Menu.GetSelectedId()

	if _, ok := op.helpMessages[id]; ok {
		op.helpMessageOpacity += f32(deltaTime) / f32(time.Millisecond*150)
	} else {
		op.helpMessageOpacity = 0
	}

	op.helpMessageOpacity = Clamp(op.helpMessageOpacity, 0, 1)
}

func (op *BaseOptionsScreen) Draw() {
	DrawPatternBackground(MenuScreenSimpleBg, 0, 0, ToRlColor(FnfColor{255, 255, 255, 255}))

	op.Menu.Draw()

	// draw help messages
	id := op.Menu.GetSelectedId()

	if elements, ok := op.helpMessages[id]; ok {
		DrawTextElements(elements, 0, 0,
			Col01(1, 1, 1, op.helpMessageOpacity))
	}
}

func (op *BaseOptionsScreen) BeforeScreenTransition() {
	op.Menu.BeforeScreenTransition()

	op.MatchItemsToOption()
}

func (op *BaseOptionsScreen) BeforeScreenEnd() {
	// TODO : options screen doesn't save settings
	// if it's quit by user
	err := SaveSettings()
	if err != nil {
		ErrorLogger.Println(err)
		DisplayAlert("failed to save settings")
	}

	if op.selectFirstItem {
		op.Menu.SelectItemAt(0, false) // select first item
		op.selectFirstItem = false
	}
}

func (op *BaseOptionsScreen) Free() {
	op.Menu.Free()
}

// ===================================
// end of BaseOptionsScreen stuff
// ===================================

func NewOptionsMainScreen() *BaseOptionsScreen {
	op := newBaseOptionsScreen()

	op.PrevScreen = TheSelectScreen
	op.ShowScreenTransitionEffect = true

	optionsDeco := NewMenuItem()
	optionsDeco.Name = "Options"
	optionsDeco.Type = MenuItemDeco
	optionsDeco.Color = FnfColor{0xE3, 0x9C, 0x02, 0xFF}
	optionsDeco.FadeIfUnselected = false
	optionsDeco.SizeRegular = MenuItemDefaults.SizeRegular * 1.7
	optionsDeco.SizeSelected = MenuItemDefaults.SizeSelected * 1.7
	op.Menu.AddItems(optionsDeco)

	backItem := NewMenuItem()
	backItem.Name = "Return To Menu"
	backItem.Type = MenuItemTrigger
	backItem.TriggerCallback = func() {
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
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemNvalue(fpsItem.Id, false, f32(TheOptions.TargetFPS))
	})

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
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemNvalue(volumeItem.Id, false, f32(TheOptions.Volume)*10)
	})

	loadAudioDuringGpItem := NewMenuItem()
	loadAudioDuringGpItem.Name = "Load Audio During Game Play"
	loadAudioDuringGpItem.Type = MenuItemToggle
	loadAudioDuringGpItem.ToggleCallback = func(bValue bool) {
		TheOptions.LoadAudioDuringGamePlay = bValue
	}
	loadAudioDuringGpItem.SizeSelected = 75
	loadAudioDuringGpItem.SelectedLeftMargin = 5
	op.Menu.AddItems(loadAudioDuringGpItem)
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemBValue(loadAudioDuringGpItem.Id, false, TheOptions.LoadAudioDuringGamePlay)
	})

	op.AddHelpMessageTopRight(loadAudioDuringGpItem.Id,
		50, 50, 460,
		`Load audio during game play. May cause some issues and definitely not recommended if you use slow PC.`,
	)

	gamePlayItem := NewMenuItem()
	gamePlayItem.Name = "Game Play"
	gamePlayItem.Type = MenuItemTrigger
	gamePlayItem.TriggerCallback = func() {
		SetNextScreen(TheOptionsGamePlayScreen)
	}
	op.Menu.AddItems(gamePlayItem)

	controlsItem := NewMenuItem()
	controlsItem.Name = "Controls"
	controlsItem.Type = MenuItemTrigger
	controlsItem.TriggerCallback = func() {
		SetNextScreen(TheOptionsControlsScreen)
	}
	op.Menu.AddItems(controlsItem)

	// ===========================
	// reset every options button
	// ===========================
	resetOptItem := NewMenuItem()
	resetOptItem.Name = "RESET OPTIONS"
	resetOptItem.Type = MenuItemTrigger

	resetOptItem.TopMargin = 40

	resetOptItem.StrokeColorSelected = FnfColor{0xF6, 0x08, 0x08, 0xFF}
	resetOptItem.StrokeWidthSelected = 10
	resetOptItem.ColorSelected = FnfWhite

	resetOptItem.TriggerCallback = func() {
		DisplayOptionsPopup(
			"Reset every options to default?", false,
			[]string{"Yes", "No"},
			func(selected string, isCanceled bool) {
				if isCanceled {
					return
				}

				if selected == "Yes" {
					TheOptions = DefaultOptions
					TheKM = DefaultKM
					op.MatchItemsToOption()
				}
			},
		)
	}

	op.Menu.AddItems(resetOptItem)

	return op
}

func NewOptionsGamePlayScreen() *BaseOptionsScreen {
	op := newBaseOptionsScreen()

	op.PrevScreen = TheOptionsMainScreen
	op.ShowScreenTransitionEffect = false

	optionsDeco := NewMenuItem()
	optionsDeco.Name = "Game Play"
	optionsDeco.Type = MenuItemDeco
	optionsDeco.Color = FnfColor{0xE3, 0x9C, 0x02, 0xFF}
	optionsDeco.FadeIfUnselected = false
	optionsDeco.SizeRegular = MenuItemDefaults.SizeRegular * 1.7
	optionsDeco.SizeSelected = MenuItemDefaults.SizeSelected * 1.7
	op.Menu.AddItems(optionsDeco)

	backItem := NewMenuItem()
	backItem.Name = "Back"
	backItem.Type = MenuItemTrigger
	backItem.TriggerCallback = func() {
		SetNextScreen(TheOptionsMainScreen)
	}
	op.Menu.AddItems(backItem)

	downScrollItem := NewMenuItem()
	downScrollItem.Name = "Down Scroll"
	downScrollItem.Type = MenuItemToggle
	downScrollItem.ToggleCallback = func(bValue bool) {
		TheOptions.DownScroll = bValue
	}
	op.Menu.AddItems(downScrollItem)
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemBValue(downScrollItem.Id, false, TheOptions.DownScroll)
	})

	middleScrollItem := NewMenuItem()
	middleScrollItem.Name = "Middle Scroll"
	middleScrollItem.Type = MenuItemToggle
	middleScrollItem.ToggleCallback = func(bValue bool) {
		TheOptions.MiddleScroll = bValue
	}
	op.Menu.AddItems(middleScrollItem)
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemBValue(middleScrollItem.Id, false, TheOptions.MiddleScroll)
	})

	ghostTapping := NewMenuItem()
	ghostTapping.Name = "Ghost Tapping"
	ghostTapping.Type = MenuItemToggle
	ghostTapping.ToggleCallback = func(bValue bool) {
		TheOptions.GhostTapping = bValue
	}
	op.Menu.AddItems(ghostTapping)
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemBValue(ghostTapping.Id, false, TheOptions.GhostTapping)
	})

	noteSplash := NewMenuItem()
	noteSplash.Name = "Note Splash"
	noteSplash.Type = MenuItemToggle
	noteSplash.ToggleCallback = func(bValue bool) {
		TheOptions.NoteSplash = bValue
	}
	op.Menu.AddItems(noteSplash)
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemBValue(noteSplash.Id, false, TheOptions.NoteSplash)
	})

	hitSoundPlayer := NewVaryingSpeedPlayer(0, 0)
	hitSoundPlayer.LoadDecodedAudio(HitSoundAudio)

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
		hitSoundPlayer.SetVolume(volume)

		if volume > 0.001 { // just in case
			hitSoundPlayer.Rewind()
			hitSoundPlayer.Play()
		}
	}
	op.Menu.AddItems(hitSoundItem)
	op.OnMatchItemsToOption(func() {
		op.Menu.SetItemNvalue(hitSoundItem.Id, false, float32(TheOptions.HitSoundVolume)*10)
	})

	// ================================
	// add rating options
	// ================================
	{
		var ratingItems [HitRatingSize]MenuItemId

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

		op.OnMatchItemsToOption(func() {
			for r := FnfHitRating(0); r < HitRatingSize; r++ {
				op.Menu.SetItemNvalue(ratingItems[r], false, f32(TheOptions.HitWindows[r])/f32(time.Millisecond))
			}
		})
	}

	return op
}

func NewOptionsControlsScreen() *BaseOptionsScreen {
	op := newBaseOptionsScreen()

	op.PrevScreen = TheOptionsMainScreen
	op.ShowScreenTransitionEffect = false

	optionsDeco := NewMenuItem()
	optionsDeco.Name = "Controls"
	optionsDeco.Type = MenuItemDeco
	optionsDeco.Color = FnfColor{0xE3, 0x9C, 0x02, 0xFF}
	optionsDeco.FadeIfUnselected = false
	optionsDeco.SizeRegular = MenuItemDefaults.SizeRegular * 1.7
	optionsDeco.SizeSelected = MenuItemDefaults.SizeSelected * 1.7
	op.Menu.AddItems(optionsDeco)

	backItem := NewMenuItem()
	backItem.Name = "Back"
	backItem.Type = MenuItemTrigger
	backItem.TriggerCallback = func() {
		SetNextScreen(TheOptionsMainScreen)
	}
	op.Menu.AddItems(backItem)

	// create key control options
	{
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

			op.OnMatchItemsToOption(func() {
				op.Menu.SetItemKeyValues(item.Id, NoteKeys(dir))
			})
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

			op.OnMatchItemsToOption(func() {
				op.Menu.SetItemKeyValues(item.Id, []int32{TheKM[key]})
			})
		}
	}

	return op
}
