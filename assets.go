package main

import (
	"bytes"
	"embed"
	"io/fs"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var EmebededAssets embed.FS

var (
	ArrowsFillSprite Sprite
	ArrowsStrokeSprite Sprite
)

var ArrowsGlowSprite Sprite

var UIarrowsSprite Sprite

const (
	UIarrowLeftStroke = iota
	UIarrowRightStroke
	UIarrowLeftFill
	UIarrowRightFill

	UIarrowsSpriteCount
)

var SustainTex rl.Texture2D

var (
	CheckBoxMark Sprite
	CheckBoxBox  rl.Texture2D
)

var (
	GameScreenBg rl.Texture2D
	MenuScreenBg rl.Texture2D
)

var (
	SongLoadingScreen rl.Texture2D
	DirSelectScreen   rl.Texture2D
)

var HitRatingTexs [HitRatingSize]rl.Texture2D

var BookMarkBigTex rl.Texture2D
var BookMarkSmallTex rl.Texture2D

var (
	//go:embed "fonts/UhBeeSe_hyun/UhBee Se_hyun.ttf"
	fontRegularData []byte
	FontRegular     rl.Font

	//go:embed "fonts/UhBeeSe_hyun/UhBee Se_hyun Bold.ttf"
	fontBoldData  []byte
	FontBold      rl.Font
	KeySelectFont rl.Font

	//go:embed fonts/Pangolin/Pangolin-Regular.ttf
	FontClearData []byte
	FontClear     rl.Font
)

var PopupBg rl.Texture2D

var BlackPixel rl.Texture2D

var (
	imgsToUnload  []*rl.Image
	texsToUnload  []rl.Texture2D
	fontsToUnload []rl.Font
)

func LoadAssets() {
	loadAssets(false)
}

func ReloadAssets() {
	loadAssets(true)
}

func UnloadAssets() {
	unloadAssets(false)
}

func loadAssets(isReload bool) {
	unloadAssets(isReload)

	loadData := func(path string) []byte {
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

		return byteArray
	}

	loadTexture := func(path string, premultiply bool, fileType string) rl.Texture2D {
		byteArray := loadData(path)

		img := rl.LoadImageFromMemory(fileType, byteArray, int32(len(byteArray)))

		if !rl.IsImageReady(img) {
			ErrorLogger.Fatalf("failed to load img : %v", path)
		}

		if premultiply {
			rl.ImageAlphaPremultiply(img)
		}

		imgsToUnload = append(imgsToUnload, img)

		tex := rl.LoadTextureFromImage(img)
		rl.GenTextureMipmaps(&tex)
		rl.SetTextureFilter(tex, rl.FilterTrilinear)

		if tex.ID == 0 {
			ErrorLogger.Fatalf("failed to load texture from img : %v", path)
		}

		texsToUnload = append(texsToUnload, tex)

		return tex
	}

	loadFont := func(fontData []byte, fontName string, fontSize int32, fileType string) rl.Font {
		// NOTE : for code points we are supplying empty code points
		// this will default to loading only ascii characters
		// I thougt about adding all the Korean code points but I think that would be too expensive
		var emptyCodePoints []rune
		font := rl.LoadFontFromMemory(fileType, fontData, fontSize, emptyCodePoints)

		if !rl.IsFontReady(font) {
			ErrorLogger.Fatalf("failed to load font : %v", fontName)
		}

		rl.GenTextureMipmaps(&font.Texture)
		rl.SetTextureFilter(font.Texture, rl.FilterTrilinear)

		fontsToUnload = append(fontsToUnload, font)

		return font
	}

	// load fnf arrows texture
	{
		strokeTex := loadTexture("assets/arrows_outer.png", true, ".png")
		fillTex := loadTexture("assets/arrows_inner.png", true, ".png")

		if strokeTex.Width != fillTex.Width || strokeTex.Height != fillTex.Height {
			ErrorLogger.Fatal("Arrow fill and stroke images should have same size")
		}

		ArrowsStrokeSprite.Texture = strokeTex
		ArrowsFillSprite.Texture = fillTex

		ArrowsStrokeSprite.Count = int(NoteDirSize)
		ArrowsFillSprite.Count = int(NoteDirSize)

		ArrowsStrokeSprite.Height = f32(strokeTex.Height)
		ArrowsFillSprite.Height = f32(fillTex.Height)

		// NOTE : we will assume that we can get arrow width
		// by just devding the texture width by 4
		ArrowsStrokeSprite.Width = f32(strokeTex.Width) / f32(NoteDirSize)
		ArrowsFillSprite.Width = f32(fillTex.Width) / f32(NoteDirSize)
	}

	// load fnf arrows glow texture
	{
		glowTex := loadTexture("assets/arrows_glow.png", true, ".png")

		ArrowsGlowSprite.Texture = glowTex

		ArrowsGlowSprite.Count = int(NoteDirSize)

		ArrowsGlowSprite.Height = f32(glowTex.Height)

		// NOTE : same goes for glow arrows
		ArrowsGlowSprite.Width = f32(glowTex.Width) / f32(NoteDirSize)
	}

	// load ui arrows texture
	{
		uiArrowsTex := loadTexture("assets/ui_arrows.png", true, ".png")

		UIarrowsSprite.Texture = uiArrowsTex

		UIarrowsSprite.Count = UIarrowsSpriteCount

		UIarrowsSprite.Height = f32(uiArrowsTex.Height)

		// NOTE : same also goes for ui arrows
		UIarrowsSprite.Width = f32(uiArrowsTex.Width) / UIarrowsSpriteCount
	}

	SustainTex = loadTexture("assets/sustain-bar.png", true, ".png")
	if SustainTex.Width > SustainTex.Height {
		ErrorLogger.Printf("SustainTex width(%v) is bigger than height(%v)",
			SustainTex.Width, SustainTex.Height)
	}

	// load checkbox sprite
	{
		CheckBoxBox = loadTexture("assets/checkbox-box.png", true, ".png")

		jsonBytes := loadData("assets/checkbox-sprites.json")
		buffer := bytes.NewBuffer(jsonBytes)

		var err error

		CheckBoxMark, err = ParseSpriteJsonMetadata(buffer)

		if err != nil {
			ErrorLogger.Fatal(err)
		}

		CheckBoxMark.Texture = loadTexture("assets/checkbox-sprites.png", true, ".png")
	}

	BookMarkBigTex = loadTexture("assets/bookmark_big.png", true, ".png")
	BookMarkSmallTex = loadTexture("assets/bookmark_small.png", true, ".png")

	GameScreenBg = loadTexture("assets/background 1.png", true, ".png")
	MenuScreenBg = loadTexture("assets/menu_background.png", true, ".png")
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

	PopupBg = loadTexture("assets/popup_bg.png", true, ".png")

	// create black pixel
	blackPixelImg := rl.GenImageColor(2, 2, rl.Color{0, 0, 0, 255})
	imgsToUnload = append(imgsToUnload, blackPixelImg)

	BlackPixel = rl.LoadTextureFromImage(blackPixelImg)
	texsToUnload = append(texsToUnload, BlackPixel)

	if !isReload {
		FontRegular = loadFont(fontRegularData, "FontRegular", 128, ".ttf")
		FontBold = loadFont(fontBoldData, "FontBold", 128, ".ttf")
		KeySelectFont = loadFont(fontBoldData, "KeySelectFont", 240, ".ttf")

		FontClear = loadFont(FontClearData, "HelpMsgFont", 30, ".ttf")
	}
}

func unloadAssets(isReload bool) {
	for _, tex := range texsToUnload {
		if tex.ID > 0 {
			rl.UnloadTexture(tex)
		}
	}

	texsToUnload = texsToUnload[:0]

	for _, img := range imgsToUnload {
		if rl.IsImageReady(img) {
			rl.UnloadImage(img)
		}
	}

	imgsToUnload = imgsToUnload[:0]

	if !isReload {
		for _, font := range fontsToUnload {
			if rl.IsFontReady(font) {
				rl.UnloadFont(font)
			}
		}
		fontsToUnload = fontsToUnload[:0]
	}
}
