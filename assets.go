package fnf

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var EmebededAssets embed.FS

var (
	ArrowsSprite     Sprite
	ArrowsGlowSprite Sprite
)

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
	GameScreenBg       rl.Texture2D
	MenuScreenBg       rl.Texture2D
	MenuScreenSimpleBg rl.Texture2D
)

var DancingNoteSprite Sprite

var (
	SongLoadingScreen rl.Texture2D
	DirSelectScreen   rl.Texture2D
)

var HitRatingTexs [HitRatingSize]rl.Texture2D

var BookMarkBigTex rl.Texture2D
var BookMarkSmallTex rl.Texture2D

var PopupBg rl.Texture2D

var BlackPixel rl.Texture2D
var WhitePixel rl.Texture2D

var HitSoundAudio []byte

var (
	texsToUnload []rl.Texture2D
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

		tex := rl.LoadTextureFromImage(img)
		rl.GenTextureMipmaps(&tex)
		rl.SetTextureFilter(tex, rl.FilterTrilinear)

		if tex.ID == 0 {
			ErrorLogger.Fatalf("failed to load texture from img : %v", path)
		}

		texsToUnload = append(texsToUnload, tex)

		rl.UnloadImage(img)

		return tex
	}

	// =============================
	// load reloadable assets
	// =============================

	// load fnf arrows texture
	{
		var err error
		ArrowsSprite, err = ParseSpriteJsonMetadata(bytes.NewBuffer(loadData("assets/arrows.json")))

		if err != nil {
			ErrorLogger.Fatal(err)
		}

		ArrowsSprite.Texture = loadTexture("assets/arrows.png", true, ".png")
	}

	// load fnf arrows glow texture
	{
		glowTex := loadTexture("assets/arrows-glow.png", true, ".png")

		ArrowsGlowSprite.Texture = glowTex

		ArrowsGlowSprite.Count = int(NoteDirSize)

		ArrowsGlowSprite.Height = f32(glowTex.Height)

		// NOTE : same goes for glow arrows
		ArrowsGlowSprite.Width = f32(glowTex.Width) / f32(NoteDirSize)
	}

	// load ui arrows texture
	{
		uiArrowsTex := loadTexture("assets/ui-arrows.png", true, ".png")

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
	// load dancing sprite
	{
		jsonBytes := loadData("assets/dancing-note.json")
		buffer := bytes.NewBuffer(jsonBytes)
		var err error
		DancingNoteSprite, err = ParseSpriteJsonMetadata(buffer)
		if err != nil {
			ErrorLogger.Fatal(err)
		}
		DancingNoteSprite.Texture = loadTexture("assets/dancing-note.png", true, ".png")
	}

	BookMarkBigTex = loadTexture("assets/bookmark-big.png", true, ".png")
	BookMarkSmallTex = loadTexture("assets/bookmark-small.png", true, ".png")

	GameScreenBg = loadTexture("assets/game-background.png", true, ".png")
	MenuScreenBg = loadTexture("assets/menu-background.png", true, ".png")
	MenuScreenSimpleBg = loadTexture("assets/menu-background-simple.png", true, ".png")
	SongLoadingScreen = loadTexture("assets/song-loading-screen.png", true, ".png")
	DirSelectScreen = loadTexture("assets/directory-select-screen.png", true, ".png")

	ratingImgPaths := [HitRatingSize]string{
		"assets/bad.png",
		"assets/good.png",
		"assets/sick.png",
	}

	for r := FnfHitRating(0); r < HitRatingSize; r++ {
		HitRatingTexs[r] = loadTexture(ratingImgPaths[r], true, ".png")
	}

	PopupBg = loadTexture("assets/popup-bg.png", true, ".png")

	// create black pixel
	blackPixelImg := rl.GenImageColor(2, 2, ToRlColor(FnfColor{0, 0, 0, 255}))
	BlackPixel = rl.LoadTextureFromImage(blackPixelImg)
	texsToUnload = append(texsToUnload, BlackPixel)

	// create white pixel
	whitePixelImg := rl.GenImageColor(2, 2, ToRlColor(FnfColor{255, 255, 255, 255}))
	WhitePixel = rl.LoadTextureFromImage(whitePixelImg)
	texsToUnload = append(texsToUnload, WhitePixel)

	// =============================
	// load unreloadable assets
	// =============================
	if !isReload {
		// load hit sound
		{
			const hitSoundPath = "assets/hit-sound.ogg"

			audioFile := loadData(hitSoundPath)

			var err error

			var decoder AudioDecoder

			if decoder, err = NewAudioDeocoder(audioFile, filepath.Ext(hitSoundPath)); err != nil {
				ErrorLogger.Fatalf("failed to create decoder %v", err)
			}

			if HitSoundAudio, err = io.ReadAll(decoder); err != nil {
				ErrorLogger.Fatalf("failed to decode audio %v", err)
			}
		}
	}
}

func unloadAssets(isReload bool) {
	for _, tex := range texsToUnload {
		if tex.ID > 0 {
			rl.UnloadTexture(tex)
		}
	}

	texsToUnload = texsToUnload[:0]
}
