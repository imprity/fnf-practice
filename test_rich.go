package main

import (
	"time"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	OverrideFirstScreen(func() Screen {
		return NewRtTestScreen()
	})
}

type RtTestScreen struct {
	factory *RichTextFactory
	textAlign RichTextAlign
	elements []RichTextElement
}

func (rt *RtTestScreen) GenerateRichTexts() {
}

func NewRtTestScreen() *RtTestScreen {
	rt := new(RtTestScreen)
	rt.factory = NewRichTextFactory(300)

	style := rt.factory.Style()
	rt.factory.LineBreakRule = LineBreakWord

	hugeStyle := style
	hugeStyle.FontSize *= 3

	rt.factory.LineSpacing *= 3

	rt.factory.SetStyle(hugeStyle)
	rt.factory.Print("The ")

	rt.factory.SetStyle(style)
	rt.factory.Print(
`Combining Diacritical Marks is a Unicode block containing the most common combining characters.
It also contains the character "Combining Grapheme Joiner", which prevents canonical reordering of combining characters, and despite the name, actually separates characters that would otherwise be considered a single grapheme in a given context.`)

	for _, e := range rt.factory.Elements(){
		rt.elements = append(rt.elements, e)
	}
	return rt
}

func (rt *RtTestScreen) Update(deltaTime time.Duration) {
	if rl.IsKeyPressed(rl.KeyA){
		rt.textAlign += 1
		if rt.textAlign >= TextAlignSize{
			rt.textAlign = 0
		}

		AlignElements(rt.elements, rt.textAlign)
	}
}

func (rt *RtTestScreen) Draw() {
	rl.ClearBackground(rl.Color{255,255,255,255})

	for _, e := range rt.elements{
		rl.DrawTextEx(e.Style.Font, e.Text, RectPos(e.Bound), e.Style.FontSize, 0, ToRlColor(e.Style.Fill))
	}
}

func (rt *RtTestScreen) BeforeScreenTransition() {
}

func (rt *RtTestScreen) Free() {
}
