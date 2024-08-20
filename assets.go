package fnf

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

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

var (
	SongLoadingScreen rl.Texture2D
	DirSelectScreen   rl.Texture2D
)

var (
	SplashFillSprite   [2]Sprite
	SplashStrokeSprite [2]Sprite
)

var HitRatingTexs [HitRatingSize]rl.Texture2D

var (
	BookMarkBigTex   rl.Texture2D
	BookMarkSmallTex rl.Texture2D
)

var PopupBg rl.Texture2D

var (
	BlackPixel rl.Texture2D
	WhitePixel rl.Texture2D
)

var HitSoundAudio []byte

var DancingNoteSprite Sprite

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

	loadData := func(pathStr string) []byte {
		var byteArray []byte
		var err error

		if *FlagHotReloading {
			byteArray, err = os.ReadFile(filepath.Clean(pathStr))
		} else {
			byteArray, err = fs.ReadFile(EmebededAssets, path.Clean(pathStr))
		}

		if err != nil {
			ErrorLogger.Fatal(err)
		}

		return byteArray
	}

	loadTexture := func(path string, premultiply bool) rl.Texture2D {
		byteArray := loadData(path)

		img := rl.LoadImageFromMemory(filepath.Ext(path), byteArray, int32(len(byteArray)))

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

	loadSprite := func(jsonPath string, imgPath string, premultiply bool) Sprite {
		jsonBytes := loadData(jsonPath)
		buffer := bytes.NewBuffer(jsonBytes)

		var err error
		var sprite Sprite
		if sprite, err = ParseSpriteJsonMetadata(buffer); err != nil {
			ErrorLogger.Fatal(err)
		}

		sprite.Texture = loadTexture(imgPath, premultiply)

		return sprite
	}

	// =============================
	// load reloadable assets
	// =============================

	// load fnf arrows texture
	ArrowsSprite = loadSprite("assets/arrows.json", "assets/arrows.png", true)

	// load fnf arrows glow texture
	{
		glowTex := loadTexture("assets/arrows-glow.png", true)

		ArrowsGlowSprite.Texture = glowTex

		ArrowsGlowSprite.Count = int(NoteDirSize)

		ArrowsGlowSprite.Height = f32(glowTex.Height)

		// NOTE : same goes for glow arrows
		ArrowsGlowSprite.Width = f32(glowTex.Width) / f32(NoteDirSize)
	}

	// load ui arrows texture
	{
		uiArrowsTex := loadTexture("assets/ui-arrows.png", true)

		UIarrowsSprite.Texture = uiArrowsTex

		UIarrowsSprite.Count = UIarrowsSpriteCount

		UIarrowsSprite.Height = f32(uiArrowsTex.Height)

		// NOTE : same also goes for ui arrows
		UIarrowsSprite.Width = f32(uiArrowsTex.Width) / UIarrowsSpriteCount
	}

	SustainTex = loadTexture("assets/sustain-bar.png", true)
	if SustainTex.Width > SustainTex.Height {
		ErrorLogger.Printf("SustainTex width(%v) is bigger than height(%v)",
			SustainTex.Width, SustainTex.Height)
	}

	// load checkbox sprite
	{
		CheckBoxBox = loadTexture("assets/checkbox-box.png", true)
		CheckBoxMark = loadSprite("assets/checkbox-sprites.json", "assets/checkbox-sprites.png", true)
	}

	// load splash fill sprite
	for i := range 2 {
		SplashFillSprite[i] = loadSprite(
			fmt.Sprintf("assets/splash-fill%d.json", i+1),
			fmt.Sprintf("assets/splash-fill%d.png", i+1), true,
		)
	}

	// load splash stroke sprite
	for i := range 2 {
		SplashStrokeSprite[i] = loadSprite(
			fmt.Sprintf("assets/splash-stroke%d.json", i+1),
			fmt.Sprintf("assets/splash-stroke%d.png", i+1), true,
		)
	}

	// check if splash fill and stroke have the same size and sprite count
	for a := range 2 {
		for b := range 2 {
			if SplashFillSprite[a].Count != SplashStrokeSprite[b].Count {
				ErrorLogger.Fatal(
					fmt.Errorf("SplashFillSprite%d and SplashStrokeSprite%d have different sprite count", a, b))
			}
			if SplashFillSprite[a].Width != SplashStrokeSprite[b].Width {
				ErrorLogger.Fatal(
					fmt.Errorf("SplashFillSprite%d and SplashStrokeSprite%d have different width", a, b))
			}
			if SplashFillSprite[a].Height != SplashStrokeSprite[b].Height {
				ErrorLogger.Fatal(
					fmt.Errorf("SplashFillSprite%d and SplashStrokeSprite%d have different height", a, b))
			}
		}
	}

	BookMarkBigTex = loadTexture("assets/bookmark-big.png", true)
	BookMarkSmallTex = loadTexture("assets/bookmark-small.png", true)

	GameScreenBg = loadTexture("assets/game-background.png", true)
	MenuScreenBg = loadTexture("assets/menu-background.png", true)
	MenuScreenSimpleBg = loadTexture("assets/menu-background-simple.png", true)
	SongLoadingScreen = loadTexture("assets/song-loading-screen.png", true)
	DirSelectScreen = loadTexture("assets/directory-select-screen.png", true)

	ratingImgPaths := [HitRatingSize]string{
		"assets/bad.png",
		"assets/good.png",
		"assets/sick.png",
	}

	for r := FnfHitRating(0); r < HitRatingSize; r++ {
		HitRatingTexs[r] = loadTexture(ratingImgPaths[r], true)
	}

	PopupBg = loadTexture("assets/popup-bg.png", true)

	// load dancing notes
	DancingNoteSprite = loadSprite("assets/dancing-note.json", "assets/dancing-note.png", true)

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

		loadCustomHitSound := func() bool {
			var err error

			var dirPath string
			if dirPath, err = RelativePath("./"); err != nil {
				ErrorLogger.Printf("failed to load custom hitsound %v", err)
				return false
			}

			var dirEntries []os.DirEntry
			if dirEntries, err = os.ReadDir(dirPath); err != nil {
				ErrorLogger.Printf("failed to load custom hitsound %v", err)
				return false
			}

			var foundHitSoundCandidate bool = false

			for _, entry := range dirEntries {
				if mode := entry.Type(); !(mode.IsRegular() && !mode.IsDir()) {
					continue
				}

				nameLow := strings.ToLower(entry.Name())

				if nameLow == "hit-sound.ogg" ||
					nameLow == "hit-sound.mp3" ||
					nameLow == "hit-sound.wav" {

					foundHitSoundCandidate = true

					var audioErr error

					var audioFile []byte
					if audioFile, audioErr = os.ReadFile(filepath.Join(dirPath, entry.Name())); audioErr != nil {
						ErrorLogger.Printf("failed to decode custom hitsound %v: %v", entry.Name(), audioErr)
						continue
					}

					var decoder AudioDecoder
					if decoder, audioErr = NewAudioDeocoder(audioFile, filepath.Ext(nameLow)); audioErr != nil {
						ErrorLogger.Printf("failed to decode custom hitsound %v: %v", entry.Name(), audioErr)
						continue
					}

					var audio []byte
					if audio, audioErr = io.ReadAll(decoder); audioErr != nil {
						ErrorLogger.Printf("failed to decode custom hitsound %v: %v", entry.Name(), audioErr)
						continue
					}

					HitSoundAudio = audio

					return true
				}
			}

			if foundHitSoundCandidate {
				DisplayAlert("failed to load custom hitsound")
			}

			return false
		}

		if !loadCustomHitSound() {
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
