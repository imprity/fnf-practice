package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"fmt"
)

var _= fmt.Println

type RichTextStyle struct {
	FontSize float32

	Font rl.Font
	SdfFont SdfFont
	UseSdfFont bool

	Fill, Stroke FnfColor
	StrokeWidth float32
}

type RichTextElement struct {
	Text string
	Pos rl.Vector2
	Style RichTextStyle
}

type RichTextFactory struct{
	Style RichTextStyle
	LineSpacing float32

	width float32

	cursor rl.Vector2

	elements []RichTextElement
}

func NewRichTextFactory (width float32) *RichTextFactory{
	return &RichTextFactory{
		Style : RichTextStyle {
			FontSize : 30,
			Font : FontRegular,
			Fill : Col(0,0,0,255),
		},
		LineSpacing : 30,

		width : width,
	}
}

func (rt *RichTextFactory) Width() float32{
	return rt.width
}

func (rt *RichTextFactory) Print(text string){
	if len(text) <= 0 {
		return
	}
	// check if cursor is out side width

	if rt.cursor.X > rt.width {
		rt.cursor.X = 0
		rt.cursor.Y += rt.LineSpacing
	}

	font := rt.Style.Font
	if rt.Style.UseSdfFont {
		font = rt.Style.SdfFont.Font
	}

	scaleFactor := rt.Style.FontSize / f32(font.BaseSize)

	start := 0
	savedCursor := rt.cursor

	commit := func(pos int){
		rt.elements = append(rt.elements, RichTextElement{
			Text : text[start:pos],
			Pos : savedCursor,
			Style : rt.Style,
		})
		savedCursor = rt.cursor
		start = pos
	}

	breakLine := func(){
		rt.cursor.X = 0
		rt.cursor.Y += rt.LineSpacing
		savedCursor = rt.cursor
	}

	for pos, char := range text{
		if char == '\n'{
			commit(pos)
			breakLine()
			start = pos + 1
		}else{
			glyph := rl.GetGlyphInfo(font, char)

			charAdvance := float32(glyph.AdvanceX)
			if charAdvance == 0{
				rec := rl.GetGlyphAtlasRec(font, char)
				charAdvance = rec.Width
			}
			charAdvance *= scaleFactor

			if rt.cursor.X + charAdvance > rt.width{
				commit(pos)
				breakLine()
				start = pos
			}			

			rt.cursor.X += charAdvance
		}
	}

	if start < len(text){
		commit(len(text))
	}
}

