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

type RtTestScreen struct {
	elements []RichTextElement
	factory *RichTextFactory
}


func NewRtTestScreen() *RtTestScreen {
	rt := new(RtTestScreen)

	rt.factory = NewRichTextFactory(100)

	rt.factory.Print("Hello World! ")
	rt.factory.Style.Fill = FnfColor{0,255,0,255}
	rt.factory.Print("This\nIs\nPretty Fucking\n\n\nCool")
	rt.factory.Style.FontSize *=2
	rt.factory.LineSpacing *= 2
	rt.factory.Style.Fill = FnfColor{0,0,255,255}
	rt.factory.Print("yoyoyoy0")


	rt.elements = rt.factory.elements

	return rt
}

func (rt *RtTestScreen) Update(deltaTime time.Duration) {
}

func (rt *RtTestScreen) Draw() {
	rl.ClearBackground(ToRlColor(Col01(1,1,1,1)))

	rl.DrawLine(
		i32(rt.factory.Width()), 0, i32(rt.factory.Width()), SCREEN_HEIGHT,
		ToRlColor(Col01(1,0,0,1)),
	)

	{
		offsetY := rt.factory.LineSpacing
		for offsetY < SCREEN_HEIGHT{
			rl.DrawLine(
				0, i32(offsetY), SCREEN_WIDTH, i32(offsetY),
				ToRlColor(Col01(1,0,0,1)),
			)

			offsetY += rt.factory.LineSpacing
		}
	}


	for _, e := range rt.elements{
		if e.Style.UseSdfFont{
			if e.Style.StrokeWidth <= 0 {
				DrawTextSdf(e.Style.SdfFont, e.Text,
					e.Pos, e.Style.FontSize, 0, ToRlColor(e.Style.Fill))
			}else{
				DrawTextSdfOutlined(e.Style.SdfFont, e.Text,
					e.Pos, e.Style.FontSize, 0,
					ToRlColor(e.Style.Fill), ToRlColor(e.Style.Stroke), e.Style.StrokeWidth)
			}
		}else{
			rl.DrawTextEx(e.Style.Font, e.Text,
				e.Pos, e.Style.FontSize, 0, ToRlColor(e.Style.Fill))
		}
	}
}

func (rt *RtTestScreen) BeforeScreenTransition() {
}

func (rt *RtTestScreen) Free() {
}
