package main

import (
	"embed"
	"os"
	"io/fs"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var EmebededAssets embed.FS

var ArrowsOuterTex rl.Texture2D
var ArrowsInnerTex rl.Texture2D

var ArrowsRects [NoteDirSize]rl.Rectangle

var ArrowsGlowTex rl.Texture2D
var ArrowsGlowRects [NoteDirSize]rl.Rectangle

var PrettyBackground rl.Texture2D

var HitRatingTexs [HitRatingSize]rl.Texture2D

var imgsToUnload []*rl.Image
var texsToUnload []rl.Texture2D

var isAssetLoaded bool

func LoadAssets() {
	if isAssetLoaded{
		for _, img := range imgsToUnload{
			if rl.IsImageReady(img){
				rl.UnloadImage(img)
			}
		}

		imgsToUnload = imgsToUnload[:0]

		for _, tex := range texsToUnload{
			if tex.ID > 0 {
				rl.UnloadTexture(tex)
			}
		}

		texsToUnload = texsToUnload[:0]
		isAssetLoaded = false
	}

	defer func(){
		isAssetLoaded = true
	}()

	loadTexture := func(path string, premultiply bool, fileType string) rl.Texture2D{
		var byteArray []byte
		var err error

		if *FlagHotReloading{
			byteArray, err = os.ReadFile(path)
		}else{
			byteArray, err = fs.ReadFile(EmebededAssets, path)
		}

		if err != nil {ErrorLogger.Fatal(err)}

		img := rl.LoadImageFromMemory(fileType, byteArray, int32(len(byteArray)))

		if !rl.IsImageReady(img){
			ErrorLogger.Fatalf("failed to load img : %v", path)
		}

		if premultiply{
			rl.ImageAlphaPremultiply(img)
		}

		imgsToUnload = append(imgsToUnload, img)

		tex := rl.LoadTextureFromImage(img)

		if tex.ID == 0{
			ErrorLogger.Fatalf("failed to load texture from img : %v", path)
		}

		texsToUnload = append(texsToUnload, tex)

		return tex
	}

	ArrowsOuterTex = loadTexture("assets/arrows_outer.png", true, ".png")
	ArrowsInnerTex = loadTexture("assets/arrows_inner.png", true, ".png") 

	if ArrowsOuterTex.Width != ArrowsInnerTex.Width || ArrowsOuterTex.Height != ArrowsInnerTex.Height{
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

	ArrowsGlowTex = loadTexture("assets/arrows_glow.png", true, ".png")

	// NOTE : same goes for glow arrows

	width = float32(ArrowsGlowTex.Width) / 4.0

	for i:=NoteDir(0); i<NoteDirSize; i++{
		x := float32(i) * width
		ArrowsGlowRects[i] = rl.Rectangle{
			x, 0, width, float32(ArrowsGlowTex.Height),
		}
	}

	PrettyBackground = loadTexture("assets/background 1.png", true, ".png") 

	ratingImgPaths := [HitRatingSize]string{
		"assets/bad.png",
		"assets/good.png",
		"assets/sick.png",
	}

	for r:= FnfHitRating(0); r<HitRatingSize; r++{
		HitRatingTexs[r]  = loadTexture(ratingImgPaths[r], true, ".png")
	}
}

