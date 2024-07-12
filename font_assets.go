package fnf

import (
	"bytes"
	"embed"
	"encoding/gob"
	rl "github.com/gen2brain/raylib-go/raylib"
	"io/fs"
)

//go:embed fonts-compiled
var EmebededFontAssets embed.FS

var (
	FontRegular    rl.Font
	SdfFontRegular SdfFont

	FontBold      rl.Font
	SdfFontBold   SdfFont
	KeySelectFont rl.Font

	FontClear    rl.Font
	SdfFontClear SdfFont
)

func GetFontFromName(fontName string) (
	success bool, font rl.Font, sdfFont SdfFont, isSdf bool,
) {
	switch fontName {
	case "FontRegular":
		return true, FontRegular, SdfFont{}, false
	case "SdfFontRegular":
		return true, rl.Font{}, SdfFontRegular, true

	case "FontBold":
		return true, FontBold, SdfFont{}, false
	case "SdfFontBold":
		return true, rl.Font{}, SdfFontBold, true

	case "KeySelectFont":
		return true, KeySelectFont, SdfFont{}, false

	case "FontClear":
		return true, FontClear, SdfFont{}, false
	case "SdfFontClear":
		return true, rl.Font{}, SdfFontClear, true

	default:
		return false, FontRegular, SdfFont{}, false
	}
}

var fontsToUnload []rl.Font

func LoadEmbededFonts() {
	loadFont := func(path, fontName string) rl.Font {
		data, err := fs.ReadFile(EmebededFontAssets, path)
		if err != nil {
			ErrorLogger.Fatalf("failed to load font \"%v\": %v", fontName, err)
		}
		font, err := DeserilaizeFont(data)
		if err != nil {
			ErrorLogger.Fatalf("failed to load font \"%v\": %v", fontName, err)
		}

		fontsToUnload = append(fontsToUnload, font)

		return font
	}

	loadSdfFont := func(path, fontName string) SdfFont {
		data, err := fs.ReadFile(EmebededFontAssets, path)
		if err != nil {
			ErrorLogger.Fatalf("failed to load sdf font \"%v\": %v", fontName, err)
		}
		font, err := DeserilaizeSdfFont(data)
		if err != nil {
			ErrorLogger.Fatalf("failed to load sdf font \"%v\": %v", fontName, err)
		}

		fontsToUnload = append(fontsToUnload, font.Font)

		return font
	}

	FontRegular = loadFont("fonts-compiled/UhBee-Se_hyun-128", "FontRegular")
	SdfFontRegular = loadSdfFont("fonts-compiled/UhBee-Se_hyun-SDF-64", "SdfFontRegular")

	FontBold = loadFont("fonts-compiled/UhBee-Se_hyun-Bold-128", "FontBold")
	SdfFontBold = loadSdfFont("fonts-compiled/UhBee-Se_hyun-SDF-64", "SdfFontBold")
	KeySelectFont = loadFont("fonts-compiled/UhBee-Se_hyun-Bold-240", "KeySelectFont")

	FontClear = loadFont("fonts-compiled/Pangolin-Regular-30", "FontClear")
	SdfFontClear = loadSdfFont("fonts-compiled/Pangolin-Regular-SDF-30", "SdfFontClear")
}

func UnloadEmbededFonts() {
	for _, font := range fontsToUnload {
		// NOTE : we actually shouldn't unload anything else because
		// everything besided textures are allocated by go
		rl.UnloadTexture(font.Texture)
	}
}

// only supports .otf or .ttf
// also it can't draw on image
func LoadFontAlphaPremultiply(fontData []byte, fontSize int32, codePoints []rune) rl.Font {
	var font rl.Font

	font.BaseSize = fontSize
	font.CharsPadding = 4 // default padding set by raylib

	glyphs := rl.LoadFontData(fontData, fontSize, codePoints, rl.FontDefault)
	atlasImg, recs := rl.GenImageFontAtlas(glyphs, fontSize, font.CharsPadding, 0)

	rl.ImageAlphaPremultiply(atlasImg)
	atlasTex := rl.LoadTextureFromImage(atlasImg)

	rl.SetFontCharGlyphs(&font, glyphs)
	rl.SetFontRecs(&font, recs)

	font.Texture = atlasTex

	rl.UnloadImage(atlasImg)

	return font
}

type FontContainer struct {
	BaseSize     int32
	CharsPadding int32
	Recs         []rl.Rectangle
	Chars        []rl.GlyphInfo
	ImageBytes   []byte
}

type SdfFontContainer struct {
	Font FontContainer

	// meta data
	SdfPadding        int32
	SdfOnEdgeValue    uint8
	SdfPixelDistScale float32
}

func SerializeFont(fontData []byte, fontSize int32) ([]byte, error) {
	serialized := FontContainer{
		BaseSize:     fontSize,
		CharsPadding: 4,
	}

	glyphs := rl.LoadFontData(fontData, fontSize, nil, rl.FontDefault)

	atlasImg, recs := rl.GenImageFontAtlas(glyphs, fontSize, serialized.CharsPadding, 0)

	rl.ImageAlphaPremultiply(atlasImg)

	serialized.Chars = glyphs
	serialized.Recs = recs

	serialized.ImageBytes = rl.ExportImageToMemory(*atlasImg, ".png")

	bufWriter := new(bytes.Buffer)

	encoder := gob.NewEncoder(bufWriter)
	if err := encoder.Encode(serialized); err != nil {
		return nil, err
	}

	return bufWriter.Bytes(), nil
}

func DeserilaizeFont(serializedFontData []byte) (rl.Font, error) {
	var serialized FontContainer

	reader := bytes.NewBuffer(serializedFontData)

	decoder := gob.NewDecoder(reader)

	if err := decoder.Decode(&serialized); err != nil {
		return rl.Font{}, err
	}

	atlasImg := rl.LoadImageFromMemory(".png", serialized.ImageBytes, i32(len(serialized.ImageBytes)))

	font := rl.Font{
		BaseSize:     serialized.BaseSize,
		CharsCount:   i32(min(len(serialized.Chars), len(serialized.Recs))),
		CharsPadding: serialized.CharsPadding,
	}

	rl.SetFontCharGlyphs(&font, serialized.Chars)
	rl.SetFontRecs(&font, serialized.Recs)

	font.Texture = rl.LoadTextureFromImage(atlasImg)

	rl.GenTextureMipmaps(&font.Texture)
	rl.SetTextureFilter(font.Texture, rl.FilterTrilinear)

	rl.UnloadImage(atlasImg)

	return font, nil
}

func SerializeSdfFont(
	fontData []byte,
	fontSize int32,
	sdfPadding int32,
	sdfOnEdgeValue uint8,
	sdfPixelDistScale float32,
) ([]byte, error) {
	serialized := SdfFontContainer{
		SdfPadding:        sdfPadding,
		SdfOnEdgeValue:    sdfOnEdgeValue,
		SdfPixelDistScale: sdfPixelDistScale,
		Font: FontContainer{
			BaseSize: fontSize,
		},
	}

	glyphs := rl.LoadFontDataSdf(
		fontData, fontSize, nil, sdfPadding, sdfOnEdgeValue, sdfPixelDistScale)

	atlasImg, rects := rl.GenImageFontAtlas(glyphs, fontSize, 0, 1)

	serialized.Font.Recs = rects
	serialized.Font.Chars = glyphs

	serialized.Font.ImageBytes = rl.ExportImageToMemory(*atlasImg, ".png")

	bufWriter := new(bytes.Buffer)

	encoder := gob.NewEncoder(bufWriter)
	if err := encoder.Encode(serialized); err != nil {
		return nil, err
	}

	return bufWriter.Bytes(), nil
}

func DeserilaizeSdfFont(serializedFontData []byte) (SdfFont, error) {
	var serialized SdfFontContainer

	reader := bytes.NewBuffer(serializedFontData)

	decoder := gob.NewDecoder(reader)

	if err := decoder.Decode(&serialized); err != nil {
		return SdfFont{}, err
	}

	atlasImg := rl.LoadImageFromMemory(".png", serialized.Font.ImageBytes, i32(len(serialized.Font.ImageBytes)))

	font := SdfFont{
		Font: rl.Font{
			BaseSize:     serialized.Font.BaseSize,
			CharsCount:   i32(min(len(serialized.Font.Chars), len(serialized.Font.Recs))),
			CharsPadding: serialized.Font.CharsPadding,
		},
		SdfPadding:        serialized.SdfPadding,
		SdfOnEdgeValue:    serialized.SdfOnEdgeValue,
		SdfPixelDistScale: serialized.SdfPixelDistScale,
	}

	rl.SetFontCharGlyphs(&font.Font, serialized.Font.Chars)
	rl.SetFontRecs(&font.Font, serialized.Font.Recs)

	font.Font.Texture = rl.LoadTextureFromImage(atlasImg)

	rl.SetTextureFilter(font.Font.Texture, rl.FilterBilinear)

	rl.UnloadImage(atlasImg)

	return font, nil
}
