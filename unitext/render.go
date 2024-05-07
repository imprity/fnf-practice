package unitext

// This is basically a copy of https://github.com/go-text/render
// But it uses oudated version of go-text/typesetting
// and I thought it would be easier to manager if it's in this single file

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"

	"golang.org/x/exp/constraints"

	scale "golang.org/x/image/draw"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/tiff" // load image formats for users of the API

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"github.com/go-text/typesetting/opentype/api"
	"github.com/go-text/typesetting/shaping"
)

// Renderer defines a type that can render strings to a bitmap canvas.
// The size and look of output depends on the various fields in this struct.
// Developers should provide suitable output images for their draw requests.
// This type is not thread safe so instances should be used from only 1 goroutine.
type renderer struct {
	// FontSize defines the point size of output text, commonly between 10 and 14 for regular text
	FontSize float32
	// Color is the pen colour for rendering
	Color color.Color

	filler      *rasterx.Filler
	fillerScale float32
}

// DrawShapedRunAt will rasterise the given shaper run into the output image using font face referenced in the shaping.
// The text will be drawn starting at the startX, startY pixel position.
// Note that startX and startY are not multiplied by the `PixScale` value as they refer to output coordinates.
// The return value is the X pixel position of the end of the drawn string.
func (r *renderer) DrawShapedRunAt(run shaping.Output, img draw.Image, startX, startY int) int {
	scale := r.FontSize / float32(run.Face.Upem())
	r.fillerScale = scale

	b := img.Bounds()
	scanner := rasterx.NewScannerGV(b.Dx(), b.Dy(), img, b)
	f := rasterx.NewFiller(b.Dx(), b.Dy(), scanner)
	r.filler = f
	f.SetColor(r.Color)
	x := float32(startX)
	y := float32(startY)
	for _, g := range run.Glyphs {
		xPos := x + fixed266ToFloat32(g.XOffset)
		yPos := y - fixed266ToFloat32(g.YOffset)
		data := run.Face.GlyphData(g.GlyphID)
		switch format := data.(type) {
		case api.GlyphOutline:
			r.drawOutline(g, format, f, scale, xPos, yPos)
		case api.GlyphBitmap:
			_ = r.drawBitmap(g, format, img, xPos, yPos)
		case api.GlyphSVG:
			_ = r.drawSVG(g, format, img, xPos, yPos)
		}

		x += fixed266ToFloat32(g.XAdvance)
	}
	f.Draw()
	r.filler = nil
	return int(math.Ceil(float64(x)))
}

func (r *renderer) drawOutline(g shaping.Glyph, bitmap api.GlyphOutline, f *rasterx.Filler, scale float32, x, y float32) {
	for _, s := range bitmap.Segments {
		switch s.Op {
		case api.SegmentOpMoveTo:
			f.Start(fixed.Point26_6{X: floatToFixed266(s.Args[0].X*scale + x), Y: floatToFixed266(-s.Args[0].Y*scale + y)})
		case api.SegmentOpLineTo:
			f.Line(fixed.Point26_6{X: floatToFixed266(s.Args[0].X*scale + x), Y: floatToFixed266(-s.Args[0].Y*scale + y)})
		case api.SegmentOpQuadTo:
			f.QuadBezier(fixed.Point26_6{X: floatToFixed266(s.Args[0].X*scale + x), Y: floatToFixed266(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: floatToFixed266(s.Args[1].X*scale + x), Y: floatToFixed266(-s.Args[1].Y*scale + y)})
		case api.SegmentOpCubeTo:
			f.CubeBezier(fixed.Point26_6{X: floatToFixed266(s.Args[0].X*scale + x), Y: floatToFixed266(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: floatToFixed266(s.Args[1].X*scale + x), Y: floatToFixed266(-s.Args[1].Y*scale + y)},
				fixed.Point26_6{X: floatToFixed266(s.Args[2].X*scale + x), Y: floatToFixed266(-s.Args[2].Y*scale + y)})
		}
	}
	f.Stop(true)
}

func (r *renderer) drawBitmap(g shaping.Glyph, bitmap api.GlyphBitmap, img draw.Image, x, y float32) error {
	// scaled glyph rect content
	top := y - fixed266ToFloat32(g.YBearing)
	bottom := top - fixed266ToFloat32(g.Height)
	right := x + fixed266ToFloat32(g.Width)
	switch bitmap.Format {
	case api.BlackAndWhite:
		rec := image.Rect(0, 0, bitmap.Width, bitmap.Height)
		sub := image.NewPaletted(rec, color.Palette{color.Transparent, r.Color})

		for i := range sub.Pix {
			sub.Pix[i] = bitAt(bitmap.Data, i)
		}

		rect := image.Rect(int(x), int(top), int(right), int(bottom))
		scale.NearestNeighbor.Scale(img, rect, sub, sub.Bounds(), draw.Over, nil)
	case api.JPG, api.PNG, api.TIFF:
		pix, _, err := image.Decode(bytes.NewReader(bitmap.Data))
		if err != nil {
			return err
		}

		rect := image.Rect(int(x), int(top), int(right), int(bottom))
		scale.BiLinear.Scale(img, rect, pix, pix.Bounds(), draw.Over, nil)
	}

	if bitmap.Outline != nil {
		r.drawOutline(g, *bitmap.Outline, r.filler, r.fillerScale, x, y)
	}
	return nil
}

func (r *renderer) drawSVG(g shaping.Glyph, svg api.GlyphSVG, img draw.Image, x, y float32) error {
	pixWidth := g.Width.Round()
	pixHeight := (-g.Height).Round()
	pix, err := renderSVGStream(bytes.NewReader(svg.Source), pixWidth, pixHeight)
	if err != nil {
		return err
	}

	rect := image.Rect((g.XBearing).Round(), (-g.YBearing).Round(), pixWidth, pixHeight)
	draw.Draw(img, rect.Add(image.Point{X: int(x), Y: int(y)}), pix, image.Point{}, draw.Over)

	// ignore the svg.Outline shapes, as they are a fallback which we won't use
	return nil
}

func renderSVGStream(stream io.Reader, width, height int) (*image.NRGBA, error) {
	icon, err := oksvg.ReadIconStream(stream)
	if err != nil {
		return nil, err
	}

	iconAspect := float32(icon.ViewBox.W / icon.ViewBox.H)
	viewAspect := float32(width) / float32(height)
	imgW, imgH := width, height
	if viewAspect > iconAspect {
		imgW = int(float32(height) * iconAspect)
	} else if viewAspect < iconAspect {
		imgH = int(float32(width) / iconAspect)
	}

	icon.SetTarget(icon.ViewBox.X, icon.ViewBox.Y, float64(imgW), float64(imgH))

	out := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	scanner := rasterx.NewScannerGV(int(icon.ViewBox.W), int(icon.ViewBox.H), out, out.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)

	icon.Draw(raster, 1)
	return out, nil
}

// ==============================
// Helper functions
// ==============================

// bitAt returns the bit at the given index in the byte slice.
func bitAt(b []byte, i int) byte {
	return (b[i/8] >> (7 - i%8)) & 1
}

func fixed266Div(a, b fixed.Int26_6)fixed.Int26_6{
	return a * 64 / b
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

// ==============================
// Debug drawing
// ==============================

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

