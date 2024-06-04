package main

import (
	"fmt"
	"math"
	"slices"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var DrawMenuDebug bool = false

type MenuItemType int

const (
	MenuItemTrigger MenuItemType = iota
	MenuItemToggle
	MenuItemNumber
	MenuItemList
	MenuItemKey
	MenuItemDeco
	MenuItemTypeSize
)

var menuItemTypeStrs [MenuItemTypeSize]string

func init() {
	menuItemTypeStrs[MenuItemTrigger] = "Trigger"
	menuItemTypeStrs[MenuItemToggle] = "Toggle"
	menuItemTypeStrs[MenuItemNumber] = "Number"
	menuItemTypeStrs[MenuItemList] = "List"
	menuItemTypeStrs[MenuItemKey] = "Key"
	menuItemTypeStrs[MenuItemDeco] = "Deco"
}

func MenuItemTypeName(t MenuItemType) string {
	if 0 <= t && t < MenuItemTypeSize {
		return menuItemTypeStrs[t]
	}
	return fmt.Sprintf("invalid(%v)", t)
}

type MenuItemId int64

type MenuItem struct {
	Type MenuItemType

	Id MenuItemId

	Name string

	SizeRegular  float32
	SizeSelected float32

	Color            Color
	FadeIfUnselected bool

	// margin between next item
	// default is 30
	BottomMargin float32

	// default is 30
	SelectedLeftMargin float32

	NameMinWidth float32

	NameValueSeperator string

	BValue bool

	NValue float32

	NValueMin      float32
	NValueMax      float32
	NValueInterval float32

	ListSelected int
	List         []string

	KeyValues []int32

	TriggerCallback func()
	ToggleCallback  func(bool)
	NumberCallback  func(float32)
	ListCallback    func(selected int, list []string)
	KeyCallback     func(index int, prevKey int32, newKey int32)

	UserData any

	IsHidden bool

	// format string to use to displat NValue
	NValueFmtString string

	// whether if toggle item will use checkbox or < yes, no >
	ToggleStyleCheckBox bool

	CheckedBoxColor   Color // default is 0x79 E4 AF FF
	UncheckedBoxColor Color // default is 0xD1 D1 D1 FF

	CheckmarkColor Color // default is 0xFF FF FF FF

	KeyColorRegular  Color // default is 0xFF FF FF C8
	KeyColorSelected Color // default is 0x0A FA 72 FF

	// variables for animations
	NameClickTimer       time.Duration
	ValueClickTimer      time.Duration
	LeftArrowClickTimer  time.Duration
	RightArrowClickTimer time.Duration
	KeySelectTimer       time.Duration

	bound rl.Rectangle
}

var menuItemIdGenerator IdGenerator[MenuItemId]

var MenuItemDefaults = MenuItem{
	SizeRegular:  70,
	SizeSelected: 90,

	Color: Col(1, 1, 1, 1),

	FadeIfUnselected:    true,
	ToggleStyleCheckBox: true,

	BottomMargin:       30,
	SelectedLeftMargin: 30,

	CheckedBoxColor:   Color255(0x79, 0xE4, 0xAF, 0xFF),
	UncheckedBoxColor: Color255(0xD1, 0xD1, 0xD1, 0xFF),

	CheckmarkColor: Color255(0xFF, 0xFF, 0xFF, 0xFF),

	KeyColorRegular:  Color255(0xFF, 0xFF, 0xFF, 0xC8),
	KeyColorSelected: Color255(0x0A, 0xFA, 0x72, 0xFF),
}

func NewMenuItem() *MenuItem {
	item := MenuItemDefaults

	item.Id = menuItemIdGenerator.NewId()

	item.NameClickTimer = -Years150
	item.ValueClickTimer = -Years150
	item.LeftArrowClickTimer = -Years150
	item.RightArrowClickTimer = -Years150
	item.KeySelectTimer = -Years150

	return &item
}

func (mi *MenuItem) AddKeys(keys ...int32) {
	for _, key := range keys {
		mi.KeyValues = append(mi.KeyValues, key)
	}
}

// creates deco item with " " as it's name
// intended to be used for rendering complex "deco" item
func NewDummyDecoMenuItem(size float32) *MenuItem {
	dummy := NewMenuItem()

	dummy.Type = MenuItemDeco

	dummy.Name = " "

	dummy.SizeRegular = size

	return dummy
}

func (mi *MenuItem) CanDecrement() bool {
	return mi.NValue-mi.NValueInterval >= mi.NValueMin-0.00001
}

func (mi *MenuItem) CanIncrement() bool {
	return mi.NValue+mi.NValueInterval <= mi.NValueMax+0.00001
}

func (mi *MenuItem) IsSelectable() bool {
	return !mi.IsHidden && mi.Type != MenuItemDeco
}

const (
	MenuInputStateNotSelectingKey = iota
	MenuInputStateWaitingKeyPress
	MenuInputStateWaitingKeyRelease
)

type MenuDrawer struct {
	SelectedIndex int

	Yoffset float32

	ScrollAnimT float32

	InputState int

	InputId InputGroupId

	keySelectedIndex int

	items []*MenuItem
}

func NewMenuDrawer() *MenuDrawer {
	md := new(MenuDrawer)

	md.ScrollAnimT = 1

	md.InputId = NewInputGroupId()

	return md
}

func (md *MenuDrawer) IsInputEnabled() bool {
	return IsInputEnabled(md.InputId)
}

func (md *MenuDrawer) IsInputDisabled() bool {
	return IsInputDisabled(md.InputId)
}

func (md *MenuDrawer) DisableInput() {
	DisableInput(md.InputId)
}

func (md *MenuDrawer) EnableInput() {
	EnableInput(md.InputId)
}

func (md *MenuDrawer) keySelected() int {
	item := md.GetSelectedItem()
	if item == nil {
		return 0
	} else if len(item.KeyValues) <= 0 {
		return 0
	} else {
		return Clamp(md.keySelectedIndex, 0, len(item.KeyValues)-1)
	}
}

func (md *MenuDrawer) Update(deltaTime time.Duration) {
	if len(md.items) <= 0 {
		return
	}

	var itemCallback func() = nil

	for index, item := range md.items {
		if item.Type == MenuItemTrigger {
			md.items[index].BValue = false
		}
	}

	prevSelected := md.SelectedIndex

	noSelectable := true
	selectableItemCount := 0

	for _, item := range md.items {
		if item.IsSelectable() {
			selectableItemCount += 1
			noSelectable = false
		}
	}

	scrollUntilSelectable := func(forward bool) {
		for {
			if forward {
				md.SelectedIndex += 1
			} else {
				md.SelectedIndex -= 1
			}

			if md.SelectedIndex >= len(md.items) {
				md.SelectedIndex = 0
			} else if md.SelectedIndex < 0 {
				md.SelectedIndex = len(md.items) - 1
			}

			if md.items[md.SelectedIndex].IsSelectable() {
				break
			}
		}
	}

	if !noSelectable {
		if !md.items[md.SelectedIndex].IsSelectable() {
			scrollUntilSelectable(true)
		}
	}

	tryingToMove := false
	tryingToMoveUp := false
	canNotMove := false

	if selectableItemCount <= 1 {
		canNotMove = true
	}

	// ==========================
	// handling input
	// ==========================
	if md.InputState == MenuInputStateWaitingKeyPress {
		if pressed, key := AnyKeyPressed(md.InputId); pressed {
			selected := md.items[md.SelectedIndex]

			if selected.Type == MenuItemKey {
				keySelected := md.keySelected()
				if len(selected.KeyValues) > 0 && selected.KeyCallback != nil {
					prevKey := selected.KeyValues[keySelected]
					newKey := key
					itemCallback = func() {
						selected.KeyCallback(keySelected, prevKey, newKey)
					}
				}
				selected.KeyValues[keySelected] = key
			} else {
				ErrorLogger.Fatalf("wrong type of MenuItem : %v", MenuItemTypeName(selected.Type))
			}

			md.InputState = MenuInputStateWaitingKeyRelease
		}
	} else if md.InputState == MenuInputStateWaitingKeyRelease {
		var menuKeys []int32
		menuKeys = append(menuKeys, TheKM.SelectKey, TheKM.EscapeKey)
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			menuKeys = append(menuKeys, NoteKeys(dir)...)
		}

		if !AreKeysDown(md.InputId, menuKeys...) {
			if IsInputSoloEnabled(md.InputId) {
				ClearSoloInput()
			}
			md.InputState = MenuInputStateNotSelectingKey
		}
	} else {
		callItemCallback := func(item *MenuItem) {
			itemCallback = func() {
				switch item.Type {
				case MenuItemTrigger:
					if item.TriggerCallback != nil {
						item.TriggerCallback()
					}
				case MenuItemToggle:
					if item.ToggleCallback != nil {
						item.ToggleCallback(item.BValue)
					}
				case MenuItemNumber:
					if item.NumberCallback != nil {
						item.NumberCallback(item.NValue)
					}
				case MenuItemList:
					if item.ListCallback != nil {
						selected := Clamp(item.ListSelected, 0, len(item.List))
						item.ListCallback(selected, item.List)
					}
				case MenuItemKey:
					ErrorLogger.Fatal("MenuItemKey should not be called here")
				case MenuItemDeco:
					// pass
				default:
					ErrorLogger.Fatalf("unknow item type : %v\n", item.Type)
				}
			}
		}

		if AreKeysDown(md.InputId, NoteKeys(NoteDirUp)...) {
			tryingToMove = true
			tryingToMoveUp = true
		}

		if AreKeysDown(md.InputId, NoteKeys(NoteDirDown)...) {
			tryingToMove = true
			tryingToMoveUp = false
		}

		// check if menu items are all deco
		const scrollFirstRate = time.Millisecond * 200
		const scrollRepeatRate = time.Millisecond * 110

		if HandleKeyRepeat(md.InputId, scrollFirstRate, scrollRepeatRate, NoteKeys(NoteDirUp)...) {
			if !noSelectable {
				scrollUntilSelectable(false)
			}
		}

		if HandleKeyRepeat(md.InputId, scrollFirstRate, scrollRepeatRate, NoteKeys(NoteDirDown)...) {
			if !noSelectable {
				scrollUntilSelectable(true)
			}
		}

		if !noSelectable {
			selected := md.items[md.SelectedIndex]

			// ===================================
			// handle select key interaction
			// ===================================
			if AreKeysPressed(md.InputId, TheKM.SelectKey) {
				switch selected.Type {
				case MenuItemTrigger:
					selected.BValue = true
					selected.NameClickTimer = GlobalTimerNow()
					callItemCallback(selected)
				case MenuItemToggle:
					selected.BValue = !selected.BValue
					selected.ValueClickTimer = GlobalTimerNow()
					callItemCallback(selected)
				case MenuItemKey:
					selected.ValueClickTimer = GlobalTimerNow()
					if len(selected.KeyValues) > 0 {
						SetSoloInput(md.InputId)
						md.InputState = MenuInputStateWaitingKeyPress
					}
				}
			}

			// ===================================
			// 'left' and 'right' key interaction
			// ===================================
			switch selected.Type {
			case MenuItemList, MenuItemNumber, MenuItemToggle, MenuItemKey:
				// check if user wants to go left or right
				const firstRate = time.Millisecond * 200
				const repeateRate = time.Millisecond * 110

				wantGoLeft := HandleKeyRepeat(md.InputId, firstRate, repeateRate, NoteKeys(NoteDirLeft)...)
				wantGoRight := HandleKeyRepeat(md.InputId, firstRate, repeateRate, NoteKeys(NoteDirRight)...)

				// check if item can go left or right
				canGoLeft := true
				canGoRight := true

				switch selected.Type {
				case MenuItemNumber:
					canGoLeft = selected.CanDecrement()
					canGoRight = selected.CanIncrement()
				case MenuItemList:
					canGoLeft = len(selected.List) > 0
					canGoRight = len(selected.List) > 0
				case MenuItemToggle:
					canGoLeft = !selected.ToggleStyleCheckBox
					canGoRight = !selected.ToggleStyleCheckBox
				case MenuItemKey:
					keySelected := md.keySelected()
					canGoLeft = keySelected > 0
					canGoRight = keySelected+1 < len(selected.KeyValues)
				}

				// check if item actually has to go left and right
				goLeft := false
				goRight := false

				goLeft = wantGoLeft && canGoLeft
				goRight = wantGoRight && canGoRight

				// handle different item interaction based on left and right
				switch selected.Type {
				case MenuItemToggle:
					if !selected.ToggleStyleCheckBox {
						if wantGoLeft {
							selected.LeftArrowClickTimer = GlobalTimerNow()
						}
						if wantGoRight {
							selected.RightArrowClickTimer = GlobalTimerNow()
						}
					}

					if goLeft || goRight {
						selected.BValue = !selected.BValue
						callItemCallback(selected)
					}
				case MenuItemList:
					if wantGoLeft {
						selected.LeftArrowClickTimer = GlobalTimerNow()
					}
					if wantGoRight {
						selected.RightArrowClickTimer = GlobalTimerNow()
					}

					if len(selected.List) > 0 {
						listSelected := selected.ListSelected

						if goLeft {
							listSelected -= 1
						} else if goRight {
							listSelected += 1
						}

						if listSelected >= len(selected.List) {
							listSelected = 0
						} else if listSelected < 0 {
							listSelected = len(selected.List) - 1
						}

						if selected.ListSelected != listSelected {
							selected.ListSelected = listSelected
							callItemCallback(selected)
						}
					}
				case MenuItemNumber:
					if wantGoLeft {
						selected.LeftArrowClickTimer = GlobalTimerNow()
					}
					if wantGoRight {
						selected.RightArrowClickTimer = GlobalTimerNow()
					}

					if goLeft {
						selected.NValue -= selected.NValueInterval
						callItemCallback(selected)
					} else if goRight {
						selected.NValue += selected.NValueInterval
						callItemCallback(selected)
					}
				case MenuItemKey:
					prevKeySelected := md.keySelected()
					if goLeft {
						md.keySelectedIndex = prevKeySelected - 1
					} else if goRight {
						md.keySelectedIndex = prevKeySelected + 1
					}

					if wantGoLeft || wantGoRight {
						if (wantGoLeft && !canGoLeft) || (wantGoRight && !canGoRight) {
							selected.ValueClickTimer = GlobalTimerNow()
						} else {
							selected.KeySelectTimer = GlobalTimerNow()
						}
					}
				}
			}
		}
	}
	// ==========================
	// end of handling input
	// ==========================

	if md.SelectedIndex != prevSelected {
		md.ScrollAnimT = 0
	}

	// but I have a strong feeling that this is not frame indipendent
	// but it's just for menu so I don't think it matters too much...
	selected := md.items[md.SelectedIndex]

	blend := Clamp(float32(deltaTime.Seconds()*20), 0.00, 1.0)

	seletionY := float32(SCREEN_HEIGHT * 0.5)
	seletionY -= selected.SizeRegular * 0.5

	for index, item := range md.items {
		if index >= md.SelectedIndex {
			break
		}

		if item.IsHidden {
			continue
		}

		seletionY -= item.SizeRegular + item.BottomMargin
	}

	if tryingToMove && canNotMove {
		push := (selected.SizeRegular*0.5 + 30) * 0.8
		if tryingToMoveUp {
			seletionY += push
		} else {
			seletionY -= push
		}
	}

	md.Yoffset = Lerp(md.Yoffset, seletionY, blend)

	md.ScrollAnimT = Lerp(md.ScrollAnimT, 1.0, blend)

	// ================================
	// actually call item callback
	// ================================
	if itemCallback != nil {
		itemCallback()
	}

}

func (md *MenuDrawer) Draw() {
	if len(md.items) <= 0 {
		return
	}

	if DrawMenuDebug {
		rl.DrawLine(
			0, SCREEN_HEIGHT*0.5,
			SCREEN_WIDTH, SCREEN_HEIGHT*0.5,
			rl.Color{255, 0, 0, 255})

		for _, item := range md.items {
			rl.DrawRectangleRec(item.bound, rl.Color{255, 0, 0, 100})
		}
	}

	calcClick := func(timer time.Duration) float32 {
		clickT := float64(GlobalTimerNow()-timer) / float64(time.Millisecond*150)

		if clickT > 0 {
			if clickT > 1 {
				clickT = 1
			}
			tt := -clickT * (clickT - 1)
			return float32(1.0 - tt*0.4)
		} else {
			return 1
		}
	}

	calcArrowClick := func(timer time.Duration) float32 {
		clickT := float64(TimeSinceNow(timer)) / float64(time.Millisecond*70)
		clickT = Clamp(clickT, 0, 1)

		tt := clickT * clickT
		return float32(tt*0.1 + 0.9)
	}

	yOffset := md.Yoffset
	xOffset := float32(100)

	xAdvance := xOffset
	yCenter := float32(0)

	itemBound := rl.Rectangle{}
	itemBoundSet := false

	updateItemBound := func(bound rl.Rectangle) {
		if !itemBoundSet {
			itemBound = bound
			itemBoundSet = true
		} else {
			itemBound = RectUnion(itemBound, bound)
		}
	}

	xDrawOffset := float32(0)
	yDrawOffset := float32(0)

	drawText := func(text string, font rl.Font, fontSize, scale float32, col Color) float32 {
		textSize := rl.MeasureTextEx(font, text, fontSize, 0)

		pos := rl.Vector2{
			X: xAdvance + textSize.X*0.5*(1-scale),
			Y: yCenter - textSize.Y*scale*0.5,
		}

		pos.X += xDrawOffset
		pos.Y += yDrawOffset

		rl.DrawTextEx(font, text, pos, fontSize*scale, 0, col.ToRlColor())

		bound := rl.Rectangle{
			X: pos.X, Y: pos.Y,
			Width: textSize.X * scale, Height: textSize.Y * scale,
		}
		updateItemBound(bound)

		return textSize.X
	}

	drawTextCentered := func(text string, font rl.Font, fontSize, scale, width float32, col Color) float32 {
		textSize := rl.MeasureTextEx(font, text, fontSize, 0)

		width = max(textSize.X, width)

		pos := rl.Vector2{
			X: xAdvance + (width-textSize.X*scale)*0.5,
			Y: yCenter - textSize.Y*scale*0.5,
		}

		pos.X += xDrawOffset
		pos.Y += yDrawOffset

		rl.DrawTextEx(font, text, pos, fontSize*scale, 0, col.ToRlColor())

		bound := rl.Rectangle{
			X: pos.X, Y: pos.Y,
			Width: textSize.X * scale, Height: textSize.Y * scale,
		}

		updateItemBound(bound)

		return width
	}

	drawImage := func(
		img rl.Texture2D, srcRect rl.Rectangle, height, scale float32, col Color) float32 {

		wScale := height / srcRect.Height

		dstRect := rl.Rectangle{
			X: xAdvance, Y: yCenter - height*0.5*scale,
			Width: wScale * srcRect.Width * scale, Height: height * scale,
		}

		dstRect.X += xDrawOffset
		dstRect.Y += yDrawOffset

		rl.DrawTexturePro(img, srcRect, dstRect, rl.Vector2{}, 0, col.ToImageRGBA())

		updateItemBound(dstRect)
		return wScale * srcRect.Width
	}

	drawSprite := func(sprite Sprite, spriteN int, height, scale float32, col Color) float32 {
		spriteRect := SpriteRect(sprite, spriteN)

		return drawImage(sprite.Texture, spriteRect, height, scale, col)
	}

	drawArrow := func(drawLeft bool, height, scale float32, fill, stroke Color) float32 {
		var innerSpriteN int
		var outerSpriteN int

		if drawLeft {
			innerSpriteN = UIarrowLeftInner
			outerSpriteN = UIarrowLeftOuter
		} else {
			innerSpriteN = UIarrowRightInner
			outerSpriteN = UIarrowRightOuter
		}

		rl.BeginBlendMode(rl.BlendAlphaPremultiply)
		advance := drawSprite(UIarrowsSprite, innerSpriteN, height, scale, fill)
		drawSprite(UIarrowsSprite, outerSpriteN, height, scale, stroke)
		rl.EndBlendMode()

		return advance
	}

	fadeC := func(col Color, fade float64) Color {
		col.A *= fade
		return col
	}

	dimmC := func(col Color, dimm float64) Color {
		hsv := ToHSV(col)

		hsv[1] *= dimm
		hsv[2] *= dimm

		return FromHSV(hsv)
	}

	for index, item := range md.items {
		if item.IsHidden {
			continue
		}

		yCenter = yOffset + item.SizeRegular*0.5

		xAdvance = xOffset

		fade := float64(0.5)
		size := item.SizeRegular

		if index == md.SelectedIndex {
			fade = Lerp(0.5, 1.0, float64(md.ScrollAnimT))
			size = Lerp(item.SizeRegular, item.SizeSelected, md.ScrollAnimT)
			xAdvance += Lerp(0, item.SelectedLeftMargin, md.ScrollAnimT)
		}

		if !item.FadeIfUnselected {
			fade = 1.0
		}

		nameScale := calcClick(item.NameClickTimer)
		valueScale := calcClick(item.ValueClickTimer)
		leftArrowScale := calcArrowClick(item.LeftArrowClickTimer)
		rightArrowScale := calcArrowClick(item.RightArrowClickTimer)

		// ==========================
		// draw name
		// ==========================
		{
			renderedWidth := drawText(item.Name, FontBold, size, nameScale, fadeC(item.Color, fade))
			xAdvance += max(renderedWidth, item.NameMinWidth)

			if item.NameValueSeperator == "" {
				xAdvance += 40
			} else {
				xAdvance += 20
				xAdvance += drawText(item.NameValueSeperator, FontBold, size, 1, fadeC(item.Color, fade))
				xAdvance += 40
			}
		}

		if item.Type == MenuItemToggle && item.ToggleStyleCheckBox {
			// ==========================
			// draw toggle check box
			// ==========================
			checkBoxScale := float32(1.2)

			checkBoxOffsetX := float32(0)
			checkBoxOffsetY := -size * 0.1

			xDrawOffset = checkBoxOffsetX
			yDrawOffset = checkBoxOffsetY

			boxRect := rl.Rectangle{
				X: 0, Y: 0,
				Width: f32(CheckBoxBox.Width), Height: f32(CheckBoxBox.Height),
			}

			if item.BValue {
				drawImage(CheckBoxBox, boxRect, size, checkBoxScale, dimmC(item.CheckedBoxColor, fade))
			} else {
				drawImage(CheckBoxBox, boxRect, size, checkBoxScale, dimmC(item.UncheckedBoxColor, fade))
			}

			if item.BValue {
				const animDuration = time.Millisecond * 200

				delta := TimeSinceNow(item.ValueClickTimer)

				t := f32(delta) / f32(animDuration)
				t = Clamp(t, 0, 1)

				spriteN := int(f32(CheckBoxMark.Count) * t)

				if spriteN >= CheckBoxMark.Count {
					spriteN = CheckBoxMark.Count - 1
				}

				drawSprite(CheckBoxMark, spriteN, size, checkBoxScale, dimmC(item.CheckmarkColor, fade))
			}

			xDrawOffset = 0
			yDrawOffset = 0
		} else if item.Type == MenuItemKey {
			// ==========================
			// draw kew binding item
			// ==========================
			for i, key := range item.KeyValues {
				keyName := GetKeyName(key)

				keyScale := float32(0.9)
				keyColor := item.KeyColorRegular

				desiredWidth := item.SizeRegular * 4

				if i == md.keySelected() && index == md.SelectedIndex {
					const animDuration = time.Millisecond * 70
					t := f32(TimeSinceNow(item.KeySelectTimer)) / f32(animDuration)
					t = Clamp(t, 0, 1)

					keyScale = Lerp(0.9, 1, t)
					keyColor = LerpRGBA(item.KeyColorRegular, item.KeyColorSelected, f64(t))

					keyScale *= calcClick(item.ValueClickTimer)
				}

				drawStrikeThrough := md.InputState == MenuInputStateWaitingKeyPress
				drawStrikeThrough = drawStrikeThrough && i == md.keySelected()
				drawStrikeThrough = drawStrikeThrough && index == md.SelectedIndex

				if drawStrikeThrough {
					keyNameSize := rl.MeasureTextEx(FontBold, keyName, size, 0)

					keyNameRect := rl.Rectangle{
						Width:  max(desiredWidth, keyNameSize.X),
						Height: keyNameSize.Y,
					}
					keyNameRect.X = xAdvance
					keyNameRect.Y = yCenter - keyNameRect.Height*0.5

					keyNameCenter := RectCenter(keyNameRect)

					strikeRect := rl.Rectangle{}
					strikeRect.Width = keyNameSize.X * 0.8 * keyScale
					strikeRect.Height = size * 0.1 * keyScale
					strikeRect = RectCenetered(strikeRect, keyNameCenter.X, keyNameCenter.Y)

					rl.DrawRectangleRounded(strikeRect, 1, 7, keyColor.ToRlColor())

					xAdvance += max(desiredWidth, keyNameSize.X)
				} else {
					keyColor = fadeC(keyColor, fade)

					xAdvance += drawTextCentered(keyName, FontBold, size, keyScale, desiredWidth, keyColor)
				}

				xAdvance += 30
			}
		} else {
			// =====================================
			// draw items with < value > style item
			// =====================================
			switch item.Type {
			case MenuItemToggle, MenuItemList, MenuItemNumber:
				arrowFill := fadeC(Col(1, 1, 1, 1), fade)
				arrowStroke := Col(0, 0, 0, 1)

				if index != md.SelectedIndex {
					arrowStroke = Color{}
				}

				xAdvance += drawArrow(true, size, leftArrowScale, arrowFill, arrowStroke)

				xAdvance += 10 // <- 10 value 10 ->

				valueWidthMax := float32(0)

				switch item.Type {
				case MenuItemToggle:
					valueWidthMax = rl.MeasureTextEx(FontBold, "yes", size, 0).X
				case MenuItemList:
					for _, entry := range item.List {
						valueWidthMax = max(rl.MeasureTextEx(FontBold, entry, size, 0).X, valueWidthMax)
					}
				case MenuItemNumber:
					minText := fmt.Sprintf(item.NValueFmtString, item.NValueMin)
					maxText := fmt.Sprintf(item.NValueFmtString, item.NValueMax)
					valueWidthMax = max(rl.MeasureTextEx(FontBold, minText, size, 0).X, valueWidthMax)
					valueWidthMax = max(rl.MeasureTextEx(FontBold, maxText, size, 0).X, valueWidthMax)
				}

				switch item.Type {
				case MenuItemToggle:
					if item.BValue {
						drawTextCentered("Yes", FontBold, size, valueScale, valueWidthMax, fadeC(item.Color, fade))
					} else {
						drawTextCentered("No", FontBold, size, valueScale, valueWidthMax, fadeC(item.Color, fade))
					}
				case MenuItemList:
					drawTextCentered(item.List[item.ListSelected], FontBold, size, valueScale, valueWidthMax, fadeC(item.Color, fade))
				case MenuItemNumber:
					toDraw := fmt.Sprintf(item.NValueFmtString, item.NValue)
					drawTextCentered(toDraw, FontBold, size, valueScale, valueWidthMax, fadeC(item.Color, fade))
				}

				xAdvance += valueWidthMax
				xAdvance += 10 // <- 10 value 10 ->

				drawArrow(false, size, rightArrowScale, arrowFill, arrowStroke)
			}
		}

		yOffset += item.SizeRegular + item.BottomMargin

		// update item's rendered rect
		item.bound = itemBound
		itemBoundSet = false
	}
}

func (md *MenuDrawer) GetSelectedItem() *MenuItem {
	if len(md.items) <= 0 {
		return nil
	}
	item := md.items[md.SelectedIndex]
	if item.IsSelectable() {
		return item
	}
	return nil
}

func (md *MenuDrawer) GetSelectedId() MenuItemId {
	if len(md.items) <= 0 {
		return 0
	}
	item := md.items[md.SelectedIndex]
	if item.IsSelectable() {
		return item.Id
	}
	return 0
}

func (md *MenuDrawer) GetUserData(id MenuItemId) any {
	item := md.GetItemById(id)
	if item == nil {
		return nil
	}

	return item.UserData
}

func (md *MenuDrawer) SearchItem(searchFunc func(item *MenuItem) bool) MenuItemId {
	for _, item := range md.items {
		if searchFunc(item) {
			return item.Id
		}
	}

	return 0
}

func (md *MenuDrawer) GetItemById(id MenuItemId) *MenuItem {
	for _, item := range md.items {
		if item.Id == id {
			return item
		}
	}

	return nil
}

func (md *MenuDrawer) AddItems(items ...*MenuItem) {
	for _, item := range items {
		if item != nil {
			md.items = append(md.items, item)
		}
	}
}

func (md *MenuDrawer) InsertAt(at int, items ...*MenuItem) {
	at = Clamp(at, 0, len(md.items))

	var newItems []*MenuItem

	newItems = append(newItems, md.items[0:at]...)
	newItems = append(newItems, items...)
	newItems = append(newItems, md.items[at:]...)

	md.items = newItems
}

func (md *MenuDrawer) DeleteItems(ids ...MenuItemId) {
	md.items = slices.DeleteFunc(md.items, func(item *MenuItem) bool {
		for _, id := range ids {
			if item.Id == id {
				return true
			}
		}
		return false
	})
}

func (md *MenuDrawer) DeleteItemsAt(indices ...int) {
	var newItems []*MenuItem

	for i, item := range md.items {
		if !slices.Contains(indices, i) {
			newItems = append(newItems, item)
		}
	}

	md.items = newItems
}

func (md *MenuDrawer) DeleteFunc(del func(*MenuItem) bool) {
	md.items = slices.DeleteFunc(md.items, del)
}

func (md *MenuDrawer) ClearItems() {
	md.items = md.items[:0]
}

func (md *MenuDrawer) IsItemHidden(id MenuItemId) bool {
	item := md.GetItemById(id)
	if item == nil {
		// NOTE : I think returning false is better since it's the default value
		return false
	}

	return item.IsHidden
}

func (md *MenuDrawer) SetItemHidden(id MenuItemId, hidden bool) {
	item := md.GetItemById(id)
	if item != nil {
		item.IsHidden = hidden
	}
}

// Sets item BValue.
// Doesn't trigger item callback
func (md *MenuDrawer) SetItemBValue(id MenuItemId, bValue bool) {
	item := md.GetItemById(id)
	if item != nil {
		if item.BValue != bValue {
			item.BValue = bValue

			// Trigger item click animation if necessary
			if item.Type == MenuItemTrigger {
				item.NameClickTimer = GlobalTimerNow()
			} else if item.Type == MenuItemToggle {
				item.ValueClickTimer = GlobalTimerNow()
			}
		}
	}
}

func (md *MenuDrawer) GetItemBValue(id MenuItemId) bool {
	item := md.GetItemById(id)
	if item != nil {
		return item.BValue
	}

	return false
}

// Sets item NValue.
// Doesn't trigger item callback
func (md *MenuDrawer) SetItemNvalue(id MenuItemId, nValue float32) {
	item := md.GetItemById(id)
	if item != nil {
		prevValue := item.NValue
		item.NValue = nValue

		if math.Abs(f64(nValue-prevValue)) > 0.0001 && // epsilon fresh from my ass
			item.Type == MenuItemNumber {

			item.ValueClickTimer = GlobalTimerNow()
		}
	}
}

func (md *MenuDrawer) GetItemNValue(id MenuItemId) float32 {
	item := md.GetItemById(id)
	if item != nil {
		return item.NValue
	}

	return 0
}

// Sets item ListSelected.
// Doesn't trigger item callback
func (md *MenuDrawer) SetItemListSelected(id MenuItemId, selected int) {
	item := md.GetItemById(id)
	if item != nil && len(item.List) > 0 {
		selected = Clamp(selected, 0, len(item.List)-1)

		if selected != item.ListSelected && item.Type == MenuItemList {
			item.ListSelected = selected
			item.ValueClickTimer = GlobalTimerNow()
		}
	}
}

func (md *MenuDrawer) GetItemListSelected(id MenuItemId) (index int, selected string) {
	item := md.GetItemById(id)

	if item != nil && len(item.List) > 0 {
		index = Clamp(item.ListSelected, 0, len(item.List)-1)
		selected = item.List[index]
	} else {
		index, selected = 0, ""
	}

	return
}

// Sets item List.
// Doesn't trigger item callback and animation
func (md *MenuDrawer) SetItemList(id MenuItemId, list []string, selected int) {
	item := md.GetItemById(id)

	if item != nil {
		if len(list) > 0 {
			selected = Clamp(selected, 0, len(list)-1)
			item.List = list
			item.ListSelected = selected
		}
	}
}

func (md *MenuDrawer) GetItemBound(id MenuItemId) (rl.Rectangle, bool) {
	item := md.GetItemById(id)

	if item != nil {
		return item.bound, true
	}

	return rl.Rectangle{}, false
}

func (md *MenuDrawer) ResetAnimation() {
	md.ScrollAnimT = 1
}
