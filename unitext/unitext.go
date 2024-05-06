package unitext

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"unicode/utf8"

	"golang.org/x/image/math/fixed"

	"github.com/srwiley/rasterx"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/fontscan"
	meta "github.com/go-text/typesetting/opentype/api/metadata"
	"github.com/go-text/typesetting/shaping"

	"golang.org/x/exp/constraints"
)

var shaper shaping.HarfbuzzShaper

func RenderUnicodeText(text string, logger *log.Logger) image.Image {
	const fontSize = 20
	runes := StringToRunes(text)

	fontMap := fontscan.NewFontMap(logger)

	var aspect meta.Aspect
	aspect.SetDefaults()

	query := fontscan.Query{
		Families: []string{fontscan.Serif},
		Aspect:   aspect,
	}

	fontMap.SetQuery(query)

	fontMap.UseSystemFonts("./fnf-font-cache/")

	input := shaping.Input{
		Text: runes,

		RunStart: 0,
		RunEnd:   len(runes),

		Size: fixed.I(fontSize),

		// TODO : we are rendering path so I think just having left to right is fine..
		// But may be that is not the case.
		// do some research
		Direction: di.DirectionLTR, // left to right
	}

	inputs := shaping.SplitByFace(input, fontMap)

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

	renderer := Renderer{
		FontSize: fontSize,
		Color:    color.NRGBA{0, 255, 0, 255},
	}

	for _, out := range outputs {
		renderer.DrawShapedRunAt(out, canvas, dotX.Round(), dotY.Round())
		dotX += out.Advance
	}

	return canvas
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

func getNewStroker(img draw.Image, c color.Color, thick float64) *rasterx.Stroker {
	bounds := img.Bounds()

	scanner := rasterx.NewScannerGV(bounds.Dx(), bounds.Dy(), img, bounds)
	stroker := rasterx.NewStroker(bounds.Dx(), bounds.Dy(), scanner)
	stroker.SetColor(c)

	stroker.SetStroke(
		floatToFixed266(thick),
		fixed.I(1),
		rasterx.RoundCap, rasterx.RoundCap,
		rasterx.RoundGap,
		rasterx.Round,
	)

	return stroker
}

func getNewFiller(img draw.Image, c color.Color) *rasterx.Filler {
	bounds := img.Bounds()

	scanner := rasterx.NewScannerGV(bounds.Dx(), bounds.Dy(), img, bounds)
	filler := rasterx.NewFiller(bounds.Dx(), bounds.Dy(), scanner)
	filler.SetColor(c)

	return filler
}

func drawCircleLine(img draw.Image, x, y, r float64, c color.Color, thick float64) {
	s := getNewStroker(img, c, thick)
	rasterx.AddCircle(x, y, r, s)
	s.Draw()
}

func drawCircleFill(img draw.Image, x, y, r float64, c color.Color) {
	f := getNewFiller(img, c)
	rasterx.AddCircle(x, y, r, f)
	f.Draw()
}

func drawRectLine(img draw.Image, x, y, w, h float64, c color.Color, thick float64) {
	s := getNewStroker(img, c, thick)
	rasterx.AddRect(x, y, x+w, x+h, 0, s)
	s.Draw()
}

func drawRectFill(img draw.Image, x, y, w, h float64, c color.Color) {
	f := getNewFiller(img, c)
	rasterx.AddRect(x, y, x+w, x+h, 0, f)
	f.Draw()
}

func drawLine(img draw.Image, x1, y1, x2, y2 float64, c color.Color, thick float64) {
	s := getNewStroker(img, c, thick)

	s.Start(
		fixed.Point26_6{
			X: floatToFixed266(x1),
			Y: floatToFixed266(y1),
		})

	s.Line(
		fixed.Point26_6{
			X: floatToFixed266(x2),
			Y: floatToFixed266(y2),
		})

	s.Stop(false)

	s.Draw()
}

func fixed266ToFloat32(i fixed.Int26_6) float32 {
	return float32(i) / 64
}

func fixed266ToFloat64(i fixed.Int26_6) float64 {
	return float64(i) / 64
}

func floatToFixed266[F constraints.Float](f F) fixed.Int26_6 {
	return fixed.Int26_6(int(f * 64))
}
