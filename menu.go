package main

import (
	"fmt"
	"math"
	"slices"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ===============================
// MenuItem stuffs
// ===============================

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

	Color       Color
	StrokeColor Color
	StrokeWidth float32

	// transparency when it's unselected
	Fade             float64
	FadeIfUnselected bool

	// margin between next item
	BottomMargin float32

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

	CheckedBoxColor   Color
	UncheckedBoxColor Color

	CheckmarkColor Color

	KeyColorRegular  Color
	KeyColorSelected Color

	KeyColorStrokeRegular  Color
	KeyColorStrokeSelected Color
	KeyStrokeWidthRegular  float32
	KeyStrokeWidthSelected float32

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
	SizeSelected: 80,

	Color: Col(0, 0, 0, 1),

	Fade:             0.35,
	FadeIfUnselected: true,

	ToggleStyleCheckBox: true,

	BottomMargin:       30,
	SelectedLeftMargin: 10,

	CheckedBoxColor:   Color255(0x79, 0xE4, 0xAF, 0xFF),
	UncheckedBoxColor: Color255(0xD1, 0xD1, 0xD1, 0xFF),

	CheckmarkColor: Color255(0xFF, 0xFF, 0xFF, 0xFF),

	KeyColorRegular:  Color255(0x00, 0x00, 0x00, 200),
	KeyColorSelected: Color255(0xFF, 0xFF, 0xFF, 0xFF),

	KeyColorStrokeSelected: Color255(0, 0, 0, 0xFF),

	KeyStrokeWidthSelected: 10,
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

// ===============================
// MenuManger stuffs
// ===============================

var TheMenuResources struct {
	CheckBoxRenderTex rl.RenderTexture2D
	UIarrowRenderTex  rl.RenderTexture2D
}

func InitMenuResources() {
	tm := &TheMenuResources

	cbw := max(i32(CheckBoxMark.Width), CheckBoxBox.Width)
	cbh := max(i32(CheckBoxMark.Height), CheckBoxBox.Height)

	tm.CheckBoxRenderTex = rl.LoadRenderTexture(cbw, cbh)

	tm.UIarrowRenderTex = rl.LoadRenderTexture(i32(UIarrowsSprite.Width), i32(UIarrowsSprite.Height))
}

func FreeMenuResources() {
	tm := &TheMenuResources

	rl.UnloadRenderTexture(tm.CheckBoxRenderTex)
	rl.UnloadRenderTexture(tm.UIarrowRenderTex)
}

func UpdateMenuManager(deltaTime time.Duration) {
	tm := &TheMenuResources

	cbw := max(i32(CheckBoxMark.Width), CheckBoxBox.Width)
	cbh := max(i32(CheckBoxMark.Height), CheckBoxBox.Height)

	if cbw != tm.CheckBoxRenderTex.Texture.Width || cbh != tm.CheckBoxRenderTex.Texture.Height {
		rl.UnloadRenderTexture(tm.CheckBoxRenderTex)
		tm.CheckBoxRenderTex = rl.LoadRenderTexture(cbw, cbh)
	}

	if i32(UIarrowsSprite.Width) != tm.UIarrowRenderTex.Texture.Width || i32(UIarrowsSprite.Height) != tm.UIarrowRenderTex.Texture.Height {
		rl.UnloadRenderTexture(tm.UIarrowRenderTex)
		tm.UIarrowRenderTex = rl.LoadRenderTexture(i32(UIarrowsSprite.Width), i32(UIarrowsSprite.Height))
	}
}

func getCheckBoxTextureWH() (float32, float32) {
	tm := &TheMenuResources
	return f32(tm.CheckBoxRenderTex.Texture.Width), f32(tm.CheckBoxRenderTex.Texture.Height)
}

// Get check box texture drawn with specified colors.
func getCheckBoxTexture(checked bool, spriteN int, boxColor, markColor Color) rl.Texture2D {
	tm := &TheMenuResources

	flipY := rl.MatrixIdentity()
	flipY = rl.MatrixMultiply(flipY, rl.MatrixScale(1, -1, 1))
	flipY = rl.MatrixMultiply(
		flipY,
		rl.MatrixTranslate(0, f32(tm.CheckBoxRenderTex.Texture.Height), 0),
	)

	FnfBeginTextureMode(tm.CheckBoxRenderTex)
	rl.BeginBlendMode(rl.BlendAlphaPremultiply)

	rl.ClearBackground(rl.Color{0, 0, 0, 0})

	DrawTextureTransfromed(
		CheckBoxBox,
		rl.Rectangle{0, 0, f32(CheckBoxBox.Width), f32(CheckBoxBox.Height)},
		flipY,
		boxColor.ToImageRGBA())

	if checked {
		DrawSpriteTransfromed(
			CheckBoxMark, spriteN,
			rl.Rectangle{0, 0, CheckBoxMark.Width, CheckBoxMark.Height},
			flipY,
			markColor.ToImageRGBA())
	}

	rl.EndBlendMode()
	FnfEndTextureMode()

	return tm.CheckBoxRenderTex.Texture
}

func getUIarrowsTextureWH() (float32, float32) {
	tm := &TheMenuResources
	return f32(tm.UIarrowRenderTex.Texture.Width), f32(tm.UIarrowRenderTex.Texture.Height)
}

func getUIarrowsTexture(drawLeft bool, fill, stroke Color) rl.Texture2D {
	tm := &TheMenuResources

	fillSpriteN := UIarrowRightFill
	strokeSpriteN := UIarrowRightStroke

	if drawLeft {
		fillSpriteN = UIarrowLeftFill
		strokeSpriteN = UIarrowLeftStroke
	}

	flipY := rl.MatrixIdentity()
	flipY = rl.MatrixMultiply(flipY, rl.MatrixScale(1, -1, 1))
	flipY = rl.MatrixMultiply(
		flipY,
		rl.MatrixTranslate(0, f32(tm.UIarrowRenderTex.Texture.Height), 0),
	)

	FnfBeginTextureMode(tm.UIarrowRenderTex)
	rl.BeginBlendMode(rl.BlendAlphaPremultiply)

	rl.ClearBackground(rl.Color{0, 0, 0, 0})

	DrawSpriteTransfromed(
		UIarrowsSprite, fillSpriteN,
		RectWH(UIarrowsSprite.Width, UIarrowsSprite.Height),
		flipY,
		fill.ToImageRGBA())

	DrawSpriteTransfromed(
		UIarrowsSprite, strokeSpriteN,
		RectWH(UIarrowsSprite.Width, UIarrowsSprite.Height),
		flipY,
		stroke.ToImageRGBA())

	rl.EndBlendMode()
	FnfEndTextureMode()

	return tm.UIarrowRenderTex.Texture
}

// ===============================
// MenuDrawer stuffs
// ===============================

const (
	MenuInputStateNotSelectingKey = iota
	MenuInputStateWaitingKeyPress
	MenuInputStateWaitingKeyRelease
)

type MenuBackground struct {
	Texture rl.Texture2D
	OffsetX float32
	OffsetY float32
}

type MenuDrawer struct {
	InputId InputGroupId

	selectedIndex int

	yOffset float32

	scrollAnimT float32

	inputState int

	keySelectedIndex int

	items []*MenuItem
}

func NewMenuDrawer() *MenuDrawer {
	md := new(MenuDrawer)

	md.scrollAnimT = 1

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

	prevSelected := md.selectedIndex

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
				md.selectedIndex += 1
			} else {
				md.selectedIndex -= 1
			}

			if md.selectedIndex >= len(md.items) {
				md.selectedIndex = 0
			} else if md.selectedIndex < 0 {
				md.selectedIndex = len(md.items) - 1
			}

			if md.items[md.selectedIndex].IsSelectable() {
				break
			}
		}
	}

	if !noSelectable {
		if !md.items[md.selectedIndex].IsSelectable() {
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
	if md.inputState == MenuInputStateWaitingKeyPress {
		if pressed, key := AnyKeyPressed(md.InputId); pressed {
			selected := md.items[md.selectedIndex]

			if selected.Type == MenuItemKey {
				keySelected := md.keySelected()
				if len(selected.KeyValues) > 0 && selected.KeyCallback != nil {
					prevKey := selected.KeyValues[keySelected]
					newKey := key
					itemCallback = func() {
						selected.KeyCallback(keySelected, prevKey, newKey)
					}
				}
			} else {
				ErrorLogger.Fatalf("wrong type of MenuItem : %v", MenuItemTypeName(selected.Type))
			}

			md.inputState = MenuInputStateWaitingKeyRelease
		}
	} else if md.inputState == MenuInputStateWaitingKeyRelease {
		var menuKeys []int32
		menuKeys = append(menuKeys, TheKM[SelectKey], TheKM[EscapeKey])
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			menuKeys = append(menuKeys, NoteKeys(dir)...)
		}

		if !AreKeysDown(md.InputId, menuKeys...) {
			if IsInputSoloEnabled(md.InputId) {
				ClearSoloInput()
			}
			md.inputState = MenuInputStateNotSelectingKey
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
			selected := md.items[md.selectedIndex]

			// ===================================
			// handle select key interaction
			// ===================================
			if AreKeysPressed(md.InputId, TheKM[SelectKey]) {
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
						md.inputState = MenuInputStateWaitingKeyPress
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

	if md.selectedIndex != prevSelected {
		md.scrollAnimT = 0
	}

	// but I have a strong feeling that this is not frame indipendent
	// but it's just for menu so I don't think it matters too much...
	selected := md.items[md.selectedIndex]

	selectionY := md.calculateSelectionY(md.selectedIndex)

	blend := Clamp(float32(deltaTime.Seconds()*20), 0.00, 1.0)

	if tryingToMove && canNotMove {
		push := (selected.SizeRegular*0.5 + 30) * 0.8
		if tryingToMoveUp {
			selectionY += push
		} else {
			selectionY -= push
		}
	}

	md.yOffset = Lerp(md.yOffset, selectionY, blend)

	md.scrollAnimT = Lerp(md.scrollAnimT, 1.0, blend)

	// ================================
	// actually call item callback
	// ================================
	if itemCallback != nil {
		itemCallback()
	}
}

// calculate yOffset if item at index is selected
func (md *MenuDrawer) calculateSelectionY(index int) float32 {
	if len(md.items) <= 0 {
		return float32(SCREEN_HEIGHT * 0.5)
	}

	index = Clamp(index, 0, len(md.items))

	selected := md.items[index]

	selectionY := float32(SCREEN_HEIGHT * 0.5)
	selectionY -= selected.SizeRegular * 0.5

	for index, item := range md.items {
		if index >= md.selectedIndex {
			break
		}

		if item.IsHidden {
			continue
		}

		selectionY -= item.SizeRegular + item.BottomMargin
	}

	return selectionY
}

func (md *MenuDrawer) Draw() {
	if len(md.items) <= 0 {
		return
	}

	if DrawDebugGraphics {
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

	yOffset := md.yOffset
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

	fadeC := func(col Color, fade float64) Color {
		col.A *= fade
		return col
	}

	screenRect := GetScreenRect()

	drawText := func(text string, fontSize, scale float32, fill, stroke Color, strokeWidth float32) float32 {
		textSize := rl.MeasureTextEx(FontBold, text, fontSize, 0)

		pos := rl.Vector2{
			X: xAdvance + textSize.X*0.5*(1-scale),
			Y: yCenter - textSize.Y*scale*0.5,
		}

		pos.X += xDrawOffset
		pos.Y += yDrawOffset

		bound := rl.Rectangle{
			X: pos.X, Y: pos.Y,
			Width: textSize.X * scale, Height: textSize.Y * scale,
		}
		updateItemBound(bound)

		if !rl.CheckCollisionRecs(screenRect, bound) {
			return textSize.X
		}

		if strokeWidth <= 0 {
			rl.DrawTextEx(FontBold, text, pos, fontSize*scale, 0, fill.ToRlColor())
		} else {
			DrawTextSdfOutlined(
				SdfFontBold, text, pos, fontSize*scale, 0,
				fill.ToImageRGBA(), stroke.ToImageRGBA(),
				strokeWidth,
			)
		}

		return textSize.X
	}

	drawTextCentered := func(
		text string, fontSize, scale, width float32, fill, stroke Color, strokeWidth float32) float32 {

		textSize := rl.MeasureTextEx(FontBold, text, fontSize, 0)

		width = max(textSize.X, width)

		pos := rl.Vector2{
			X: xAdvance + (width-textSize.X*scale)*0.5,
			Y: yCenter - textSize.Y*scale*0.5,
		}

		pos.X += xDrawOffset
		pos.Y += yDrawOffset

		bound := rl.Rectangle{
			X: pos.X, Y: pos.Y,
			Width: textSize.X * scale, Height: textSize.Y * scale,
		}

		updateItemBound(bound)

		if !rl.CheckCollisionRecs(screenRect, bound) {
			return width
		}

		if strokeWidth <= 0 {
			rl.DrawTextEx(FontBold, text, pos, fontSize*scale, 0, fill.ToRlColor())
		} else {
			DrawTextSdfOutlined(
				SdfFontBold, text, pos, fontSize*scale, 0,
				fill.ToImageRGBA(), stroke.ToImageRGBA(),
				strokeWidth,
			)
		}

		return width
	}

	drawCheck := func(imgW, imgH, height, scale float32) (rl.Rectangle, bool) {
		wScale := height / imgH

		dstRect := rl.Rectangle{
			X: xAdvance, Y: yCenter - height*0.5*scale,
			Width: wScale * imgW * scale, Height: height * scale,
		}

		dstRect.X += xDrawOffset
		dstRect.Y += yDrawOffset

		updateItemBound(dstRect)

		return dstRect, rl.CheckCollisionRecs(screenRect, dstRect)
	}

	drawImage := func(
		img rl.Texture2D, srcRect rl.Rectangle, height, scale float32, col rl.Color) float32 {

		rect, draw := drawCheck(srcRect.Width, srcRect.Height, height, scale)

		if draw {
			rl.DrawTexturePro(img, srcRect, rect, rl.Vector2{}, 0, col)
		}

		return rect.Width
	}

	drawArrow := func(
		drawLeft bool, height, scale float32, fill, stroke Color, alpha float64) float32 {

		w, h := getUIarrowsTextureWH()

		if rect, draw := drawCheck(w, h, height, scale); !draw {
			return rect.Width
		}

		arrowTex := getUIarrowsTexture(drawLeft, fill, stroke)

		return drawImage(
			arrowTex, RectWH(arrowTex.Width, arrowTex.Height), height, scale, Col(1, 1, 1, alpha).ToRlColor(),
		)
	}

	drawCheckBox := func(
		checked bool, spriteN int, height, scale float32, boxColor, markColor Color, alpha float64) float32 {

		w, h := getCheckBoxTextureWH()

		if rect, draw := drawCheck(w, h, height, scale); !draw {
			return rect.Width
		}

		checkBoxTex := getCheckBoxTexture(checked, spriteN, boxColor, markColor)
		return drawImage(
			checkBoxTex, RectWH(checkBoxTex.Width, checkBoxTex.Height),
			height, scale,
			Col(1, 1, 1, alpha).ToRlColor(),
		)
	}

	for index, item := range md.items {
		if item.IsHidden {
			continue
		}

		yCenter = yOffset + item.SizeRegular*0.5

		xAdvance = xOffset

		fade := item.Fade
		size := item.SizeRegular

		if index == md.selectedIndex {
			fade = Lerp(item.Fade, 1.0, float64(md.scrollAnimT))
			size = Lerp(item.SizeRegular, item.SizeSelected, md.scrollAnimT)
			xAdvance += Lerp(0, item.SelectedLeftMargin, md.scrollAnimT)
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
			renderedWidth := drawText(
				item.Name, size, nameScale, fadeC(item.Color, fade), fadeC(item.StrokeColor, fade), item.StrokeWidth)

			xAdvance += max(renderedWidth, item.NameMinWidth)

			if item.NameValueSeperator == "" {
				xAdvance += 40
			} else {
				xAdvance += 20
				xAdvance += drawText(
					item.NameValueSeperator, size, 1, fadeC(item.Color, fade), fadeC(item.StrokeColor, fade), item.StrokeWidth)
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

			boxColor := item.UncheckedBoxColor
			if item.BValue {
				boxColor = item.CheckedBoxColor
			}

			const animDuration = time.Millisecond * 200

			delta := TimeSinceNow(item.ValueClickTimer)

			t := f32(delta) / f32(animDuration)
			t = Clamp(t, 0, 1)

			spriteN := int(f32(CheckBoxMark.Count) * t)

			if spriteN >= CheckBoxMark.Count {
				spriteN = CheckBoxMark.Count - 1
			}

			drawCheckBox(item.BValue, spriteN, size, checkBoxScale, boxColor, item.CheckmarkColor, fade)

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
				keyColorStroke := item.KeyColorStrokeRegular
				keyStrokeWidth := item.KeyStrokeWidthRegular

				desiredWidth := item.SizeRegular * 4

				if i == md.keySelected() && index == md.selectedIndex {
					const animDuration = time.Millisecond * 70
					t := f32(TimeSinceNow(item.KeySelectTimer)) / f32(animDuration)
					t = Clamp(t, 0, 1)

					keyScale = Lerp(0.9, 1, t)
					keyColor = LerpRGBA(item.KeyColorRegular, item.KeyColorSelected, f64(t))
					keyStrokeWidth = Lerp(item.KeyStrokeWidthRegular, item.KeyStrokeWidthSelected, t)
					keyColorStroke = LerpRGBA(item.KeyColorStrokeRegular, item.KeyColorStrokeSelected, f64(t))

					keyScale *= calcClick(item.ValueClickTimer)
				}

				drawStrikeThrough := md.inputState == MenuInputStateWaitingKeyPress
				drawStrikeThrough = drawStrikeThrough && i == md.keySelected()
				drawStrikeThrough = drawStrikeThrough && index == md.selectedIndex

				if drawStrikeThrough {
					// ==========================
					// draw key strike through
					// ==========================
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

					if keyStrokeWidth > 0.5 {
						rl.DrawRectangleRoundedLines(strikeRect, 1, 7, keyStrokeWidth, keyColorStroke.ToRlColor())
					}

					rl.DrawRectangleRounded(strikeRect, 1, 7, keyColor.ToRlColor())
					updateItemBound(strikeRect)

					xAdvance += max(desiredWidth, keyNameSize.X)
				} else {
					xAdvance += drawTextCentered(keyName, size, keyScale, desiredWidth,
						fadeC(keyColor, fade), fadeC(keyColorStroke, fade), keyStrokeWidth)
				}

				xAdvance += 30
			}
		} else {
			// =====================================
			// draw items with < value > style item
			// =====================================
			switch item.Type {
			case MenuItemToggle, MenuItemList, MenuItemNumber:
				arrowFill := Col(1, 1, 1, 1)
				arrowStroke := Col(0, 0, 0, 1)

				xAdvance += drawArrow(true, size, leftArrowScale, arrowFill, arrowStroke, fade)

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
						drawTextCentered("Yes", size, valueScale, valueWidthMax,
							fadeC(item.Color, fade), fadeC(item.StrokeColor, fade), item.StrokeWidth)
					} else {
						drawTextCentered("No", size, valueScale, valueWidthMax,
							fadeC(item.Color, fade), fadeC(item.StrokeColor, fade), item.StrokeWidth)
					}
				case MenuItemList:
					drawTextCentered(item.List[item.ListSelected], size, valueScale, valueWidthMax,
						fadeC(item.Color, fade), fadeC(item.StrokeColor, fade), item.StrokeWidth)
				case MenuItemNumber:
					toDraw := fmt.Sprintf(item.NValueFmtString, item.NValue)
					drawTextCentered(toDraw, size, valueScale, valueWidthMax,
						fadeC(item.Color, fade), fadeC(item.StrokeColor, fade), item.StrokeWidth)
				}

				xAdvance += valueWidthMax
				xAdvance += 10 // <- 10 value 10 ->

				drawArrow(false, size, rightArrowScale, arrowFill, arrowStroke, fade)
			}
		}

		yOffset += item.SizeRegular + item.BottomMargin

		// update item's rendered rect
		item.bound = itemBound
		itemBoundSet = false
	}
}

// Try to select the item at index.
// If no item at, per say, 0 is unselectable, tries to select next and next and so on.
// Returns -1, 0 if no item can be selected.
// Else returns actually selected index and id.
//
// Set playScrollAnimation to control whether menu scrolls towards selected item
// or just jumps with out any animation.
func (md *MenuDrawer) SelectItemAt(index int, playScrollAnimation bool) (int, MenuItemId) {
	selectedIndex := -1
	var selectedId MenuItemId

	for i, item := range md.items {
		if i >= index && item.IsSelectable() {
			selectedIndex = i
			selectedId = item.Id

			md.selectedIndex = i

			if playScrollAnimation {
				md.scrollAnimT = 0
			} else {
				md.scrollAnimT = 1
				md.yOffset = md.calculateSelectionY(md.selectedIndex)
			}

			break
		}
	}

	return selectedIndex, selectedId
}

func (md *MenuDrawer) SelectedIndex() int {
	return md.selectedIndex
}

func (md *MenuDrawer) GetSelectedItem() *MenuItem {
	if len(md.items) <= 0 {
		return nil
	}
	item := md.items[md.selectedIndex]
	if item.IsSelectable() {
		return item
	}
	return nil
}

func (md *MenuDrawer) GetSelectedId() MenuItemId {
	if len(md.items) <= 0 {
		return 0
	}
	item := md.items[md.selectedIndex]
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

func (md *MenuDrawer) BeforeScreenTransition() {
	md.scrollAnimT = 1
	md.yOffset = md.calculateSelectionY(md.selectedIndex)
}

func (md *MenuDrawer) Free() {
	// pass
}
