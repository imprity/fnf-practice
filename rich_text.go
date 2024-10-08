package fnf

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type RichTextStyle struct {
	FontSize float32

	Font FnfFont

	Fill, Stroke FnfColor
	StrokeWidth  float32
}

type RichTextElement struct {
	Text string

	Bound rl.Rectangle

	Style *RichTextStyle

	Metadata int64
}

type RichTextLineBreakRule int

const (
	LineBreakWord RichTextLineBreakRule = iota
	LineBreakChar
	LineBreakNever
	LineBreakRuleSize
)

var RichTextLineBreakRuleStrs = [LineBreakRuleSize]string{
	"word",
	"character",
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

	LineBreakRule RichTextLineBreakRule

	style *RichTextStyle

	width float32

	cursor float32

	elements []RichTextElement
}

func NewRichTextFactory(width float32) *RichTextFactory {
	return &RichTextFactory{
		style: &RichTextStyle{
			FontSize: 30,
			Font:     FontRegular,
			Fill:     Col(0, 0, 0, 255),
		},
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

	breakLine := func() {
		rt.elements = append(rt.elements, RichTextElement{
			Text: "\n",
			Bound: rl.Rectangle{
				X: rt.cursor, Y: 0,
				Width: 0, Height: 0,
			},
			Style:    rt.style,
			Metadata: rt.Metadata,
		})
		rt.cursor = 0
	}

	// check if cursor is out side width
	if rt.LineBreakRule != LineBreakNever && rt.cursor > rt.width {
		breakLine()
	}

	font := rt.style.Font

	scaleFactor := rt.style.FontSize / f32(font.BaseSize())

	getTextSize := func(start, end int) float32 {
		textSize := float32(0)
		for _, char := range text[start:end] {
			glyph := rl.GetGlyphInfo(font.Font, char)

			charSize := float32(glyph.AdvanceX)
			if charSize == 0 {
				rec := rl.GetGlyphAtlasRec(font.Font, char)
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
					X: rt.cursor, Y: 0,
					Width: textSize, Height: rt.style.FontSize,
				},
				Style:    rt.style,
				Metadata: rt.Metadata,
			})

			rt.cursor += textSize
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
				if rt.cursor+textSize+tkSize > rt.width {
					printSavedToken()
					if rt.cursor > 0 {
						breakLine()
					}

					saveToken(tkStart, tkEnd, tkSize)

					if rt.cursor+textSize > rt.width {
						printSavedToken()
						breakLine()
					}
				} else {
					saveToken(tkStart, tkEnd, tkSize)
					if rt.cursor+textSize > rt.width {
						printSavedToken()
						breakLine()
					}
				}
			}
		}
	}

	printSavedToken()
}

func EscapeRichText(str string) string {
	replacer := strings.NewReplacer("<<", "<", ">>", ">")
	return replacer.Replace(str)
}

// custom rich text syntax
//
//	<size 50>          FontSize
//
//	<fill #aabbccdd>   Fill
//	<stroke #aabbcc>   Stroke
//
//	<thick 1.2>        StrokeWidth
//
//	<font FontRegular> Font
//
//	<meta 1>           Metadata
//
// can be combined like this <fill #aabbccdd stroke #ffffff thick 1.3>
//
//	<< escaped <
//	>> escaped >
func (rt *RichTextFactory) PrintRichText(text string) {
	getNext := func(at int) (byte, bool) {
		if at+1 >= len(text) || at+1 < 0 {
			return 0, false
		}

		return text[at+1], true
	}

	findBracket := func(from int, findClosing bool) (int, bool) {
		var toFind byte = '<'
		if findClosing {
			toFind = '>'
		}

		i := from
		for i < len(text) {
			if text[i] == toFind {
				next, hasNext := getNext(i)
				if hasNext && next == toFind {
					i += 2
				} else {
					return i, true
				}
			}

			i += 1
		}

		return i, false
	}

	findOpen := func(from int) (int, bool) {
		return findBracket(from, false)
	}

	findClose := func(from int) (int, bool) {
		return findBracket(from, true)
	}

	// returns parsed color and if it succeeded
	parseColor := func(str string) (FnfColor, bool) {
		if len(str) <= 0 {
			return FnfColor{}, false
		}
		if str[0] != '#' {
			return FnfColor{}, false
		}

		n, err := strconv.ParseUint(str[1:], 16, 32)

		if err == nil {
			color := FnfColor{}
			if len(str) > 7 {
				color.A = uint8(n & 0xFF)
				n = n >> 8
			} else {
				color.A = 0xFF
			}
			color.B = uint8(n & 0xFF)
			n = n >> 8
			color.G = uint8(n & 0xFF)
			n = n >> 8
			color.R = uint8(n & 0xFF)

			return color, true
		} else {
			return FnfColor{}, false
		}
	}

	startingStyle := rt.Style()

	// parses style text
	// don't include opening and closing brackets
	parseTag := func(tagString string) {
		words := strings.Fields(tagString)

		cursor := 0

		newStyle := startingStyle

		for cursor+1 < len(words) {
			word := words[cursor]
			nextWord := words[cursor+1]

			switch word {
			case "size":
				if f, err := strconv.ParseFloat(nextWord, 32); err == nil {
					newStyle.FontSize = float32(f)
				}
			case "fill":
				if fill, success := parseColor(nextWord); success {
					newStyle.Fill = fill
				}
			case "stroke":
				if stroke, success := parseColor(nextWord); success {
					newStyle.Stroke = stroke
				}
			case "thick":
				if f, err := strconv.ParseFloat(nextWord, 32); err == nil {
					newStyle.StrokeWidth = float32(f)
				}
			case "font":
				if font, success := GetFontFromName(nextWord); success {
					newStyle.Font = font
				}
			case "meta":
				if meta, err := strconv.ParseInt(nextWord, 10, 64); err == nil {
					rt.Metadata = meta
				}
			}
			cursor += 2
		}

		rt.SetStyle(newStyle)
	}

	cursor := 0

	for cursor < len(text) {
		open, foundOpen := findOpen(cursor)

		if foundOpen {
			closing, foundClosing := findClose(open)

			if foundClosing {
				rt.Print(EscapeRichText(text[cursor:open]))
				parseTag(EscapeRichText(text[open+1 : closing]))
				cursor = closing + 1
			} else {
				rt.Print(EscapeRichText(text[cursor:]))
				break
			}
		} else {
			rt.Print(EscapeRichText(text[cursor:]))
			break
		}
	}
}

func RichTextStyleToStr(style RichTextStyle) string {
	colorToStr := func(color FnfColor) string {
		return fmt.Sprintf("#%02X%02X%02X%02X", color.R, color.G, color.B, color.A)
	}

	str := fmt.Sprintf(
		"<size %.4f fill %s stroke %s thick %.4f ",
		style.FontSize, colorToStr(style.Fill), colorToStr(style.Stroke), style.StrokeWidth)

	if fontName, ok := GetNameFromFont(style.Font); ok {
		str += fmt.Sprintf("font %s >", EscapeRichText(fontName))
	} else {
		str += " >"
	}

	return str
}

func (rt *RichTextFactory) Elements(
	align RichTextAlign,
	lineMargin float32,
	emptyLineHeight float32,
) []RichTextElement {
	// align texts in line to their most bottom
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

	SetElementsLineSpacing(rt.elements, lineMargin, emptyLineHeight)
	AlignElements(rt.elements, align)

	return rt.elements
}

func ElementsBound(elements []RichTextElement) rl.Rectangle {
	if len(elements) <= 0 {
		return rl.Rectangle{}
	}

	bound := elements[0].Bound

	for i := 1; i < len(elements); i++ {
		bound = RectUnion(bound, elements[i].Bound)
	}

	return bound
}

func AlignElements(elements []RichTextElement, align RichTextAlign) {
	iter := NewRTElineIterator(elements)

	totalBound := ElementsBound(elements)

	for iter.HasNext() {
		b, e := iter.Next()

		lineBound := ElementsBound(elements[b:e])

		var offsetX float32

		switch align {
		case TextAlignLeft:
			offsetX = -lineBound.X
		case TextAlignRight:
			offsetX = totalBound.Width - lineBound.Width - lineBound.X
		case TextAlignCenter:
			offsetX = totalBound.Width*0.5 - lineBound.Width*0.5 - lineBound.X
		}

		for i := b; i < e; i++ {
			elements[i].Bound.X += offsetX
		}
	}
}

func SetElementsLineSpacing(elements []RichTextElement, margin float32, emptyLineHeight float32) {
	// bring all the elements to the top
	iter := NewRTElineIterator(elements)

	for iter.HasNext() {
		b, e := iter.Next()

		bound := ElementsBound(elements[b:e])

		for i := b; i < e; i++ {
			elements[i].Bound.Y -= bound.Y
		}
	}

	// calculate new y offset
	iter = NewRTElineIterator(elements)
	offsetY := float32(0)

	for iter.HasNext() {
		b, e := iter.Next()

		if e-b == 1 && elements[b].Text == "\n" {
			elements[b].Bound.Y += offsetY
			offsetY += margin + emptyLineHeight
		} else {
			bound := ElementsBound(elements[b:e])

			for i := b; i < e; i++ {
				elements[i].Bound.Y += offsetY
			}

			offsetY += margin + bound.Height
		}
	}
}

func DrawTextElements(elements []RichTextElement, x, y float32, tint FnfColor) {
	for _, e := range elements {
		pos := RectPos(e.Bound)
		pos.X += x
		pos.Y += y

		fill := ToRlColor(TintColor(e.Style.Fill, tint))
		stroke := ToRlColor(TintColor(e.Style.Stroke, tint))

		if e.Style.StrokeWidth > 0 {
			DrawTextOutlined(e.Style.Font, e.Text, pos,
				e.Style.FontSize, 0,
				fill, stroke,
				e.Style.StrokeWidth,
			)
		} else {
			DrawText(e.Style.Font, e.Text, pos,
				e.Style.FontSize, 0, fill,
			)
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

	for it.pos < len(it.elements) {
		if it.elements[it.pos].Text == "\n" {
			it.pos += 1
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
