package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

/*
func init() {
	OverrideFirstScreen(func() Screen {
		return NewRtTestScreen()
	})
}
*/

type RtTestScreen struct {
	factory   *RichTextFactory
	textAlign RichTextAlign
	elements  []RichTextElement
}

func (rt *RtTestScreen) GenerateRichTexts() {
}

func NewRtTestScreen() *RtTestScreen {
	rt := new(RtTestScreen)
	rt.factory = NewRichTextFactory(300)

	style := rt.factory.Style()
	rt.factory.LineBreakRule = LineBreakWord

	rt.factory.SetStyle(style)
	rt.factory.Print(
		`

meme

lol`)

	for _, e := range rt.factory.Elements(TextAlignLeft, 0, style.FontSize) {
		rt.elements = append(rt.elements, e)
	}
	return rt
}

func (rt *RtTestScreen) Update(deltaTime time.Duration) {
	if rl.IsKeyPressed(rl.KeyA) {
		rt.textAlign += 1
		if rt.textAlign >= TextAlignSize {
			rt.textAlign = 0
		}

		AlignElements(rt.elements, rt.textAlign)
	}
}

func (rt *RtTestScreen) Draw() {
	rl.ClearBackground(rl.Color{255, 255, 255, 255})

	{
		offsetY := int32(0)

		for offsetY < SCREEN_HEIGHT {
			rl.DrawLine(0, offsetY, SCREEN_WIDTH, offsetY, rl.Red)
			offsetY += 30
		}
	}

	for _, e := range rt.elements {
		rl.DrawTextEx(e.Style.Font, e.Text, RectPos(e.Bound), e.Style.FontSize, 0, ToRlColor(e.Style.Fill))
	}
}

func (rt *RtTestScreen) BeforeScreenTransition() {
}

func (rt *RtTestScreen) Free() {
}
