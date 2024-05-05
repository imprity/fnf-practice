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
