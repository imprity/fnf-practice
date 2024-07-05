package main

import (
	"fmt"
	"time"

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

	elements []RichTextElement

	fontSize float32
	lineSpacing float32
	textWidth float32
	lineBreakRule int
}

func (rt *RtTestScreen) GenerateRichTexts() {
	rt.elements = rt.elements[:0]
	factory := NewRichTextFactory(rt.textWidth)
	factory.LineBreakRule = rt.lineBreakRule
	factory.Style.FontSize = rt.fontSize
	factory.LineSpacing = rt.lineSpacing

	elements := factory.Print(_appleText)

	for _, e := range elements{
		rt.elements = append(rt.elements, e)
	}
}

func NewRtTestScreen() *RtTestScreen {
	rt := new(RtTestScreen)

	rt.InputId = NewInputGroupId()

	rt.textWidth = 300
	rt.fontSize = 40
	rt.lineSpacing = rt.fontSize
	rt.lineBreakRule = LineBreakChar

	rt.GenerateRichTexts()

	return rt
}

func (rt *RtTestScreen) Update(deltaTime time.Duration) {
	repeat := time.Millisecond * 100

	changed := false

	if HandleKeyRepeat(rt.InputId, repeat, repeat, rl.KeyRight){
		rt.textWidth += 10
		changed = true
	}
	if HandleKeyRepeat(rt.InputId, repeat, repeat, rl.KeyLeft){
		rt.textWidth -= 10
		changed = true
	}

	if changed{
		rt.GenerateRichTexts()
	}
}

func (rt *RtTestScreen) Draw() {
	rl.ClearBackground(ToRlColor(Col01(1, 1, 1, 1)))

	rl.DrawLine(
		i32(rt.textWidth), 0, i32(rt.textWidth), SCREEN_HEIGHT,
		ToRlColor(Col01(1, 0, 0, 1)),
	)

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

	for _, e := range rt.elements {
		if e.Style.UseSdfFont {
			if e.Style.StrokeWidth <= 0 {
				DrawTextSdf(e.Style.SdfFont, e.Text,
					e.Pos, e.Style.FontSize, 0, ToRlColor(e.Style.Fill))
			} else {
				DrawTextSdfOutlined(e.Style.SdfFont, e.Text,
					e.Pos, e.Style.FontSize, 0,
					ToRlColor(e.Style.Fill), ToRlColor(e.Style.Stroke), e.Style.StrokeWidth)
			}
		} else {
			rl.DrawTextEx(e.Style.Font, e.Text,
				e.Pos, e.Style.FontSize, 0, ToRlColor(e.Style.Fill))
		}
	}
}

func (rt *RtTestScreen) BeforeScreenTransition() {
}

func (rt *RtTestScreen) Free() {
}
