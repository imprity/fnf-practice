package rl

/*
#include "raylib.h"
*/
import "C"

import (
	"image"
	"image/color"
	"unsafe"
)

// SwapScreenBuffer - Swap back buffer with front buffer (screen drawing)
func SwapScreenBuffer() {
	C.SwapScreenBuffer()
}

// PollInputEvents - Register all input events
func PollInputEvents() {
	C.PollInputEvents()
}

// WaitTime - Wait for some time (halt program execution)
func WaitTime(seconds float64) {
	cseconds := (C.double)(seconds)
	C.WaitTime(cseconds)
}

// NewImageFromImage - Returns new Image from Go image.Image
func NewImageFromImagePro(img image.Image, bgColor Color, alphaMultiply bool) *Image {
	size := img.Bounds().Size()

	cx := (C.int)(size.X)
	cy := (C.int)(size.Y)
	cBgColor := colorCptr(bgColor)
	ret := C.GenImageColor(cx, cy, *cBgColor)

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			imgColor := img.At(x, y)

			rlColor := Color{}

			if alphaMultiply {
				converted := color.RGBAModel.Convert(imgColor)
				v := converted.(color.RGBA)

				rlColor = Color{
					R: v.R,
					G: v.G,
					B: v.B,
					A: v.A,
				}
			} else {
				converted := color.NRGBAModel.Convert(imgColor)
				v := converted.(color.NRGBA)

				rlColor = Color{
					R: v.R,
					G: v.G,
					B: v.B,
					A: v.A,
				}
			}

			cRlColor := colorCptr(rlColor)

			cx = (C.int)(x)
			cy = (C.int)(y)
			C.ImageDrawPixel(&ret, cx, cy, *cRlColor)
		}
	}

	v := newImageFromPointer(unsafe.Pointer(&ret))
	return v
}

func LoadFontDataSdf(
	fileData []byte,
	fontSize int32,
	codepoints []rune,

	sdfPadding int32,
	sdfOnEdgeValue uint8,
	sdfPixelDistScale float32,
) []GlyphInfo {
	cFileData := (*C.uchar)(unsafe.Pointer(&fileData[0]))
	cDataSize := (C.int)(len(fileData))
	cFontSize := (C.int)(fontSize)

	// we do this because we want to make sure that zero length array is passed as nil
	var cCodepoints (*C.int) = nil
	if len(codepoints) > 0 {
		cCodepoints = (*C.int)(unsafe.SliceData(codepoints))
	}

	// It's kinda sad to use hard coded value but this is what happens if you pass zero length
	// array
	var cCodePointCount C.int = 95

	if len(codepoints) > 0 {
		cCodePointCount = (C.int)(len(codepoints))
	}

	cSdfPadding := (C.int)(sdfPadding)
	cSdfOnEdgeValue := (C.uchar)(sdfOnEdgeValue)
	cSdfPixelDistScale := (C.float)(sdfPixelDistScale)

	ret := C.LoadFontDataSdf(
		cFileData, cDataSize,
		cFontSize,
		cCodepoints, cCodePointCount,

		cSdfPadding,
		cSdfOnEdgeValue,
		cSdfPixelDistScale,
	)

	v := unsafe.Slice((*GlyphInfo)(unsafe.Pointer(ret)), cCodePointCount)
	return v
}

func GenImageFontAtlas(glyphs []GlyphInfo, fontSize, padding, packMethod int32) (*Image, []Rectangle) {
	cGlyphs := (unsafe.SliceData(glyphs)).cptr()

	cGlyphsCount := (C.int)(len(glyphs))

	cFontSize := (C.int)(fontSize)
	cPadding := (C.int)(padding)
	cPackMethod := (C.int)(packMethod)

	var cRectPointer *(C.Rectangle)

	cImage := C.GenImageFontAtlas(
		cGlyphs,
		(**C.Rectangle)(unsafe.Pointer(&cRectPointer)), cGlyphsCount, cFontSize, cPadding, cPackMethod,
	)

	if unsafe.Pointer(cRectPointer) == nil {
		return newImageFromPointer(unsafe.Pointer(&cImage)), []Rectangle{}
	} else {
		return newImageFromPointer(unsafe.Pointer(&cImage)), unsafe.Slice(
			(*Rectangle)(unsafe.Pointer(cRectPointer)),
			len(glyphs),
		)
	}

}

// SetFontCharGlyphs - Set font Chars
func SetFontCharGlyphs(font *Font, glyphs []GlyphInfo) {
	font.Chars = unsafe.SliceData(glyphs)
	font.CharsCount = int32(len(glyphs))
}

// SetFontRecs - Set font Recs
func SetFontRecs(font *Font, recs []Rectangle) {
	font.Recs = unsafe.SliceData(recs)
}
