package main

import (
	//"math"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type MenuItemType int

const (
	MenuItemTrigger MenuItemType = iota
	MenuItemToggle
	MenuItemList
	MenuItemDeco
)

type MenuItem struct {
	Type MenuItemType

	SizeRegular  float32
	SizeSelected float32

	ColSelected Color
	ColRegular  Color

	Id int64

	Name string

	Bvalue bool

	ListSelected int
	List         []string

	ValueChangedAt time.Duration
}

var MenuItemMaxId int64
var MenuItemIdMutex sync.Mutex

const (
	MenuItemSizeRegularDefault  = 70
	MenuItemSizeSelectedDefault = 90
)

func MakeMenuItem() MenuItem {
	MenuItemIdMutex.Lock()
	defer MenuItemIdMutex.Unlock()

	item := MenuItem{}

	MenuItemMaxId += 1

	item.Id = MenuItemMaxId

	item.SizeRegular = MenuItemSizeRegularDefault
	item.SizeSelected = MenuItemSizeSelectedDefault

	item.ColSelected = Col(1, 1, 1, 1)
	item.ColRegular = Col(1, 1, 1, 0.5)

	item.ValueChangedAt = Years150

	return item
}

type MenuDrawer struct {
	Items []MenuItem

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

func (md *MenuDrawer) Update() {
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
		if AreKeysDown(NoteKeysUp...) {
			tryingToMove = true
			tryingToMoveUp = true
		}

		if AreKeysDown(NoteKeysDown...) {
			tryingToMove = true
			tryingToMoveUp = false
		}

		// check if menu items are all deco
		firstRate := time.Millisecond * 200
		repeateRate := time.Millisecond * 110

		if HandleKeyRepeat(firstRate, repeateRate, NoteKeysUp...) {
			if !allDeco {
				scrollUntilNonDeco(false)
			}

		}

		if HandleKeyRepeat(firstRate, repeateRate, NoteKeysDown...) {
			if !allDeco {
				scrollUntilNonDeco(true)
			}
		}

		if AreKeysPressed(SelectKey) {
			item := md.Items[md.SelectedIndex]

			switch item.Type {
			case MenuItemTrigger:
				md.Items[md.SelectedIndex].Bvalue = true
				md.Items[md.SelectedIndex].ValueChangedAt = GlobalTimerNow()
			case MenuItemToggle:
				md.Items[md.SelectedIndex].Bvalue = !md.Items[md.SelectedIndex].Bvalue
				md.Items[md.SelectedIndex].ValueChangedAt = GlobalTimerNow()
			}
		}

		item := md.Items[md.SelectedIndex]

		if item.Type == MenuItemList && len(item.List) > 0 {
			selectedNew := item.ListSelected

			if AreKeysPressed(NoteKeysLeft...) {
				md.Items[md.SelectedIndex].ValueChangedAt = GlobalTimerNow()
				selectedNew -= 1
			}

			if AreKeysPressed(NotekeysRight...) {
				md.Items[md.SelectedIndex].ValueChangedAt = GlobalTimerNow()
				selectedNew += 1
			}

			selectedNew = Clamp(selectedNew, 0, len(item.List)-1)
			md.Items[md.SelectedIndex].ListSelected = selectedNew
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
	blend := Clamp(float32(rl.GetFrameTime()*20), 0.01, 1.0)

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

	yOffset := md.Yoffset
	xOffset := float32(100)

	for index, item := range md.Items {
		y := yOffset
		x := xOffset

		scale := float32(1.0)

		col := item.ColRegular
		size := item.SizeRegular

		if index == md.SelectedIndex {
			col = LerpRGBA(item.ColRegular, item.ColSelected, float64(md.ScrollAnimT))
			size = Lerp(item.SizeRegular, item.SizeSelected, md.ScrollAnimT)
			x += Lerp(0, 30, md.ScrollAnimT)
		}

		if item.Type == MenuItemTrigger || item.Type == MenuItemToggle {
			clickT := float32(GlobalTimerNow()-item.ValueChangedAt) / float32(md.TriggerAnimDuraiton)

			if clickT > 0 {
				if clickT > 1 {
					clickT = 1
				}
				tt := -clickT * (clickT - 1)

				scale *= (1 - tt*0.4)
			}
		}

		y += item.SizeRegular*0.5 - size*0.5*scale

		textToDraw := item.Name

		if item.Type == MenuItemToggle {
			if item.Bvalue {
				textToDraw += " : Yes"
			} else {
				textToDraw += " : No"
			}
		} else if item.Type == MenuItemList && len(item.List) > 0 {
			textToDraw += " : "
			textToDraw += item.List[item.ListSelected]
		}

		rl.DrawTextEx(FontBold, textToDraw, rl.Vector2{x, y},
			size*scale, 0, col.ToRlColor())

		yOffset += item.SizeRegular + md.ListInterval
	}
}

func (md *MenuDrawer) GetSelectedItem() MenuItem {
	if len(md.Items) <= 0 {
		return MenuItem{}
	}
	return md.Items[md.SelectedIndex]
}

func (md *MenuDrawer) GetSeletedId() int64 {
	if len(md.Items) <= 0 {
		return 0
	}
	return md.Items[md.SelectedIndex].Id
}

func (md *MenuDrawer) GetItemById(id int64) (MenuItem, bool) {
	for _, item := range md.Items {
		if item.Id == id {
			return item, true
		}
	}

	return MenuItem{}, false
}

func (md *MenuDrawer) SetItem(toSet MenuItem) bool {
	for i, item := range md.Items {
		if item.Id == toSet.Id {
			md.Items[i] = toSet
		}
	}

	return false
}

func (md *MenuDrawer) InsertAt(at int, items ...MenuItem) {
	at = Clamp(at, 0, len(md.Items))

	var newItems []MenuItem

	newItems = append(newItems, md.Items[0:at]...)
	newItems = append(newItems, items...)
	newItems = append(newItems, md.Items[at:]...)

	md.Items = newItems
}

func (md *MenuDrawer) ResetAnimation() {
	md.ScrollAnimT = 1
}
