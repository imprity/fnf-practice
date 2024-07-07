package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"strconv"
	"unicode/utf8"
)

var _ = fmt.Println
var _ = strconv.Quote

type RichTextStyle struct {
	FontSize float32

	Font       rl.Font
	SdfFont    SdfFont
	UseSdfFont bool

	Fill, Stroke FnfColor
	StrokeWidth  float32
}

type RichTextElement struct {
	Text string

	Bound rl.Rectangle

	Style RichTextStyle

	StartsAfterLineBreak bool
}

type RichTextLineBreakRule int

const (
	LineBreakChar RichTextLineBreakRule = iota
	LineBreakWord
	LineBreakNever
	LineBreakRuleSize
)

var RichTextLineBreakRuleStrs = [LineBreakRuleSize]string{
	"character",
	"word",
	"never",
}

type RichTextFactory struct {
	Style       RichTextStyle
	LineSpacing float32

	LineBreakRule RichTextLineBreakRule

	width float32

	cursor rl.Vector2

	elements []RichTextElement

	brokeLine bool
}

func NewRichTextFactory(width float32) *RichTextFactory {
	return &RichTextFactory{
		Style: RichTextStyle{
			FontSize: 30,
			Font:     FontRegular,
			Fill:     Col(0, 0, 0, 255),
		},
		LineSpacing: 30,

		width: width,
	}
}

func (rt *RichTextFactory) Width() float32 {
	return rt.width
}

func (rt *RichTextFactory) Print(text string) []RichTextElement {
	if len(text) <= 0 {
		return []RichTextElement{}
	}
	var newElements []RichTextElement
	// check if cursor is out side width

	breakLine := func() {
		rt.brokeLine = true
		rt.cursor.X = 0
		rt.cursor.Y += rt.LineSpacing
	}

	if rt.LineBreakRule != LineBreakNever && rt.cursor.X > rt.width {
		breakLine()
	}

	font := rt.Style.Font
	if rt.Style.UseSdfFont {
		font = rt.Style.SdfFont.Font
	}

	scaleFactor := rt.Style.FontSize / f32(font.BaseSize)

	getTextSize := func(start, end int) float32 {
		textSize := float32(0)
		for _, char := range text[start:end] {
			glyph := rl.GetGlyphInfo(font, char)

			charSize := float32(glyph.AdvanceX)
			if charSize == 0 {
				rec := rl.GetGlyphAtlasRec(font, char)
				charSize = rec.Width
			}
			charSize *= scaleFactor

			textSize += charSize
		}
		return textSize
	}

	textStart := 0
	textEnd := 0
	textSize := float32(0)

	saveToken := func(tkStart, tkEnd int, tkSize float32) {
		textSize += tkSize
		textEnd = tkEnd
	}

	printSavedToken := func() bool {
		if textEnd > textStart {
			newElements = append(newElements, RichTextElement{
				Text: text[textStart:textEnd],
				Bound: rl.Rectangle{
					X: rt.cursor.X, Y: rt.cursor.Y,
					Width: textSize, Height: rt.Style.FontSize,
				},
				Style:                rt.Style,
				StartsAfterLineBreak: rt.brokeLine,
			})

			rt.brokeLine = false
			rt.cursor.X += textSize
			textStart = textEnd
			textSize = 0

			return true
		}
		return false
	}

	iter := newIteratorForRT([]byte(text), rt.LineBreakRule)

	for iter.HasNext() {
		tkStart, tkEnd := iter.Next()

		if text[tkStart:tkEnd] == "\n" {
			printSavedToken()
			breakLine()
			textStart = tkEnd
			textEnd = tkEnd
		} else {
			tkSize := getTextSize(tkStart, tkEnd)

			if rt.LineBreakRule == LineBreakNever {
				saveToken(tkStart, tkEnd, tkSize)
			} else {
				if rt.cursor.X+textSize+tkSize > rt.width {
					printSavedToken()
					if rt.cursor.X > 0 {
						breakLine()
					}

					saveToken(tkStart, tkEnd, tkSize)

					if rt.cursor.X+textSize > rt.width {
						printSavedToken()
						breakLine()
					}
				} else {
					saveToken(tkStart, tkEnd, tkSize)
					if rt.cursor.X+textSize > rt.width {
						printSavedToken()
						breakLine()
					}
				}
			}
		}
	}

	printSavedToken()

	for _, e := range newElements {
		rt.elements = append(rt.elements, e)
	}

	return newElements
}

func (rt *RichTextFactory) Elements() []RichTextElement {
	return rt.elements
}

func (rt *RichTextFactory) Cursor() rl.Vector2 {
	return rt.cursor
}

type iteratorForRT struct {
	text          []byte
	lineBreakRule RichTextLineBreakRule
	pos           int
}

func newIteratorForRT(
	textAsBytes []byte,
	lineBreakRule RichTextLineBreakRule,
) *iteratorForRT {
	return &iteratorForRT{
		text:          textAsBytes,
		lineBreakRule: lineBreakRule,
	}
}

func (it *iteratorForRT) HasNext() bool {
	return it.pos < len(it.text)
}

func (it *iteratorForRT) Next() (int, int) {
	if it.lineBreakRule == LineBreakWord {
		prevPos := it.pos

		for it.pos < len(it.text) {
			r, s := utf8.DecodeRune(it.text[it.pos:])

			isSpecial := r == '\n' || r == ' '

			if isSpecial {
				if prevPos == it.pos { // this is the first character we encountered
					it.pos += s
				}

				return prevPos, it.pos
			}

			it.pos += s
		}

		return prevPos, it.pos
	} else {
		prevPos := it.pos
		_, s := utf8.DecodeRune(it.text[it.pos:])
		it.pos += s

		return prevPos, it.pos
	}
}
