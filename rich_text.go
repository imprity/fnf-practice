package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"strconv"
	"unicode/utf8"
)

var _ = fmt.Println
var _=strconv.Quote

type RichTextStyle struct {
	FontSize float32

	Font       rl.Font
	SdfFont    SdfFont
	UseSdfFont bool

	Fill, Stroke FnfColor
	StrokeWidth  float32
}

type RichTextElement struct {
	Text  string
	Pos   rl.Vector2
	Style RichTextStyle
}

const (
	LineBreakChar = iota
	LineBreakWord
)

type RichTextFactory struct {
	Style       RichTextStyle
	LineSpacing float32

	LineBreakRule int

	width float32

	cursor rl.Vector2

	elements []RichTextElement
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

func (rt *RichTextFactory) Print2(text string) {
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

	commit := func(pos int) {
		rt.elements = append(rt.elements, RichTextElement{
			Text:  text[start:pos],
			Pos:   savedCursor,
			Style: rt.Style,
		})
		savedCursor = rt.cursor
		start = pos
	}

	breakLine := func() {
		rt.cursor.X = 0
		rt.cursor.Y += rt.LineSpacing
		savedCursor = rt.cursor
	}

	for pos, char := range text {
		if char == '\n' {
			commit(pos)
			breakLine()
			start = pos + 1
		} else {
			glyph := rl.GetGlyphInfo(font, char)

			charAdvance := float32(glyph.AdvanceX)
			if charAdvance == 0 {
				rec := rl.GetGlyphAtlasRec(font, char)
				charAdvance = rec.Width
			}
			charAdvance *= scaleFactor

			if rt.cursor.X+charAdvance > rt.width {
				commit(pos)
				breakLine()
				start = pos
			}

			rt.cursor.X += charAdvance
		}
	}

	if start < len(text) {
		commit(len(text))
	}
}

func (rt *RichTextFactory) Print(text string) []RichTextElement{
	if len(text) <= 0 {
		return []RichTextElement{}
	}
	var newElements []RichTextElement
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

	getTextSize := func(start, end int) float32{
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

	saveToken := func(tkStart, tkEnd int, tkSize float32){
		textSize += tkSize
		textEnd = tkEnd
	}

	printSavedToken := func() bool{
		if textEnd > textStart{
			newElements = append(newElements, RichTextElement{
				Text:  text[textStart:textEnd],
				Pos:   rt.cursor,
				Style: rt.Style,
			})

			rt.cursor.X += textSize
			textStart = textEnd
			textSize = 0

			return true
		}
		return false
	}

	iter := newIteratorForRT([]byte(text), font, rt.LineBreakRule)

	for iter.HasNext(){
		tkStart, tkEnd := iter.Next()

		if text[tkStart:tkEnd] == "\n"{
			printSavedToken()
			rt.cursor.Y += rt.LineSpacing
			rt.cursor.X = 0
			textStart = tkEnd
			textEnd = tkEnd
		}else{
			tkSize := getTextSize(tkStart, tkEnd)
			if rt.cursor.X + textSize + tkSize > rt.width{
				printSavedToken()
				if rt.cursor.X > 0 {
					rt.cursor.Y += rt.LineSpacing
					rt.cursor.X = 0
				}

				saveToken(tkStart, tkEnd, tkSize)

				if rt.cursor.X + textSize > rt.width{
					printSavedToken()
					rt.cursor.Y += rt.LineSpacing
					rt.cursor.X = 0
				}
			}else{
				saveToken(tkStart, tkEnd, tkSize)
				if rt.cursor.X + textSize > rt.width{
					printSavedToken()
					rt.cursor.Y += rt.LineSpacing
					rt.cursor.X = 0
				}
			}
		}
	}

	printSavedToken()

	return newElements
}

type iteratorForRT struct {
	text          []byte
	lineBreakRule int
	pos           int
}

func newIteratorForRT(textAsBytes []byte, font rl.Font, lineBreakRule int) *iteratorForRT {
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
