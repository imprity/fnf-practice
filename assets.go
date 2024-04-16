package main

import (
	"embed"
	"io/fs"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var EmebededAssets embed.FS

var ArrowsOuterTex rl.Texture2D
var ArrowsInnerTex rl.Texture2D

var ArrowsRects [NoteDirSize]rl.Rectangle

var ArrowsGlowTex rl.Texture2D
var ArrowsGlowRects [NoteDirSize]rl.Rectangle

var GameScreenBg rl.Texture2D

var SongLoadingScreen rl.Texture2D

var DirSelectScreen rl.Texture2D

var HitRatingTexs [HitRatingSize]rl.Texture2D

var FontRegular rl.Font
var FontBold rl.Font

var HelpMsgFont rl.Font

var imgsToUnload []*rl.Image
var texsToUnload []rl.Texture2D
var fontsToUnload []rl.Font

var isAssetLoaded bool

func LoadAssets() {
	if isAssetLoaded {
		for _, img := range imgsToUnload {
			if rl.IsImageReady(img) {
				rl.UnloadImage(img)
			}
		}

		imgsToUnload = imgsToUnload[:0]

		for _, tex := range texsToUnload {
			if tex.ID > 0 {
				rl.UnloadTexture(tex)
			}
		}

		texsToUnload = texsToUnload[:0]

		for _, font := range fontsToUnload {
			if rl.IsFontReady(font) {
				rl.UnloadFont(font)
			}
		}

		fontsToUnload = fontsToUnload[:0]

		isAssetLoaded = false
	}

	defer func() {
		isAssetLoaded = true
	}()

	loadTexture := func(path string, premultiply bool, fileType string) rl.Texture2D {
		var byteArray []byte
		var err error

		if *FlagHotReloading {
			byteArray, err = os.ReadFile(path)
		} else {
			byteArray, err = fs.ReadFile(EmebededAssets, path)
		}

		if err != nil {
			ErrorLogger.Fatal(err)
		}

		img := rl.LoadImageFromMemory(fileType, byteArray, int32(len(byteArray)))

		if !rl.IsImageReady(img) {
			ErrorLogger.Fatalf("failed to load img : %v", path)
		}

		if premultiply {
			rl.ImageAlphaPremultiply(img)
		}

		imgsToUnload = append(imgsToUnload, img)

		tex := rl.LoadTextureFromImage(img)

		if tex.ID == 0 {
			ErrorLogger.Fatalf("failed to load texture from img : %v", path)
		}

		texsToUnload = append(texsToUnload, tex)

		return tex
	}

	loadFont := func(path string, fontSize int32, fileType string) rl.Font {
		var byteArray []byte
		var err error

		if *FlagHotReloading {
			byteArray, err = os.ReadFile(path)
		} else {
			byteArray, err = fs.ReadFile(EmebededAssets, path)
		}

		if err != nil {
			ErrorLogger.Fatal(err)
		}

		// NOTE : for code points we are supplying empty code points
		// this will default to loading only ascii characters
		// I thougt about adding all the corean code points but I think that would be too expensive

		// TODO : SUPPORT UNICODE (somehow)
		var emptyCodePoints []rune
		font := rl.LoadFontFromMemory(fileType, byteArray, fontSize, emptyCodePoints)

		if !rl.IsFontReady(font) {
			ErrorLogger.Fatalf("failed to load font : %v", path)
		}

		fontsToUnload = append(fontsToUnload, font)

		return font
	}

	ArrowsOuterTex = loadTexture("assets/arrows_outer.png", true, ".png")
	ArrowsInnerTex = loadTexture("assets/arrows_inner.png", true, ".png")

	if ArrowsOuterTex.Width != ArrowsInnerTex.Width || ArrowsOuterTex.Height != ArrowsInnerTex.Height {
		ErrorLogger.Fatal("Arrow inner and outer images should have same size")
	}

	// NOTE : we will assume that we can get arrow rects
	// by just devding the width by 4

	width := float32(ArrowsOuterTex.Width) / 4.0

	for i := NoteDir(0); i < NoteDirSize; i++ {
		x := float32(i) * width
		ArrowsRects[i] = rl.Rectangle{
			x, 0, width, float32(ArrowsOuterTex.Height),
		}
	}

	ArrowsGlowTex = loadTexture("assets/arrows_glow.png", true, ".png")

	// NOTE : same goes for glow arrows

	width = float32(ArrowsGlowTex.Width) / 4.0

	for i := NoteDir(0); i < NoteDirSize; i++ {
		x := float32(i) * width
		ArrowsGlowRects[i] = rl.Rectangle{
			x, 0, width, float32(ArrowsGlowTex.Height),
		}
	}

	GameScreenBg = loadTexture("assets/background 1.png", true, ".png")
	SongLoadingScreen = loadTexture("assets/song loading screen.png", true, ".png")
	DirSelectScreen = loadTexture("assets/directory select screen.png", true, ".png")

	ratingImgPaths := [HitRatingSize]string{
		"assets/bad.png",
		"assets/good.png",
		"assets/sick.png",
	}

	for r := FnfHitRating(0); r < HitRatingSize; r++ {
		HitRatingTexs[r] = loadTexture(ratingImgPaths[r], true, ".png")
	}

	regularPath := "assets/UhBeeSe_hyun/UhBee Se_hyun.ttf"
	boldPath := "assets/UhBeeSe_hyun/UhBee Se_hyun Bold.ttf"

	FontRegular = loadFont(regularPath, 128, ".ttf")
	FontBold = loadFont(boldPath, 128, ".ttf")

	helpFontPath := "assets/Pangolin/Pangolin-Regular.ttf"

	HelpMsgFont = loadFont(helpFontPath, 30, ".ttf")
}
