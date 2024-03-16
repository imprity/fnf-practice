package kitty

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image"
	"image/color"
	"math"
)

func DrawLine(dst *ebiten.Image, from, to Vec2, stroke float64, c Color) {
	vector.StrokeLine(
		dst,
		float32(from.X), float32(from.Y), float32(to.X), float32(to.Y),
		float32(stroke),
		ToImageColor(c),
		true,
	)
}

func DrawRect(dst *ebiten.Image, rect FRect, c Color) {
	vector.DrawFilledRect(
		dst,
		float32(rect.X), float32(rect.Y),
		float32(rect.W), float32(rect.H),
		ToImageColor(c),
		true,
	)
}

func StrokeRect(dst *ebiten.Image, rect FRect, stroke float64, c Color) {
	vector.StrokeRect(
		dst,
		float32(rect.X), float32(rect.Y),
		float32(rect.W), float32(rect.H),
		float32(stroke),
		ToImageColor(c),
		true,
	)
}

func DrawCircle(dst *ebiten.Image, circle Circle, c Color) {
	vector.DrawFilledCircle(
		dst,
		float32(circle.X), float32(circle.Y),
		float32(circle.R),
		ToImageColor(c),
		true,
	)
}

func StrokeCircle(dst *ebiten.Image, circle Circle, stroke float64, c Color) {
	vector.StrokeCircle(
		dst,
		float32(circle.X), float32(circle.Y),
		float32(circle.R),
		float32(stroke),
		ToImageColor(c),
		true,
	)
}

func getRoundRectPath(rect FRect, radius float64) vector.Path {
	radius = min(radius, min(rect.W*0.5, rect.H*0.5)) //clamp the radius to the size of rect

	inLeftTop := Vec2{rect.X + radius, rect.Y + radius}
	inRightTop := Vec2{rect.X + rect.W - radius, rect.Y + radius}
	inLeftBottom := Vec2{rect.X + radius, rect.Y + rect.H - radius}
	inRightBottom := Vec2{rect.X + rect.W - radius, rect.Y + rect.H - radius}

	const (
		d0   float32 = math.Pi * 0.0
		d90  float32 = math.Pi * 0.5
		d180 float32 = math.Pi * 1.0
		d270 float32 = math.Pi * 1.5
		d360 float32 = math.Pi * 2.0
	)

	var path vector.Path

	path.Arc(float32(inLeftTop.X), float32(inLeftTop.Y), float32(radius), d180, d270, vector.Clockwise)
	path.LineTo(float32(inRightTop.X), float32(inRightTop.Y-radius))

	path.Arc(float32(inRightTop.X), float32(inRightTop.Y), float32(radius), d270, d0, vector.Clockwise)
	path.LineTo(float32(inRightBottom.X+radius), float32(inRightBottom.Y))

	path.Arc(float32(inRightBottom.X), float32(inRightBottom.Y), float32(radius), d0, d90, vector.Clockwise)
	path.LineTo(float32(inLeftBottom.X), float32(inLeftBottom.Y+radius))

	path.Arc(float32(inLeftBottom.X), float32(inLeftBottom.Y), float32(radius), d90, d180, vector.Clockwise)
	path.Close()

	return path
}

func DrawRoundRect(dst *ebiten.Image, rect FRect, radius float64, c Color) {
	path := getRoundRectPath(rect, radius)
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	drawVerticesForUtil(dst, vs, is, ToImageColor(c), true)
}

func StrokeRoundRect(dst *ebiten.Image, rect FRect, radius float64, stroke float64, c Color) {
	path := getRoundRectPath(rect, radius)
	strokeOp := &vector.StrokeOptions{}
	strokeOp.Width = float32(stroke)
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	drawVerticesForUtil(dst, vs, is, ToImageColor(c), true)
}

// below codes are directrly copied from github.com/hajimehoshi/ebiten/v2/vector
// since it doesn't expose it to end users

// image filled with white for drawing triangles
// it's kinda hacky but ebiten library does exactly this so I don't feel
// too bad about it...
var (
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	initWhiteImage()
}

func initWhiteImage() {
	b := whiteImage.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	// This is hacky, but WritePixels is better than Fill in term of automatic texture packing.
	whiteImage.WritePixels(pix)
}

func drawVerticesForUtil(dst *ebiten.Image, vs []ebiten.Vertex, is []uint16, clr color.Color, antialias bool) {
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	op.AntiAlias = antialias
	dst.DrawTriangles(vs, is, whiteSubImage, op)
}

func DrawImageOnImageRect(src *ebiten.Image, dst *ebiten.Image, srcRect *FRect, dstRect *FRect) {
	if srcRect == nil {
		srcRect = new(FRect)
		srcRect.X = 0
		srcRect.Y = 0
		srcRect.W = float64(src.Bounds().Dx())
		srcRect.H = float64(src.Bounds().Dy())
	}

	if dstRect == nil {
		dstRect = new(FRect)
		dstRect.X = 0
		dstRect.Y = 0
		dstRect.W = float64(dst.Bounds().Dx())
		dstRect.H = float64(dst.Bounds().Dy())
	}

	if int(srcRect.W) == 0 || int(srcRect.H) == 0 {
		return
	}

	if int(dstRect.W) == 0 || int(dstRect.H) == 0 {
		return
	}

	// TODO : we should check if srcRect contains the src at all
	// TODO : we should check if dstRect contains the dst at all

	//go fucking sucks....
	srcSubRect := image.Rect(
		int(srcRect.X)+src.Bounds().Min.X,
		int(srcRect.Y)+src.Bounds().Min.Y,
		int(srcRect.X+srcRect.W)+src.Bounds().Min.X,
		int(srcRect.Y+srcRect.H)+src.Bounds().Min.Y,
	)
	srcSubImage := src.SubImage(srcSubRect).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear

	op.GeoM.Scale(
		dstRect.W/srcRect.W, dstRect.H/srcRect.H,
	)
	op.GeoM.Translate(
		dstRect.X, dstRect.Y,
	)

	dst.DrawImage(srcSubImage, op)
}
