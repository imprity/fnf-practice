package kitty

import (
	"image"
	//"image/color"

	"os"
	"bufio"
	
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

var defaultFont font.Face
var defaultFontInitialized bool = false

func GetDefaultFont() font.Face{
	if !defaultFontInitialized{
		otf, err := opentype.Parse(defaultFontData[:])
		if err != nil {	panic(err) }

		defaultFont, err = opentype.NewFace(otf, &opentype.FaceOptions{
			Size    : 18,
			DPI     : 72,
			Hinting : font.HintingFull,
		})
		if err != nil {	panic(err) }
	}

	defaultFontInitialized = true

	return defaultFont
}

func DrawTextCentered(dst *ebiten.Image, at Vec2, str string, fontFace font.Face, c Color){
	var strImageRect  image.Rectangle = text.BoundString(fontFace, str)
	strRectCenter := Vec2{}

	//caculate rect center
	{
		minX := float64(strImageRect.Min.X)
		minY := float64(strImageRect.Min.Y)

		maxX := float64(strImageRect.Max.X)
		maxY := float64(strImageRect.Max.Y)

		strRectCenter.X = (minX + maxX) * 0.5
		strRectCenter.Y = (minY + maxY) * 0.5
	}

	translation := at.SubV(strRectCenter)

	op := &ebiten.DrawImageOptions{}

	{
		op.GeoM.Translate(translation.X, translation.Y)
		op.ColorScale.ScaleWithColor(ToImageColor(c))
	} 

	text.DrawWithOptions(dst, str, fontFace, op)
}

func fitTextInRectImpl(dst *ebiten.Image, rect FRect, str string, fontFace font.Face, c Color, inHoz bool, inVert bool){
	var strImageRect  image.Rectangle = text.BoundString(fontFace, str)
	strRectCenter := Vec2{}

	//caculate rect center
	{
		minX := float64(strImageRect.Min.X)
		minY := float64(strImageRect.Min.Y)

		maxX := float64(strImageRect.Max.X)
		maxY := float64(strImageRect.Max.Y)

		strRectCenter.X = (minX + maxX) * 0.5
		strRectCenter.Y = (minY + maxY) * 0.5
	}

	//var scale Vec2 = Vec2{1.0, 1.0}
	var scale float64 = 1.0

	if inHoz && float64(strImageRect.Dx()) > rect.W{
		scale = min(scale, rect.W / float64(strImageRect.Dx()))
	}
	if inVert && float64(strImageRect.Dy()) > rect.H{
		scale = min(scale, rect.H / float64(strImageRect.Dy()))
	}

	translation := rect.Center().SubV(strRectCenter.Mul1(scale))

	op := &ebiten.DrawImageOptions{}

	{
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(translation.X, translation.Y)

		op.ColorScale.ScaleWithColor(ToImageColor(c))
	} 

	text.DrawWithOptions(dst, str, fontFace, op)
}

func FitTextInRectHoz(dst *ebiten.Image, r FRect, str string, f font.Face, c Color){
	fitTextInRectImpl(dst, r, str, f, c, true, false)
}

func FitTextInRectVert(dst *ebiten.Image, r FRect, str string, f font.Face, c Color){
	fitTextInRectImpl(dst, r, str, f, c, false, true)
}

func FitTextInRect(dst *ebiten.Image, r FRect, str string, f font.Face, c Color){
	fitTextInRectImpl(dst, r, str, f, c, true, true)
}

func GetFontHeight(fontFace font.Face) int{
	return fontFace.Metrics().Height.Round()
}

func GetFontHeightF(fontFace font.Face) float64{
	heightI := fontFace.Metrics().Height
	
	heightF := float64(heightI) / float64(1 << 6)
	return heightF
}

func LoadFontFace(path string, size float64) (font.Face, error){
	const dpi      int = 72

	file, err := os.Open(path)
	if err != nil {	return nil, err }
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {	return nil, err }

	bs := make([]byte, stat.Size())
	_, err = bufio.NewReader(file).Read(bs)
	if err != nil {	return nil, err }

	otf, err := opentype.Parse(bs)
	if err != nil {	return nil, err }

	face ,err := opentype.NewFace(otf, &opentype.FaceOptions{
		Size:    size,
		DPI:     float64(dpi), //it's a float???
		Hinting: font.HintingFull,
	})
	if err != nil {	return nil, err }

	return face, nil
}