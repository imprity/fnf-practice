package main

import (
	//"math"
	"fmt"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

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

type MenuItem struct {
	Type MenuItemType

	SizeRegular  float32
	SizeSelected float32

	Color            Color
	FadeIfUnselected bool

	Id int64

	Name string

	Bvalue bool

	NValue float32

	NValueMin      float32
	NValueMax      float32
	NValueInterval float32

	NValueFmtString string

	ListSelected int
	List         []string

	// variables for animations
	NameClickTimer       time.Duration
	ValueClickTimer      time.Duration
	LeftArrowClickTimer  time.Duration
	RightArrowClickTimer time.Duration

	OnValueChange func(bValue bool, nValue float32, listSelection string)
}

var MenuItemMaxId int64
var MenuItemIdMutex sync.Mutex

const (
	MenuItemSizeRegularDefault  = 70
	MenuItemSizeSelectedDefault = 90
)

func NewMenuItem() *MenuItem {
	MenuItemIdMutex.Lock()
	defer MenuItemIdMutex.Unlock()

	item := new(MenuItem)

	MenuItemMaxId += 1

	item.Id = MenuItemMaxId

	item.SizeRegular = MenuItemSizeRegularDefault
	item.SizeSelected = MenuItemSizeSelectedDefault

	item.NameClickTimer = -Years150
	item.ValueClickTimer = -Years150
	item.LeftArrowClickTimer = -Years150
	item.RightArrowClickTimer = -Years150

	item.Color = Col(1, 1, 1, 1)

	item.FadeIfUnselected = true

	return item
}

func (mi *MenuItem) CanDecrement() bool {
	return mi.NValue-mi.NValueInterval >= mi.NValueMin-0.00001
}

func (mi *MenuItem) CanIncrement() bool {
	return mi.NValue+mi.NValueInterval <= mi.NValueMax+0.00001
}

type MenuDrawer struct {
	Items []*MenuItem

	SelectedIndex int

	ListInterval float32

	Yoffset float32

	ScrollAnimT float32

	TriggerAnimDuraiton time.Duration

	InputDisabled bool
}

func NewMenuDrawer() *MenuDrawer {
	md := new(MenuDrawer)

	md.ScrollAnimT = 1

	md.TriggerAnimDuraiton = time.Millisecond * 150

	md.ListInterval = 30

	return md
}

func (md *MenuDrawer) Update(deltaTime time.Duration) {
	if len(md.Items) <= 0 {
		return
	}

	for index, item := range md.Items {
		if item.Type == MenuItemTrigger {
			md.Items[index].Bvalue = false
		}
	}

	prevSelected := md.SelectedIndex

	allDeco := true
	nonDecoCount := 0

	for _, item := range md.Items {
		if item.Type != MenuItemDeco {
			nonDecoCount += 1
			allDeco = false
		}
	}

	scrollUntilNonDeco := func(forward bool) {
		for {
			if forward {
				md.SelectedIndex += 1
			} else {
				md.SelectedIndex -= 1
			}

			if md.SelectedIndex >= len(md.Items) {
				md.SelectedIndex = 0
			} else if md.SelectedIndex < 0 {
				md.SelectedIndex = len(md.Items) - 1
			}

			if md.Items[md.SelectedIndex].Type != MenuItemDeco {
				break
			}
		}
	}

	if !allDeco {
		if md.Items[md.SelectedIndex].Type == MenuItemDeco {
			scrollUntilNonDeco(true)
		}
	}

	tryingToMove := false
	tryingToMoveUp := false
	canNotMove := false

	if nonDecoCount <= 1 {
		canNotMove = true
	}

	// ==========================
	// handling input
	// ==========================
	if !md.InputDisabled {
		callItemCallaback := func(item *MenuItem) {
			listSelection := ""
			if 0 <= item.ListSelected && item.ListSelected < len(item.List) {
				listSelection = item.List[item.ListSelected]
			}
			if item.OnValueChange != nil {
				item.OnValueChange(item.Bvalue, item.NValue, listSelection)
			}
		}

		if AreKeysDown(NoteKeysUp...) {
			tryingToMove = true
			tryingToMoveUp = true
		}

		if AreKeysDown(NoteKeysDown...) {
			tryingToMove = true
			tryingToMoveUp = false
		}

		// check if menu items are all deco
		const scrollFirstRate = time.Millisecond * 200
		const scrollRepeatRate = time.Millisecond * 110

		if HandleKeyRepeat(scrollFirstRate, scrollRepeatRate, NoteKeysUp...) {
			if !allDeco {
				scrollUntilNonDeco(false)
			}
		}

		if HandleKeyRepeat(scrollFirstRate, scrollRepeatRate, NoteKeysDown...) {
			if !allDeco {
				scrollUntilNonDeco(true)
			}
		}

		selected := md.Items[md.SelectedIndex]

		if AreKeysPressed(SelectKey) {
			switch selected.Type {
			case MenuItemTrigger:
				selected.Bvalue = true
				selected.NameClickTimer = GlobalTimerNow()
			case MenuItemToggle:
				selected.Bvalue = !selected.Bvalue
				selected.ValueClickTimer = GlobalTimerNow()
			}

			callItemCallaback(selected)
		}

		switch selected.Type {
		case MenuItemList, MenuItemNumber, MenuItemToggle:
			canGoLeft := true
			canGoRight := true

			if selected.Type == MenuItemNumber {
				canGoLeft = selected.CanDecrement()
				canGoRight = selected.CanIncrement()
			} else if selected.Type == MenuItemList {
				canGoLeft = len(selected.List) > 0
				canGoRight = len(selected.List) > 0
			}

			if AreKeysDown(NoteKeysLeft...) && canGoLeft {
				selected.LeftArrowClickTimer = GlobalTimerNow()
			}

			if AreKeysDown(NoteKeysRight...) && canGoRight {
				selected.RightArrowClickTimer = GlobalTimerNow()
			}

			goLeft := false
			goRight := false

			const firstRate = time.Millisecond * 200
			const repeateRate = time.Millisecond * 110

			goLeft = HandleKeyRepeat(firstRate, repeateRate, NoteKeysLeft...) && canGoLeft
			goRight = HandleKeyRepeat(firstRate, repeateRate, NoteKeysRight...) && canGoRight

			switch selected.Type {
			case MenuItemToggle:
				if goLeft || goRight {
					selected.Bvalue = !selected.Bvalue
					callItemCallaback(selected)
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
						callItemCallaback(selected)
					}
				}
			case MenuItemNumber:
				if goLeft {
					selected.NValue -= selected.NValueInterval
					callItemCallaback(selected)
				} else if goRight {
					selected.NValue += selected.NValueInterval
					callItemCallaback(selected)
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
	blend := Clamp(float32(deltaTime.Seconds()*20), 0.00, 1.0)

	seletionY := float32(SCREEN_HEIGHT * 0.5)
	seletionY -= md.GetSelectedItem().SizeRegular * 0.5

	for index, item := range md.Items {
		if index >= md.SelectedIndex {
			break
		}
		seletionY -= item.SizeRegular + md.ListInterval
	}

	if tryingToMove && canNotMove {
		push := (md.GetSelectedItem().SizeRegular*0.5 + md.ListInterval) * 0.8
		if tryingToMoveUp {
			seletionY += push
		} else {
			seletionY -= push
		}
	}

	md.Yoffset = Lerp(md.Yoffset, seletionY, blend)

	md.ScrollAnimT = Lerp(md.ScrollAnimT, 1.0, blend)
}

func (md *MenuDrawer) Draw() {
	if len(md.Items) <= 0 {
		return
	}

	// DEBUG =======================================
	rl.DrawLine(
		0, SCREEN_HEIGHT*0.5,
		SCREEN_WIDTH, SCREEN_HEIGHT*0.5,
		rl.Color{255, 0, 0, 255})
	// DEBUG =======================================

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

	drawText := func(text string, fontSize, scale float32, col Color) float32 {
		textSize := rl.MeasureTextEx(FontBold, text, fontSize, 0)

		pos := rl.Vector2{
			X: xAdvance,
			Y: yCenter - textSize.Y*scale*0.5,
		}

		rl.DrawTextEx(FontBold, text, pos, fontSize*scale, 0, col.ToRlColor())
		return textSize.X
	}

	drawTextCentered := func(text string, fontSize, scale, width float32, col Color) float32 {
		textSize := rl.MeasureTextEx(FontBold, text, fontSize, 0)

		pos := rl.Vector2{
			X: (width-textSize.X)*0.5 + xAdvance,
			Y: yCenter - textSize.Y*scale*0.5,
		}

		rl.DrawTextEx(FontBold, text, pos, fontSize*scale, 0, col.ToRlColor())
		return width
	}

	drawImage := func(
		img rl.Texture2D, srcRect rl.Rectangle, height, scale float32, col Color) float32 {

		wScale := height / srcRect.Height

		dstRect := rl.Rectangle{
			X: xAdvance, Y: yCenter - height*0.5*scale,
			Width: wScale * srcRect.Width * scale, Height: height * scale,
		}

		rl.DrawTexturePro(img, srcRect, dstRect, rl.Vector2{}, 0, col.ToImageRGBA())
		return wScale * srcRect.Width
	}

	drawArrow := func(drawLeft bool, height, scale float32, fill, stroke Color) float32 {
		var innerRect rl.Rectangle
		var outerRect rl.Rectangle

		if drawLeft {
			innerRect = UIarrowRects[UIarrowLeftInner]
			outerRect = UIarrowRects[UIarrowLeftOuter]
		} else {
			innerRect = UIarrowRects[UIarrowRightInner]
			outerRect = UIarrowRects[UIarrowRightOuter]
		}

		rl.BeginBlendMode(rl.BlendAlphaPremultiply)
		advance := drawImage(UIarrowsTex, innerRect, height, scale, fill)
		drawImage(UIarrowsTex, outerRect, height, scale, stroke)
		rl.EndBlendMode()

		return advance
	}

	fadeC := func(col Color, fade float64) Color {
		col.A *= fade
		return col
	}

	for index, item := range md.Items {
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
				if item.Bvalue {
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

		yOffset += item.SizeRegular + md.ListInterval
	}
}

func (md *MenuDrawer) GetSelectedItem() *MenuItem {
	if len(md.Items) <= 0 {
		return nil
	}
	return md.Items[md.SelectedIndex]
}

func (md *MenuDrawer) GetSeletedId() int64 {
	if len(md.Items) <= 0 {
		return 0
	}
	return md.Items[md.SelectedIndex].Id
}

func (md *MenuDrawer) GetItemById(id int64) *MenuItem {
	for _, item := range md.Items {
		if item.Id == id {
			return item
		}
	}

	return nil
}

func (md *MenuDrawer) InsertAt(at int, items ...*MenuItem) {
	at = Clamp(at, 0, len(md.Items))

	var newItems []*MenuItem

	newItems = append(newItems, md.Items[0:at]...)
	newItems = append(newItems, items...)
	newItems = append(newItems, md.Items[at:]...)

	md.Items = newItems
}

func (md *MenuDrawer) ResetAnimation() {
	md.ScrollAnimT = 1
}
