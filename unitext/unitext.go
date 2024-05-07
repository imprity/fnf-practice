// Package unitext provides functionality to render unicode text.
// Basically a frontend for typesetting packages.
// https://pkg.go.dev/github.com/go-text/typesetting
//
// This package is not that robust and doesn't expect multiline text.
// Also it's pretty slow so don't use this for text editor or whatever.
// It's only meant to be used for fnf-practice.
// Look at example.go to see how to use this package.
package unitext

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"golang.org/x/image/math/fixed"

	//"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/fontscan"
	meta "github.com/go-text/typesetting/opentype/api/metadata"
	"github.com/go-text/typesetting/shaping"
)


// Logger to use. If nil uses log.Default()
var Logger fontscan.Logger

// Unitext looks up system fonts to determine what font to use
// Which is slow so we cache the lookup result
// Set CacheDir to set directory to store the lookup result
// If not set uses appropriate directory
//
// Except for android, you HAVE TO set directory for android
var CacheDir string

var shaper shaping.HarfbuzzShaper

func RenderUnicodeText(text string, desiredFont DesiredFont, fontSize float32, textColor color.Color) (image.Image, error) {
	runes := StringToRunes(text)

	fontMap := fontscan.NewFontMap(Logger)

	var aspect meta.Aspect

	aspect.Style = desiredFont.Style
	aspect.Weight = desiredFont.Weight
	aspect.Stretch = desiredFont.Stretch

	query := fontscan.Query{
		Families: desiredFont.Families,
		Aspect:   aspect,
	}

	fontMap.SetQuery(query)

	err := fontMap.UseSystemFonts(CacheDir)
	if err != nil{
		return nil, err
	}

	input := shaping.Input{
		Text: runes,

		RunStart: 0,
		RunEnd:   len(runes),

		Size: floatToFixed266(fontSize),
	}

	segmenter := shaping.Segmenter{}

	inputs := segmenter.Split(input, fontMap)

	var outputs []shaping.Output

	for _, in := range inputs {
		outputs = append(outputs, shaper.Shape(in))
	}

	// ==========================
	// calculate boundary
	// ==========================

	var ascent fixed.Int26_6 = math.MaxInt32
	var descentAndGap fixed.Int26_6 = math.MinInt32
	var maxX fixed.Int26_6 = 0

	for _, out := range outputs {
		maxX += out.Advance

		ascent = min(ascent, -out.LineBounds.Ascent)
		descentAndGap = max(descentAndGap, -out.LineBounds.Descent+out.LineBounds.Gap)
	}

	// ==========================
	// actual drawing
	// ==========================

	canvas := image.NewRGBA(image.Rect(0, 0, maxX.Ceil(), (descentAndGap - ascent).Ceil()))

	dotX := fixed.I(0)
	dotY := -ascent

	renderer := renderer{
		FontSize: fontSize,
		Color:    textColor,
	}

	for _, out := range outputs {
		renderer.DrawShapedRunAt(out, canvas, dotX.Round(), dotY.Round())
		dotX += out.Advance
	}

	return canvas, nil
}

func StringToRunes(text string) []rune {
	strBytes := []byte(text)

	var runes []rune

	for {
		r, size := utf8.DecodeRune(strBytes)

		if r == utf8.RuneError && size == 0 {
			break
		} else if r == utf8.RuneError {
			byteString := fmt.Sprintf("<0x%X> ", strBytes[0])
			for _, char := range byteString {
				runes = append(runes, char)
			}
		} else {
			runes = append(runes, r)
		}
		strBytes = strBytes[size:]
	}

	return runes
}

