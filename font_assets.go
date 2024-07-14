package fnf

import (
	"bytes"
	"embed"
	"encoding/gob"
	"fmt"
	"io/fs"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed fonts-compiled
var EmebededFontAssets embed.FS

var (
	FontRegular    FnfFont
	SdfFontRegular FnfFont

	FontBold    FnfFont
	SdfFontBold FnfFont

	FontClear    FnfFont
	SdfFontClear FnfFont
)

var NameToFont map[string]FnfFont = map[string]FnfFont{
	"FontRegular":    FontRegular,
	"SdfFontRegular": SdfFontRegular,

	"FontBold":    FontBold,
	"SdfFontBold": SdfFontBold,

	"FontClear":    FontClear,
	"SdfFontClear": SdfFontClear,
}

var (
	//go:embed "fonts/UhBeeSe_hyun/UhBee Se_hyun.ttf"
	backupFontData   []byte
	backupFont       FnfFont
	backupFontLoaded bool
)

func GetBackupFont() FnfFont {
	if !backupFontLoaded {
		backupFontLoaded = true
		backupFont = LoadSdfFontFromMemory(
			backupFontData, 128, nil,
			21, 200, 10,
		)

		fontsToUnload = append(fontsToUnload, backupFont.Font)
	}

	return backupFont
}

func GetFontFromName(fontName string) (font FnfFont, ok bool) {
	font, ok = NameToFont[fontName]
	if !ok {
		font = GetBackupFont()
	}
	return
}

var fontsToUnload []rl.Font

func LoadEmbededFonts() {
	loadFont := func(path, fontName string) FnfFont {
		data, err := fs.ReadFile(EmebededFontAssets, path)
		if err != nil {
			ErrorLogger.Printf("failed to load font \"%v\": %v", fontName, err)
			return GetBackupFont()
		}
		font, err := DeserilaizeFont(data)
		if err != nil {
			ErrorLogger.Printf("failed to load font \"%v\": %v", fontName, err)
			return GetBackupFont()
		}

		fontsToUnload = append(fontsToUnload, font.Font)

		return font
	}

	FontRegular = loadFont("fonts-compiled/UhBee-Se_hyun-128", "FontRegular")
	SdfFontRegular = loadFont("fonts-compiled/UhBee-Se_hyun-SDF-64", "SdfFontRegular")

	FontBold = loadFont("fonts-compiled/UhBee-Se_hyun-Bold-128", "FontBold")
	SdfFontBold = loadFont("fonts-compiled/UhBee-Se_hyun-SDF-64", "SdfFontBold")

	FontClear = loadFont("fonts-compiled/Pangolin-Regular-30", "FontClear")
	SdfFontClear = loadFont("fonts-compiled/Pangolin-Regular-SDF-30", "SdfFontClear")
}

func UnloadEmbededFonts() {
	for _, font := range fontsToUnload {
		// NOTE : we actually shouldn't unload anything else because
		// everything besided textures are allocated by go
		rl.UnloadTexture(font.Texture)
	}
}

type FontContainer struct {
	BaseSize     int32
	CharsPadding int32
	Recs         []rl.Rectangle
	Chars        []rl.GlyphInfo
	ImageBytes   []byte

	IsSdfFont bool

	// meta data
	SdfPadding        int32
	SdfOnEdgeValue    uint8
	SdfPixelDistScale float32
}

func SerializeFont(fontData []byte, fontSize int32) ([]byte, error) {
	serialized := FontContainer{
		BaseSize:     fontSize,
		CharsPadding: 4,
		IsSdfFont:    false,
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

func SerializeSdfFont(
	fontData []byte,
	fontSize int32,
	sdfPadding int32,
	sdfOnEdgeValue uint8,
	sdfPixelDistScale float32,
) ([]byte, error) {
	serialized := FontContainer{
		BaseSize: fontSize,

		IsSdfFont: true,

		SdfPadding:        sdfPadding,
		SdfOnEdgeValue:    sdfOnEdgeValue,
		SdfPixelDistScale: sdfPixelDistScale,
	}

	glyphs := rl.LoadFontDataSdf(
		fontData, fontSize, nil, sdfPadding, sdfOnEdgeValue, sdfPixelDistScale)

	atlasImg, rects := rl.GenImageFontAtlas(glyphs, fontSize, 0, 1)

	serialized.Recs = rects
	serialized.Chars = glyphs

	serialized.ImageBytes = rl.ExportImageToMemory(*atlasImg, ".png")

	bufWriter := new(bytes.Buffer)

	encoder := gob.NewEncoder(bufWriter)
	if err := encoder.Encode(serialized); err != nil {
		return nil, err
	}

	return bufWriter.Bytes(), nil
}

func DeserilaizeFont(serializedFontData []byte) (FnfFont, error) {
	var serialized FontContainer

	reader := bytes.NewBuffer(serializedFontData)

	decoder := gob.NewDecoder(reader)

	if err := decoder.Decode(&serialized); err != nil {
		return FnfFont{}, err
	}

	if len(serialized.Chars) <= 0 || len(serialized.Recs) <= 0 || len(serialized.ImageBytes) <= 0 {
		return FnfFont{}, fmt.Errorf("data needed for font creation is empty")
	}

	atlasImg := rl.LoadImageFromMemory(".png", serialized.ImageBytes, i32(len(serialized.ImageBytes)))

	font := FnfFont{
		Font: rl.Font{
			BaseSize:     serialized.BaseSize,
			CharsCount:   i32(min(len(serialized.Chars), len(serialized.Recs))),
			CharsPadding: serialized.CharsPadding,
		},

		IsSdfFont: serialized.IsSdfFont,

		SdfPadding:        serialized.SdfPadding,
		SdfOnEdgeValue:    serialized.SdfOnEdgeValue,
		SdfPixelDistScale: serialized.SdfPixelDistScale,
	}

	rl.SetFontCharGlyphs(&font.Font, serialized.Chars)
	rl.SetFontRecs(&font.Font, serialized.Recs)

	font.Font.Texture = rl.LoadTextureFromImage(atlasImg)

	if font.IsSdfFont {
		rl.SetTextureFilter(font.Font.Texture, rl.FilterBilinear)
	} else {
		rl.GenTextureMipmaps(&font.Font.Texture)
		rl.SetTextureFilter(font.Font.Texture, rl.FilterTrilinear)
	}

	rl.UnloadImage(atlasImg)

	return font, nil
}
