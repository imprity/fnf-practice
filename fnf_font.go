package fnf

import (
	_ "embed"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type FnfFont struct {
	Font rl.Font

	// meta data
	IsSdfFont bool

	SdfPadding        int32
	SdfOnEdgeValue    uint8
	SdfPixelDistScale float32
}

func (f *FnfFont) BaseSize() int32 {
	return f.Font.BaseSize
}

// only supports .otf or .ttf
// also it can't draw on image
func LoadFontAlphaPremultiply(fontData []byte, fontSize int32, codePoints []rune) FnfFont {
	var rlFont rl.Font

	rlFont.BaseSize = fontSize
	rlFont.CharsPadding = 4 // default padding set by raylib

	glyphs := rl.LoadFontData(fontData, fontSize, codePoints, rl.FontDefault)
	atlasImg, recs := rl.GenImageFontAtlas(glyphs, fontSize, rlFont.CharsPadding, 0)

	rl.ImageAlphaPremultiply(atlasImg)
	atlasTex := rl.LoadTextureFromImage(atlasImg)

	rl.SetFontCharGlyphs(&rlFont, glyphs)
	rl.SetFontRecs(&rlFont, recs)

	rlFont.Texture = atlasTex

	rl.UnloadImage(atlasImg)

	font := FnfFont{
		Font:      rlFont,
		IsSdfFont: false,
	}

	return font
}

func LoadSdfFontFromMemory(
	fontData []byte,
	fontSize int32,
	codePoints []rune,
	sdfPadding int32,
	sdfOnEdgeValue uint8,
	sdfPixelDistScale float32,
) FnfFont {
	glyphs := rl.LoadFontDataSdf(
		fontData, fontSize, codePoints, sdfPadding, sdfOnEdgeValue, sdfPixelDistScale)

	image, rects := rl.GenImageFontAtlas(glyphs, fontSize, 0, 1)

	texture := rl.LoadTextureFromImage(image)
	rl.UnloadImage(image)

	rl.SetTextureFilter(texture, rl.FilterBilinear)

	var font FnfFont

	font.IsSdfFont = true

	rl.SetFontCharGlyphs(&font.Font, glyphs)
	rl.SetFontRecs(&font.Font, rects)

	font.Font.Texture = texture
	font.Font.BaseSize = fontSize

	font.SdfPadding = sdfPadding
	font.SdfOnEdgeValue = sdfOnEdgeValue
	font.SdfPixelDistScale = sdfPixelDistScale

	return font
}

var TheFnfFontDrawer struct {
	OutlineShader      rl.Shader
	OutlineUniform0Loc int32
	OutlineUniform1Loc int32

	FillShader      rl.Shader
	FillUniform0Loc int32

	RenderTexture rl.RenderTexture2D
}

//go:embed shaders/sdf_outline.fs
var sdfOutlineShaderFsCode string

//go:embed shaders/sdf.fs
var sdfFillShaderFsCode string

func InitFnfFontDrawer() {
	tf := &TheFnfFontDrawer

	tf.OutlineShader = rl.LoadShaderFromMemory("", sdfOutlineShaderFsCode)
	tf.OutlineUniform0Loc = rl.GetShaderLocation(tf.OutlineShader, "uValues0")
	tf.OutlineUniform1Loc = rl.GetShaderLocation(tf.OutlineShader, "uValues1")

	tf.FillShader = rl.LoadShaderFromMemory("", sdfFillShaderFsCode)
	tf.FillUniform0Loc = rl.GetShaderLocation(tf.FillShader, "uValues0")

	tf.RenderTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)
}

func FreeFnfFontDrawer() {
	tf := &TheFnfFontDrawer
	rl.UnloadShader(tf.OutlineShader)
	rl.UnloadShader(tf.FillShader)
	rl.UnloadRenderTexture(tf.RenderTexture)
}

func DrawText(
	font FnfFont,
	text string,
	position rl.Vector2,
	fontSize float32,
	spacing float32,
	tint rl.Color,
) {
	if font.IsSdfFont {
		tf := &TheFnfFontDrawer

		uniform := make([]float32, 4)
		uniform[0] = f32(font.SdfOnEdgeValue) / 255

		rl.BeginShaderMode(tf.FillShader)
		rl.SetShaderValue(tf.FillShader, tf.FillUniform0Loc, uniform, rl.ShaderUniformVec4)

		rl.DrawTextEx(font.Font, text, position, fontSize, spacing, tint)

		rl.EndShaderMode()
	} else {
		rl.DrawTextEx(font.Font, text, position, fontSize, spacing, tint)
	}
}

// This function ignores text outside of game screen.
// May need to change later if there is a need to draw text at some big offscreen buffer
func DrawTextOutlined(
	font FnfFont,
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

	if !font.IsSdfFont {
		DrawText(font, text, position, fontSize, spacing, fill)
		return
	}

	textSize := MeasureText(font, text, fontSize, spacing)
	textRect := rl.Rectangle{
		X: position.X, Y: position.Y, Width: textSize.X, Height: textSize.Y,
	}

	// expand text rect with thick
	textRect = RectExpand(textRect, thick*1.1)

	// if we have nothing to draw skip
	if !rl.CheckCollisionRecs(RectWH(SCREEN_WIDTH, SCREEN_HEIGHT), textRect) {
		return
	}

	tf := &TheFnfFontDrawer

	// ===============================
	// draw text to off screen buffer
	// ===============================
	FnfBeginTextureMode(tf.RenderTexture)

	rl.ClearBackground(ToRlColor(FnfColor{0, 0, 0, 0}))
	rl.SetBlendFactors(rl.RlOne, rl.RlOne, rl.RlMax)
	rl.BeginBlendMode(rl.BlendCustom)
	rl.DrawTextEx(font.Font, text, position, fontSize, spacing, ToRlColor(FnfColor{255, 255, 255, 255}))
	FnfEndBlendMode()

	FnfEndTextureMode()

	// ===============================
	// draw to actual screen
	// ===============================

	rl.BeginShaderMode(tf.OutlineShader)

	uniform0 := make([]float32, 4)
	uniform0[0] = f32(font.SdfOnEdgeValue) / 255
	uniform0[1] = thick / 255 * font.SdfPixelDistScale * f32(font.Font.BaseSize) / fontSize
	rl.SetShaderValue(tf.OutlineShader, tf.OutlineUniform0Loc, uniform0, rl.ShaderUniformVec4)

	uniform1 := make([]float32, 4)
	uniform1[0] = f32(stroke.R) / 255
	uniform1[1] = f32(stroke.G) / 255
	uniform1[2] = f32(stroke.B) / 255
	uniform1[3] = f32(stroke.A) / 255
	rl.SetShaderValue(tf.OutlineShader, tf.OutlineUniform1Loc, uniform1, rl.ShaderUniformVec4)

	intersect := RectIntersect(textRect, RectWH(SCREEN_WIDTH, SCREEN_HEIGHT))

	//	0 -- 3
	//	|    |
	//	|    |
	//	1 -- 2

	var uvs [4]rl.Vector2

	uvs[0] = rl.Vector2{intersect.X / SCREEN_WIDTH, intersect.Y / SCREEN_HEIGHT}
	uvs[1] = rl.Vector2{intersect.X / SCREEN_WIDTH, (intersect.Y + intersect.Height) / SCREEN_HEIGHT}
	uvs[2] = rl.Vector2{(intersect.X + intersect.Width) / SCREEN_WIDTH, (intersect.Y + intersect.Height) / SCREEN_HEIGHT}
	uvs[3] = rl.Vector2{(intersect.X + intersect.Width) / SCREEN_WIDTH, intersect.Y / SCREEN_HEIGHT}

	for i := range uvs {
		uvs[i].Y = 1 - uvs[i].Y
	}

	var verts [4]rl.Vector2

	verts[0] = rl.Vector2{intersect.X, intersect.Y}
	verts[1] = rl.Vector2{intersect.X, (intersect.Y + intersect.Height)}
	verts[2] = rl.Vector2{(intersect.X + intersect.Width), (intersect.Y + intersect.Height)}
	verts[3] = rl.Vector2{(intersect.X + intersect.Width), intersect.Y}

	DrawTextureUvVertices(
		tf.RenderTexture.Texture,
		uvs, verts,
		fill,
	)

	rl.EndShaderMode()
}

func MeasureText(font FnfFont, text string, fontSize float32, spacing float32) rl.Vector2 {
	return rl.MeasureTextEx(font.Font, text, fontSize, spacing)
}
