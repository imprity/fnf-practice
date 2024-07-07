package main

import (
	"fmt"
	"math/rand"
	"time"
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var _ = fmt.Printf

func init() {
	OverrideFirstScreen(func() Screen {
		return NewRtTestScreen()
	})
}

const _appleText = `An apple is a round, edible fruit produced by an apple tree (Malus spp., among them the domestic or orchard apple; Malus domestica).
Apple trees are cultivated worldwide and are the most widely grown species in the genus Malus. The tree originated in Central Asia, where its wild ancestor, Malus sieversii, is still found.
Apples have been grown for thousands of years in Eurasia and were introduced to North America by European colonists.
Apples have religious and mythological significance in many cultures, including Norse, Greek, and European Christian tradition.`

type RtTestScreen struct {
	InputId InputGroupId

	menu *MenuDrawer

	elements []RichTextElement

	fontSize      float32
	lineSpacing   float32
	textWidth     float32
	lineBreakRule RichTextLineBreakRule

	// 0 : left
	// 1 : center
	// 2 : right
	textAlign int

	splitTextRandom bool
}

func (rt *RtTestScreen) GenerateRichTexts() {
	rt.elements = rt.elements[:0]
	factory := NewRichTextFactory(rt.textWidth)
	factory.LineBreakRule = rt.lineBreakRule
	factory.Style.FontSize = rt.fontSize
	factory.LineSpacing = rt.lineSpacing

	if !rt.splitTextRandom {
		elements := factory.Print(_appleText)
		for _, e := range elements {
			rt.elements = append(rt.elements, e)
		}
	} else {
		slice := []byte(_appleText)

		factory.Style.Stroke = Col01(0, 0, 0, 1)
		factory.Style.StrokeWidth = 3
		factory.Style.SdfFont = SdfFontRegular
		factory.Style.UseSdfFont = true

		for len(slice) > 0 {
			n := rand.Intn(30)
			n += 1

			end := 0

			for range n {
				_, s := utf8.DecodeRune(slice[end:])
				end += s

				if end >= len(slice) {
					break
				}
			}

			factory.Style.Fill = Col(uint8(rand.Intn(256)), uint8(rand.Intn(256)), uint8(rand.Intn(256)), 100)
			newElements := factory.Print(string(slice[:end]))

			for _, e := range newElements {
				rt.elements = append(rt.elements, e)
			}

			slice = slice[end:]
		}
	}

}

func NewRtTestScreen() *RtTestScreen {
	rt := new(RtTestScreen)

	rt.InputId = NewInputGroupId()

	rt.textWidth = 300
	rt.fontSize = 40
	rt.lineSpacing = rt.fontSize
	rt.lineBreakRule = LineBreakChar

	rt.menu = NewMenuDrawer()

	rt.GenerateRichTexts()

	return rt
}

func (rt *RtTestScreen) Update(deltaTime time.Duration) {
	repeat := time.Millisecond * 100

	changed := false

	if HandleKeyRepeat(rt.InputId, repeat, repeat, rl.KeyRight) {
		rt.textWidth += 10
		changed = true
	}
	if HandleKeyRepeat(rt.InputId, repeat, repeat, rl.KeyLeft) {
		rt.textWidth -= 10
		changed = true
	}

	if AreKeysPressed(rt.InputId, rl.KeyT) {
		rt.lineBreakRule += 1

		if rt.lineBreakRule >= LineBreakRuleSize {
			rt.lineBreakRule = 0
		}
		changed = true
	}

	if AreKeysPressed(rt.InputId, rl.KeyR) {
		rt.splitTextRandom = !rt.splitTextRandom
		changed = true
	}

	if changed {
		rt.GenerateRichTexts()
	}
}

func (rt *RtTestScreen) Draw() {
	rl.ClearBackground(ToRlColor(Col01(1, 1, 1, 1)))

	// draw vertical grid
	rl.DrawLine(
		i32(rt.textWidth), 0, i32(rt.textWidth), SCREEN_HEIGHT,
		ToRlColor(Col01(1, 0, 0, 1)),
	)

	// draw horizontal grid
	{
		offsetY := rt.lineSpacing
		for offsetY < SCREEN_HEIGHT {
			rl.DrawLine(
				0, i32(offsetY), SCREEN_WIDTH, i32(offsetY),
				ToRlColor(Col01(1, 0, 0, 1)),
			)

			offsetY += rt.lineSpacing
		}
	}

	var elements []RichTextElement

	for i := 0; i < len(rt.elements); i++ {
		elements = append(elements, rt.elements[i])

		doPrint := false

		if i+1 < len(rt.elements) && rt.elements[i+1].StartsAfterLineBreak {
			doPrint = true
		}
		if i == len(rt.elements)-1 {
			doPrint = true
		}

		if doPrint {
			if len(elements) <= 0 {
				continue
			}

			bound := elements[0].Bound

			for j := 1; j < len(elements); j++ {
				bound = RectUnion(elements[j].Bound, bound)
			}

			offsetX := float32(0)

			switch rt.textAlign {
			case 0:
				offsetX = 0
			case 1:
				offsetX = SCREEN_WIDTH*0.5 - bound.Width*0.5
			case 2:
				offsetX = SCREEN_WIDTH - bound.Width
			}

			for _, e := range elements {
				rng := rand.New(rand.NewSource(int64(i)))

				bg := Col(uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 100)

				rl.DrawRectangleRec(e.Bound, ToRlColor(bg))

				fillColor := e.Style.Fill

				if e.StartsAfterLineBreak {
					fillColor = Col01(1, 0, 0, 1)
				}

				pos := RectPos(e.Bound)
				pos.X += offsetX

				if e.Style.UseSdfFont {
					if e.Style.StrokeWidth <= 0 {
						DrawTextSdf(e.Style.SdfFont, e.Text,
							pos, e.Style.FontSize, 0, ToRlColor(fillColor))
					} else {
						DrawTextSdfOutlined(e.Style.SdfFont, e.Text,
							pos, e.Style.FontSize, 0,
							ToRlColor(fillColor), ToRlColor(e.Style.Stroke), e.Style.StrokeWidth)
					}
				} else {
					rl.DrawTextEx(e.Style.Font, e.Text,
						pos, e.Style.FontSize, 0, ToRlColor(fillColor))
				}
			}

			elements = elements[:0]
		}
	}
}

func (rt *RtTestScreen) BeforeScreenTransition() {
}

func (rt *RtTestScreen) Free() {
}
