package main

import (
	"sync"
	"time"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)
type MenuItemType int

const (
	MenuItemTrigger MenuItemType = iota
	MenuItemDeco
)

type MenuItem struct{
	Type MenuItemType

	SizeRegular float32
	SizeSelected float32

	ColSelected Color
	ColRegular Color

	Id int64

	Name string

	Bvalue bool

	ValueChangedAt time.Duration
}

var MenuItemMaxId int64
var MenuItemIdMutex sync.Mutex

const (
	MenuItemSizeRegularDefault = 70
	MenuItemSizeSelectedDefault = 90
)

func MakeMenuItem() MenuItem{
	MenuItemIdMutex.Lock()
	defer MenuItemIdMutex.Unlock()

	item := MenuItem{}

	MenuItemMaxId += 1

	item.Id = MenuItemMaxId

	item.SizeRegular = MenuItemSizeRegularDefault
	item.SizeSelected = MenuItemSizeSelectedDefault

	item.ColSelected = Col(1,1,1,1)
	item.ColRegular = Col(1,1,1,0.5)

	item.ValueChangedAt = time.Duration(math.MaxInt64)

	return item
}

type MenuDrawer struct{
	Items []MenuItem

	SelectedIndex int

	ListInterval float32

	Yoffset float32

	ScrollAnimT float32

	TriggerAnimDuraiton time.Duration

	IsInputDiabled bool
}

func NewMenuDrawer() *MenuDrawer{
	md := new(MenuDrawer)

	md.ScrollAnimT = 1

	md.TriggerAnimDuraiton = time.Millisecond * 150

	md.ListInterval = 30

	return md
}

func (md *MenuDrawer) Update(){
	if len(md.Items) <= 0{
		return
	}

	for index, item := range md.Items{
		if item.Type == MenuItemTrigger{
			md.Items[index].Bvalue = false
		}
	}

	prevSelected := md.SelectedIndex

	allDeco := true
	nonDecoCount := 0

	for _, item :=range md.Items{
		if item.Type != MenuItemDeco{
			nonDecoCount += 1
			allDeco = false
		}
	}

	scrollUntilNonDeco := func(forward bool){
		for {
			if forward{
				md.SelectedIndex += 1
			}else{
				md.SelectedIndex -= 1
			}

			if md.SelectedIndex >= len(md.Items){
				md.SelectedIndex =  0
			}else if md.SelectedIndex < 0{
				md.SelectedIndex = len(md.Items) - 1
			}

			if md.Items[md.SelectedIndex].Type != MenuItemDeco{
				break
			}
		}
	}

	if !allDeco{
		if md.Items[md.SelectedIndex].Type == MenuItemDeco{
			scrollUntilNonDeco(true)
		}
	}

	tryingToMove := false
	tryingToMoveUp := false
	canNotMove := false

	if nonDecoCount <= 1{
		canNotMove = true
	}


	if !md.IsInputDiabled{
		if rl.IsKeyDown(rl.KeyUp){
			tryingToMove = true
			tryingToMoveUp = true
		}

		if rl.IsKeyDown(rl.KeyDown){
			tryingToMove = true
			tryingToMoveUp = false
		}

		// check if menu items are all deco
		firstRate := time.Millisecond * 200
		repeateRate := time.Millisecond * 110

		if HandleKeyRepeat(rl.KeyUp, firstRate, repeateRate){
			if !allDeco{
				scrollUntilNonDeco(false)
			}

		}

		if HandleKeyRepeat(rl.KeyDown, firstRate, repeateRate){
			if !allDeco{
				scrollUntilNonDeco(true)
			}
		}


		if rl.IsKeyPressed(rl.KeyEnter){
			item := md.Items[md.SelectedIndex]

			if item.Type == MenuItemTrigger{
				md.Items[md.SelectedIndex].Bvalue = true
				md.Items[md.SelectedIndex].ValueChangedAt = GlobalTimerNow()
			}
		}
	}

	if md.SelectedIndex != prevSelected{
		md.ScrollAnimT = 0
	}

	// but I have a strong feeling that this is not frame indipendent
	// but it's just for menu so I don't think it matters too much...
	blend := Clamp(float32(rl.GetFrameTime() * 20), 0.01, 1.0)

	seletionY := float32(SCREEN_HEIGHT * 0.5)
	seletionY -= md.GetSelectedItem().SizeRegular * 0.5

	for index, item := range md.Items{
		if index >= md.SelectedIndex{
			break
		}
		seletionY -= item.SizeRegular + md.ListInterval
	}

	if tryingToMove && canNotMove{
		push := (md.GetSelectedItem().SizeRegular * 0.5 + md.ListInterval) * 0.8
		if tryingToMoveUp {
			seletionY += push
		}else{
			seletionY -= push
		}
	}

	md.Yoffset = Lerp(md.Yoffset, seletionY, blend)

	md.ScrollAnimT = Lerp(md.ScrollAnimT, 1.0, blend)
}

func (md *MenuDrawer) Draw(){
	if len(md.Items) <= 0{
		return
	}

	yOffset := md.Yoffset
	xOffset := float32(100)

	for index, item := range md.Items{
		y := yOffset
		x := xOffset

		scale := float32(1.0)

		col := item.ColRegular
		size := item.SizeRegular

		if index == md.SelectedIndex{
			col = LerpRGBA(item.ColRegular, item.ColSelected, float64(md.ScrollAnimT))
			size = Lerp(item.SizeRegular, item.SizeSelected, md.ScrollAnimT)
			x += Lerp(0, 30, md.ScrollAnimT)
		}

		if item.Type == MenuItemTrigger{
			triggerT := float32(GlobalTimerNow() - item.ValueChangedAt) / float32(md.TriggerAnimDuraiton)

			if triggerT > 0{
				if triggerT > 1{ triggerT = 1 }
				tt := -triggerT * (triggerT - 1)

				scale *= (1-tt * 0.4)
			}
		}

		y += item.SizeRegular * 0.5 - size * 0.5 * scale

		rl.DrawTextEx(FontBold, item.Name, rl.Vector2{x, y},
			size * scale, 0, col.ToRlColor())

		yOffset += item.SizeRegular + md.ListInterval
	}

	// DEBUG =======================================
	rl.DrawLine(
		0, SCREEN_HEIGHT * 0.5,
		SCREEN_WIDTH, SCREEN_HEIGHT * 0.5,
		rl.Color{255, 0, 0, 255})
	// DEBUG =======================================
}

func (md *MenuDrawer) GetSelectedItem() MenuItem{
	if len(md.Items) <= 0{
		return MenuItem{}
	}
	return md.Items[md.SelectedIndex]
}

func (md *MenuDrawer) GetSeletedId() int64{
	if len(md.Items) <= 0{
		return 0
	}
	return md.Items[md.SelectedIndex].Id
}

func (md *MenuDrawer) GetItemById(id int64) (MenuItem, bool) {
	for _, item := range md.Items{
		if item.Id == id{
			return item, true
		}
	}

	return MenuItem{}, false
}

func (md *MenuDrawer) InsertAt(at int, items ...MenuItem){
	at = Clamp(at, 0, len(md.Items))

	var newItems []MenuItem

	newItems = append(newItems, md.Items[0:at]...)
	newItems = append(newItems, items...)
	newItems = append(newItems, md.Items[at:]...)

	md.Items = newItems
}

func (md *MenuDrawer) ResetAnimation(){
	md.ScrollAnimT = 1
}

