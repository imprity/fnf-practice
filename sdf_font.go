package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type SdfFont struct {
	Font rl.Font

	// meta data
	SdfPadding        int32
	SdfOnEdgeValue    uint8
	SdfPixelDistScale float32
}

func LoadSdfFontFromMemory(
	fontData []byte,
	fontSize int32,
	codePoints []rune,
	sdfPadding int32,
	sdfOnEdgeValue uint8,
	sdfPixelDistScale float32,
) SdfFont {
	glyphs := rl.LoadFontDataSdf(
		fontData, fontSize, codePoints, sdfPadding, sdfOnEdgeValue, sdfPixelDistScale)

	image, rects := rl.GenImageFontAtlas(glyphs, fontSize, 0, 1)

	texture := rl.LoadTextureFromImage(image)
	rl.UnloadImage(image)

	rl.SetTextureFilter(texture, rl.FilterBilinear)

	var font SdfFont

	rl.SetFontCharGlyphs(&font.Font, glyphs)
	rl.SetFontRecs(&font.Font, rects)

	font.Font.Texture = texture
	font.Font.BaseSize = fontSize

	font.SdfPadding = sdfPadding
	font.SdfOnEdgeValue = sdfOnEdgeValue
	font.SdfPixelDistScale = sdfPixelDistScale

	return font
}

var TheSdfDrawer struct {
	SdfShader rl.Shader

	UniformLoc int32
}

var sdfShaderFsCode string

func InitSdfFontDrawer() {
	ts := &TheSdfDrawer

	ts.SdfShader = rl.LoadShaderFromMemory("", sdfShaderFsCode)
	ts.UniformLoc = rl.GetShaderLocation(ts.SdfShader, "uValues")
}

func FreeSdfFontDrawer(){
	ts := &TheSdfDrawer
	rl.UnloadShader(ts.SdfShader)
}

// Tint expects alpha premultiplied color
func DrawTextSdf(
	font SdfFont,
	text string,
	position rl.Vector2,
	fontSize float32,
	spacing float32,
	tint rl.Color,
) {
	ts := &TheSdfDrawer

	uniform := make([]float32, 4)
	uniform[0] = f32(font.SdfOnEdgeValue) / 255

	rl.SetShaderValue(ts.SdfShader, ts.UniformLoc, uniform, rl.ShaderUniformVec4)

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)
	rl.BeginShaderMode(ts.SdfShader)

	rl.DrawTextEx(font.Font, text, position, fontSize, spacing, tint)

	rl.EndShaderMode()
	rl.EndBlendMode()
}

// fill and stroke doesn't work well with color with transparency
func DrawTextSdfOutlined(
	font SdfFont,
	text string,
	position rl.Vector2,
	fontSize float32,
	spacing float32,
	fill, stroke rl.Color,
	thick float32,
) {
	if fontSize < 1 {
		return
	}
	ts := &TheSdfDrawer

	uniform := make([]float32, 4)
	uniform[0] = f32(font.SdfOnEdgeValue) / 255
	uniform[1] = thick / 255 * font.SdfPixelDistScale * f32(font.Font.BaseSize) / fontSize

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)
	rl.BeginShaderMode(ts.SdfShader)

	rl.SetShaderValue(ts.SdfShader, ts.UniformLoc, uniform, rl.ShaderUniformVec4)
	rl.DrawTextEx(font.Font, text, position, fontSize, spacing, stroke)

	uniform[1] = 0

	rl.DrawTextEx(font.Font, text, position, fontSize, spacing, fill)

	rl.EndShaderMode()
	rl.EndBlendMode()
}
