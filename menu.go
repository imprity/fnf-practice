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
)

type MenuItem struct{
	Type MenuItemType

	Id int64

	Name string

	IsTriggered bool
}

var MenuItemMaxId int64
var MenuItemIdMutex sync.Mutex

func MakeMenuItem() MenuItem{
	MenuItemIdMutex.Lock()
	defer MenuItemIdMutex.Unlock()

	item := MenuItem{}

	MenuItemMaxId += 1

	item.Id = MenuItemMaxId

	return item
}

type MenuDrawer struct{
	Items []MenuItem

	SelectedIndex int

	FloatIndex float32

	ScrollAnimT float32

	TriggeredAt time.Duration
	TriggerAnimDuraiton time.Duration

	IsInputDiabled bool
}

func NewMenuDrawer() *MenuDrawer{
	md := new(MenuDrawer)

	md.ScrollAnimT = 1

	md.TriggeredAt = time.Duration(math.MaxInt64)
	md.TriggerAnimDuraiton = time.Millisecond * 150

	return md
}

func (md *MenuDrawer) Update(){
	if len(md.Items) <= 0{
		return
	}

	prevSelected := md.SelectedIndex

	if !md.IsInputDiabled{
		if HandleKeyRepeat(rl.KeyUp, time.Millisecond * 200, time.Millisecond * 110){
			md.SelectedIndex -= 1
		}

		if HandleKeyRepeat(rl.KeyDown, time.Millisecond * 200, time.Millisecond * 110){
			md.SelectedIndex += 1
		}

		if md.SelectedIndex < 0{
			md.SelectedIndex = len(md.Items) -1
		}else if md.SelectedIndex >= len(md.Items){
			md.SelectedIndex = 0
		}
		if prevSelected != md.SelectedIndex{
			md.ScrollAnimT = 0
		}

		if rl.IsKeyPressed(rl.KeyEnter){
			item := md.Items[md.SelectedIndex]

			if item.Type == MenuItemTrigger{
				md.Items[md.SelectedIndex].IsTriggered = true
				md.TriggeredAt = GlobalTimerNow()
				md.IsInputDiabled = true
			}
		}
	}

	// NOTE : I tried to make it frame indipendent from watching this youtube video
	// https://youtu.be/yGhfUcPjXuE?t=1175
	// but I have a strong feeling that this is not frame indipendent
	// but it's just for menu so I don't think it matters too much...
	blend := float32(math.Pow(0.5, float64(rl.GetFrameTime()) * 2000))

	md.FloatIndex = Lerp(md.FloatIndex, float32(md.SelectedIndex), blend)
	md.ScrollAnimT = Lerp(md.ScrollAnimT, 1.0, blend)
}

func (md *MenuDrawer) Draw(){
	if len(md.Items) <= 0{
		return
	}

	const listInterval = 70
	const fontSize = float32(60)

	for index, item := range md.Items{
		x := float32(100)
		yHalf := float32(SCREEN_HEIGHT * 0.5)

		y := yHalf + (float32(index) - md.FloatIndex) * listInterval

		col := Col(0.5, 0.5, 0.5, 1.0)
		size := fontSize

		if index == md.SelectedIndex{
			col = LerpRGB(col, Col(1, 1, 1, 1), float64(md.ScrollAnimT))
			size = Lerp(fontSize, fontSize * 1.05, md.ScrollAnimT)

			triggerT := float32(GlobalTimerNow() - md.TriggeredAt) / float32(md.TriggerAnimDuraiton)

			if triggerT > 0{
				if triggerT > 1{
					triggerT = 1
				}

				tt := -triggerT * (triggerT - 1)

				size *= (1-tt * 0.4)
			}
		}

		y -= size * 0.5

		rl.DrawTextEx(FontBold, item.Name, rl.Vector2{x, y},
			size, 0, col.ToRlColor())
	}
}

func (md *MenuDrawer) GetSeletedId() int64{
	if len(md.Items) <= 0{
		return 0
	}
	return md.Items[md.SelectedIndex].Id
}

func (md *MenuDrawer) Reset(){
	md.IsInputDiabled = false

	md.ScrollAnimT = 1

	md.TriggeredAt = time.Duration(math.MaxInt64)

	for i, item := range md.Items{
		if item.Type == MenuItemTrigger{
			item.IsTriggered = false
		}

		md.Items[i] = item
	}
}
