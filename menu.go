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
	MenuItemDeco
	MenuItemTypeSize
)

var MenuItemTypeStrs [MenuItemTypeSize]string

func init() {
	MenuItemTypeStrs[MenuItemTrigger] = "Trigger"
	MenuItemTypeStrs[MenuItemToggle] = "Toggle"
	MenuItemTypeStrs[MenuItemNumber] = "Number"
	MenuItemTypeStrs[MenuItemList] = "List"
	MenuItemTypeStrs[MenuItemDeco] = "Deco"
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

	BValue bool

	NValue float32

	NValueMin      float32
	NValueMax      float32
	NValueInterval float32

	ListSelected int
	List         []string

	OnValueChange func(bValue bool, nValue float32, listSelection string)

	UserData any

	IsHidden bool

	// format string to use to displat NValue
	NValueFmtString string

	// whether if toggle item will use checkbox or < yes, no >
	ToggleStyleCheckBox bool

	CheckedBoxColor   Color // default is 0x79 E4 AF FF
	UncheckedBoxColor Color // default is 0xD1 D1 D1 FF

	CheckmarkColor Color // default is 0xFF FF FF FF

	// variables for animations
	NameClickTimer       time.Duration
	ValueClickTimer      time.Duration
	LeftArrowClickTimer  time.Duration
	RightArrowClickTimer time.Duration

	bound rl.Rectangle
}

var menuItemIdGenerator IdGenerator[MenuItemId]

const (
	MenuItemSizeRegularDefault  = 70
	MenuItemSizeSelectedDefault = 90
)

func NewMenuItem() *MenuItem {
	item := new(MenuItem)

	item.Id = menuItemIdGenerator.NewId()

	item.SizeRegular = MenuItemSizeRegularDefault
	item.SizeSelected = MenuItemSizeSelectedDefault

	item.NameClickTimer = -Years150
	item.ValueClickTimer = -Years150
	item.LeftArrowClickTimer = -Years150
	item.RightArrowClickTimer = -Years150

	item.Color = Col(1, 1, 1, 1)

	item.FadeIfUnselected = true

	item.ToggleStyleCheckBox = true

	item.CheckedBoxColor = Color255(0x79, 0xE4, 0xAF, 0xFF)
	item.UncheckedBoxColor = Color255(0xD1, 0xD1, 0xD1, 0xFF)

	item.CheckmarkColor = Color255(0xFF, 0xFF, 0xFF, 0xFF)

	return item
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

type MenuDrawer struct {
	SelectedIndex int

	ListInterval float32

	Yoffset float32

	ScrollAnimT float32

	InputId InputGroupId

	items []*MenuItem
}

func NewMenuDrawer() *MenuDrawer {
	md := new(MenuDrawer)

	md.ScrollAnimT = 1

	md.ListInterval = 30

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
	{
		callItemCallback := func(item *MenuItem) {
			if item.OnValueChange != nil {

				// NOTE : we don't actually wanna call item callback now
				// we will call it when update is over
				itemCallback = func() {
					listSelection := ""
					if 0 <= item.ListSelected && item.ListSelected < len(item.List) {
						listSelection = item.List[item.ListSelected]
					}
					item.OnValueChange(item.BValue, item.NValue, listSelection)
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

			if AreKeysPressed(md.InputId, TheKM.SelectKey) {
				switch selected.Type {
				case MenuItemTrigger:
					selected.BValue = true
					selected.NameClickTimer = GlobalTimerNow()
				case MenuItemToggle:
					selected.BValue = !selected.BValue
					selected.ValueClickTimer = GlobalTimerNow()
				}

				callItemCallback(selected)
			}

			switch selected.Type {
			case MenuItemList, MenuItemNumber, MenuItemToggle:
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
				}

				if AreKeysDown(md.InputId, NoteKeys(NoteDirLeft)...) && canGoLeft {
					selected.LeftArrowClickTimer = GlobalTimerNow()
				}

				if AreKeysDown(md.InputId, NoteKeys(NoteDirRight)...) && canGoRight {
					selected.RightArrowClickTimer = GlobalTimerNow()
				}

				goLeft := false
				goRight := false

				const firstRate = time.Millisecond * 200
				const repeateRate = time.Millisecond * 110

				goLeft = HandleKeyRepeat(md.InputId, firstRate, repeateRate, NoteKeys(NoteDirLeft)...) && canGoLeft
				goRight = HandleKeyRepeat(md.InputId, firstRate, repeateRate, NoteKeys(NoteDirRight)...) && canGoRight

				switch selected.Type {
				case MenuItemToggle:
					if goLeft || goRight {
						selected.BValue = !selected.BValue
						callItemCallback(selected)
					}
				case MenuItemList:
					if len(selected.List) > 0 {
						listSelected := selected.ListSelected

						if goLeft && canGoLeft {
							listSelected -= 1
						} else if goRight && canGoRight {
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
					if goLeft {
						selected.NValue -= selected.NValueInterval
						callItemCallback(selected)
					} else if goRight {
						selected.NValue += selected.NValueInterval
						callItemCallback(selected)
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

		seletionY -= item.SizeRegular + md.ListInterval
	}

	if tryingToMove && canNotMove {
		push := (selected.SizeRegular*0.5 + md.ListInterval) * 0.8
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

	drawText := func(text string, fontSize, scale float32, col Color) float32 {
		textSize := rl.MeasureTextEx(FontBold, text, fontSize, 0)

		pos := rl.Vector2{
			X: xAdvance,
			Y: yCenter - textSize.Y*scale*0.5,
		}

		pos.X += xDrawOffset
		pos.Y += yDrawOffset

		rl.DrawTextEx(FontBold, text, pos, fontSize*scale, 0, col.ToRlColor())

		bound := rl.Rectangle{
			X: pos.X, Y: pos.Y,
			Width: textSize.X * scale, Height: textSize.Y * scale,
		}
		updateItemBound(bound)

		return textSize.X
	}

	drawTextCentered := func(text string, fontSize, scale, width float32, col Color) float32 {
		textSize := rl.MeasureTextEx(FontBold, text, fontSize, 0)

		pos := rl.Vector2{
			X: (width-textSize.X)*0.5 + xAdvance,
			Y: yCenter - textSize.Y*scale*0.5,
		}

		pos.X += xDrawOffset
		pos.Y += yDrawOffset

		rl.DrawTextEx(FontBold, text, pos, fontSize*scale, 0, col.ToRlColor())

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
			xAdvance += Lerp(0, 30, md.ScrollAnimT)
		}

		if !item.FadeIfUnselected {
			fade = 1.0
		}

		nameScale := calcClick(item.NameClickTimer)
		valueScale := calcClick(item.ValueClickTimer)
		leftArrowScale := calcArrowClick(item.LeftArrowClickTimer)
		rightArrowScale := calcArrowClick(item.RightArrowClickTimer)

		xAdvance += drawText(item.Name, size, nameScale, fadeC(item.Color, fade))
		xAdvance += 40

		if item.Type == MenuItemToggle && item.ToggleStyleCheckBox {
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
		} else {
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
						drawTextCentered("Yes", size, valueScale, valueWidthMax, fadeC(item.Color, fade))
					} else {
						drawTextCentered("No", size, valueScale, valueWidthMax, fadeC(item.Color, fade))
					}
				case MenuItemList:
					drawTextCentered(item.List[item.ListSelected], size, valueScale, valueWidthMax, fadeC(item.Color, fade))
				case MenuItemNumber:
					toDraw := fmt.Sprintf(item.NValueFmtString, item.NValue)
					drawTextCentered(toDraw, size, valueScale, valueWidthMax, fadeC(item.Color, fade))
				}

				xAdvance += valueWidthMax
				xAdvance += 10 // <- 10 value 10 ->

				drawArrow(false, size, rightArrowScale, arrowFill, arrowStroke)
			}
		}

		yOffset += item.SizeRegular + md.ListInterval

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
