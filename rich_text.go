package main

import (
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

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

	Style *RichTextStyle

	Metadata int64

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

type RichTextAlign int

const (
	TextAlignLeft RichTextAlign = iota
	TextAlignCenter
	TextAlignRight
	TextAlignSize
)

var RichTextAlignStrs = [TextAlignSize]string{
	"left",
	"center",
	"right",
}

type RichTextFactory struct {
	Metadata int64

	LineSpacing float32

	LineBreakRule RichTextLineBreakRule

	style *RichTextStyle

	width float32

	cursor rl.Vector2

	elements []RichTextElement

	brokeLine bool

	alignedLineToBottom bool
}

func NewRichTextFactory(width float32) *RichTextFactory {
	return &RichTextFactory{
		style: &RichTextStyle{
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

func (rt *RichTextFactory) Style() RichTextStyle {
	return *rt.style
}

func (rt *RichTextFactory) SetStyle(style RichTextStyle) {
	rt.style = &style
}

func (rt *RichTextFactory) Print(text string) {
	if len(text) <= 0 {
		return
	}

	rt.alignedLineToBottom = false
	// check if cursor is out side width

	breakLine := func() {
		rt.brokeLine = true
		rt.cursor.X = 0
		rt.cursor.Y += rt.LineSpacing
	}

	if rt.LineBreakRule != LineBreakNever && rt.cursor.X > rt.width {
		breakLine()
	}

	font := rt.style.Font
	if rt.style.UseSdfFont {
		font = rt.style.SdfFont.Font
	}

	scaleFactor := rt.style.FontSize / f32(font.BaseSize)

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
			rt.elements = append(rt.elements, RichTextElement{
				Text: text[textStart:textEnd],
				Bound: rl.Rectangle{
					X: rt.cursor.X, Y: rt.cursor.Y,
					Width: textSize, Height: rt.style.FontSize,
				},
				Style:                rt.style,
				StartsAfterLineBreak: rt.brokeLine,
				Metadata:             rt.Metadata,
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
}

func (rt *RichTextFactory) Elements() []RichTextElement {
	if !rt.alignedLineToBottom {
		iter := NewRTElineIterator(rt.elements)

		for iter.HasNext() {
			b, e := iter.Next()

			if e-b > 1 {
				maxBound := rt.elements[b].Bound

				for i := b + 1; i < e; i++ {
					maxBound = RectUnion(maxBound, rt.elements[i].Bound)
				}

				for i := b; i < e; i++ {
					rt.elements[i].Bound.Y = maxBound.Y + maxBound.Height - rt.elements[i].Bound.Height
				}
			}
		}

		rt.alignedLineToBottom = true
	}

	return rt.elements
}

func (rt *RichTextFactory) Cursor() rl.Vector2 {
	return rt.cursor
}

func ElementsBound(elements []RichTextElement) rl.Rectangle{
	if len(elements) <= 0 {
		return rl.Rectangle{}
	}

	bound := elements[0].Bound

	for i:=1; i<len(elements); i++{
		bound = RectUnion(bound, elements[i].Bound)
	}

	return bound
}

func AlignElements(elements []RichTextElement, align RichTextAlign){
	iter := NewRTElineIterator(elements)

	totalBound := ElementsBound(elements)

	for iter.HasNext(){
		b, e := iter.Next()

		lineBound := ElementsBound(elements[b:e])

		var offsetX float32

		switch align {
		case TextAlignLeft :
			offsetX = -lineBound.X
		case TextAlignRight :
			offsetX = totalBound.Width - lineBound.Width - lineBound.X
		case TextAlignCenter :
			offsetX = totalBound.Width * 0.5 - lineBound.Width * 0.5 - lineBound.X
		}

		for i:=b; i<e; i++{
			elements[i].Bound.X += offsetX
		}
	}
}

type RTElineIterator struct {
	elements []RichTextElement
	pos      int
}

func NewRTElineIterator(elements []RichTextElement) *RTElineIterator {
	return &RTElineIterator{elements: elements}
}

func (it *RTElineIterator) HasNext() bool {
	return it.pos < len(it.elements)
}

func (it *RTElineIterator) Next() (int, int) {
	prevPos := it.pos
	it.pos += 1

	for it.pos < len(it.elements) {
		if it.elements[it.pos].StartsAfterLineBreak {
			return prevPos, it.pos
		}

		it.pos += 1
	}

	return prevPos, it.pos
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

