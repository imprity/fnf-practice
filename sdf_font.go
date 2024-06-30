package main

import (
	_ "embed"
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

	UniformLoc    int32
	RenderTexture rl.RenderTexture2D
}

//go:embed shaders/sdf.fs
var sdfShaderFsCode string

func InitSdfFontDrawer() {
	ts := &TheSdfDrawer

	ts.SdfShader = rl.LoadShaderFromMemory("", sdfShaderFsCode)
	ts.UniformLoc = rl.GetShaderLocation(ts.SdfShader, "uValues")
	ts.RenderTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)
}

func FreeSdfFontDrawer() {
	ts := &TheSdfDrawer
	rl.UnloadShader(ts.SdfShader)
	rl.UnloadRenderTexture(ts.RenderTexture)
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

// Fill and stroke doesn't work well with color with transparency.
// Set alpha to control transparency
func DrawTextSdfOutlined(
	font SdfFont,
	text string,
	position rl.Vector2,
	fontSize float32,
	spacing float32,
	fill, stroke rl.Color, alpha float64,
	thick float32,
) {
	if fontSize < 1 {
		return
	}

	if position.X > SCREEN_HEIGHT || position.Y > SCREEN_WIDTH{
		return
	}

	ts := &TheSdfDrawer
	uniform := make([]float32, 4)

	FnfBeginTextureMode(ts.RenderTexture)

	rl.ClearBackground(rl.Color{0, 0, 0, 0})

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)

	rl.BeginShaderMode(ts.SdfShader)
	{
		uniform[0] = f32(font.SdfOnEdgeValue) / 255
		uniform[1] = thick / 255 * font.SdfPixelDistScale * f32(font.Font.BaseSize) / fontSize
		rl.SetShaderValue(ts.SdfShader, ts.UniformLoc, uniform, rl.ShaderUniformVec4)
		rl.DrawTextEx(font.Font, text, position, fontSize, spacing, stroke)

	}
	rl.EndShaderMode()

	rl.BeginShaderMode(ts.SdfShader)
	{

		uniform[1] = 0
		rl.SetShaderValue(ts.SdfShader, ts.UniformLoc, uniform, rl.ShaderUniformVec4)

		rl.DrawTextEx(font.Font, text, position, fontSize, spacing, fill)

	}
	rl.EndShaderMode()

	rl.EndBlendMode()

	FnfEndTextureMode()

	rl.DrawTexturePro(
		ts.RenderTexture.Texture,
		rl.Rectangle{0, 0, SCREEN_WIDTH, -SCREEN_HEIGHT},
		GetScreenRect(),
		rl.Vector2{},
		0,
		rl.Color{255, 255, 255, uint8(255 * alpha)},
	)

}
