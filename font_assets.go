package main

import (
	"bytes"
	"encoding/gob"
	rl "github.com/gen2brain/raylib-go/raylib"
)

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
