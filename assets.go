package main

import (
	_ "embed"
	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets/arrows_outer.png
var arrowsOuterBytes []byte

//go:embed assets/arrows_inner.png
var arrowsInnerBytes []byte

var ArrowsOuterTex rl.Texture2D
var ArrowsInnerTex rl.Texture2D

var ArrowsRects [NoteDirSize]rl.Rectangle

//go:embed assets/arrows_glow.png
var arrowsGlowBytes []byte

var ArrowsGlowTex rl.Texture2D
var ArrowsGlowRects [NoteDirSize]rl.Rectangle

//go:embed "assets/background 1.png"
var backgroundBytes []byte
var PrettyBackground rl.Texture2D

//go:embed "assets/bad.png"
var hitRatingBadBytes []byte
//go:embed "assets/good.png"
var hitRatingGoodBytes []byte
//go:embed "assets/sick.png"
var hitRatingSickBytes []byte

var HitRatingTexs [HitRatingSize]rl.Texture2D

func InitAssets() {
	outerImg := rl.LoadImageFromMemory(".png", arrowsOuterBytes, int32(len(arrowsOuterBytes)))
	innerImg := rl.LoadImageFromMemory(".png", arrowsInnerBytes, int32(len(arrowsInnerBytes)))

	rl.ImageAlphaPremultiply(outerImg)
	rl.ImageAlphaPremultiply(innerImg)

	ArrowsOuterTex = rl.LoadTextureFromImage(outerImg)
	ArrowsInnerTex = rl.LoadTextureFromImage(innerImg)

	if outerImg.Width != innerImg.Width || outerImg.Height != innerImg.Height{
		ErrorLogger.Fatal("Arrow inner and outer images should have same size")
	}

	// NOTE : we will assume that we can get arrow rects
	// by just devding the width by 4

	width := float32(ArrowsOuterTex.Width) / 4.0

	for i:=NoteDir(0); i<NoteDirSize; i++{
		x := float32(i) * width
		ArrowsRects[i] = rl.Rectangle{
			x, 0, width, float32(ArrowsOuterTex.Height),
		}
	}

	glowImg := rl.LoadImageFromMemory(".png", arrowsGlowBytes, int32(len(arrowsGlowBytes)))
	rl.ImageAlphaPremultiply(glowImg)
	ArrowsGlowTex = rl.LoadTextureFromImage(glowImg)

	// NOTE : same goes for glow arrows

	width = float32(ArrowsGlowTex.Width) / 4.0

	for i:=NoteDir(0); i<NoteDirSize; i++{
		x := float32(i) * width
		ArrowsGlowRects[i] = rl.Rectangle{
			x, 0, width, float32(ArrowsGlowTex.Height),
		}
	}

	backgroundImg := rl.LoadImageFromMemory(".png", backgroundBytes, int32(len(backgroundBytes)))
	PrettyBackground = rl.LoadTextureFromImage(backgroundImg)

	var ratingImgs [HitRatingSize]*rl.Image

	ratingImgs[HitRatingBad]  = rl.LoadImageFromMemory(".png", hitRatingBadBytes, int32(len(hitRatingBadBytes)))
	ratingImgs[HitRatingGood] = rl.LoadImageFromMemory(".png", hitRatingGoodBytes, int32(len(hitRatingGoodBytes)))
	ratingImgs[HitRatingSick] = rl.LoadImageFromMemory(".png", hitRatingSickBytes, int32(len(hitRatingSickBytes)))

	for _, img := range ratingImgs{
		rl.ImageAlphaPremultiply(img)
	}

	HitRatingTexs[HitRatingBad] = rl.LoadTextureFromImage(ratingImgs[HitRatingBad])
	HitRatingTexs[HitRatingGood] = rl.LoadTextureFromImage(ratingImgs[HitRatingGood])
	HitRatingTexs[HitRatingSick] = rl.LoadTextureFromImage(ratingImgs[HitRatingSick])
}

