package main

import (
	_ "embed"
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

type TransitionManager struct {
	DiamonWidth  float32
	DiamonHeight float32

	ShowTransition bool

	AnimStartedAt time.Duration
	AnimDuration  time.Duration

	ImgTexture  rl.Texture2D
	MaskTexture rl.RenderTexture2D

	Callback func()

	MaskShader        rl.Shader
	TransitionTexture rl.RenderTexture2D

	MaskLoc  int32
	ImageLoc int32

	ScreenSizeLoc int32

	ImageSizeLoc int32
}

var TheTransitionManager TransitionManager

//go:embed mask.fs
var maskFsShader string

func InitTransition() {
	manager := &TheTransitionManager

	manager.DiamonWidth = 80
	manager.DiamonHeight = 100

	manager.AnimStartedAt = Years150
	manager.AnimDuration = time.Millisecond * 300

	manager.MaskTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)
	manager.TransitionTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)

	if !rl.IsRenderTextureReady(manager.MaskTexture) {
		ErrorLogger.Fatal("failed to load mask render texture")
	}

	manager.MaskShader = rl.LoadShaderFromMemory("", maskFsShader)

	// NOTE : There's no point in checking if shader is successfuly loaded
	// using IsShaderReady since raylib assigns default shader if something goes wrong
	// look at tracelog to check if anything failed...

	// get locations
	manager.MaskLoc = rl.GetShaderLocation(manager.MaskShader, "mask")
	manager.ImageLoc = rl.GetShaderLocation(manager.MaskShader, "image")

	manager.ScreenSizeLoc = rl.GetShaderLocation(manager.MaskShader, "screenSize")
	manager.ImageSizeLoc = rl.GetShaderLocation(manager.MaskShader, "imageSize")
}

func FreeTransition() {
	manager := &TheTransitionManager

	rl.UnloadRenderTexture(manager.MaskTexture)
	rl.UnloadRenderTexture(manager.TransitionTexture)
	rl.UnloadShader(manager.MaskShader)
}

func CallTransitionCallbackIfNeeded() {
	manager := &TheTransitionManager

	if manager.ShowTransition && TimeSinceNow(manager.AnimStartedAt) > manager.AnimDuration {
		if manager.Callback != nil {
			manager.Callback()
			manager.Callback = nil
		}
	}
}

func UpdateTransitionTexture() {
	manager := &TheTransitionManager

	if manager.ImgTexture.ID <= 0 { //just in case
		return
	}

	timeT := float32(GlobalTimerNow() - manager.AnimStartedAt)
	timeT /= float32(manager.AnimDuration)

	if timeT < 0 || timeT > 1 {
		if manager.ShowTransition {
			DrawPatternBackground(manager.ImgTexture, 0, 0, rl.Color{255, 255, 255, 255})
		}
		return
	}

	diaW := manager.DiamonWidth
	diaH := manager.DiamonHeight

	intW := int(SCREEN_WIDTH / diaW)
	intH := int(SCREEN_HEIGHT / (diaH * 0.5))

	diaNx1 := intW + 2
	diaNx2 := diaNx1 + 1

	diaNy := intH + 3

	count := 0

	if diaNy%2 == 1 {
		count = diaNx1*(diaNy/2+1) + diaNx2*diaNy/2
	} else {
		count = diaNx1*diaNy/2 + diaNx2*diaNy/2
	}

	diaTotalW1 := f32(diaNx1) * diaW
	diaTotalW2 := f32(diaNx2) * diaW

	diaTotalH := f32(diaNy) * diaH * 0.5

	xStart1 := -(diaTotalW1 - SCREEN_WIDTH) * 0.5
	xStart2 := -(diaTotalW2 - SCREEN_WIDTH) * 0.5

	yStart := -(diaTotalH - SCREEN_HEIGHT) * 0.5

	points := make([]rl.Vector2, 4)

	index := 0

	FnfBeginTextureMode(manager.MaskTexture)
	rl.ClearBackground(rl.Color{0, 0, 0, 0})

	for yi := 0; yi < diaNy; yi++ {
		xEnd := diaNx1
		xStart := xStart1

		if yi%2 == 1 {
			xEnd = diaNx2
			xStart = xStart2
		}

		y := yStart + f32(yi)*diaH*0.5

		for xi := 0; xi < xEnd; xi++ {
			x := xStart + f32(xi)*diaW

			scale := float32(1.0)

			t := (f32(index) + Lerp(f32(-count+1), f32(count-1), 1-timeT)) / f32(count-1)
			t = Clamp(t, 0, 1)
			t = 1 - t
			t = t * t
			scale = t

			if !manager.ShowTransition {
				scale = 1 - scale
			}

			points[0] = rl.Vector2{x, y - diaH*scale*0.5}
			points[1] = rl.Vector2{x - diaW*scale*0.5, y}
			points[2] = rl.Vector2{x + diaW*scale*0.5, y}
			points[3] = rl.Vector2{x, y + diaH*scale*0.5}

			rl.DrawTriangleStrip(points, rl.Color{255, 255, 255, 255})
			index++
		}
	}

	FnfEndTextureMode()

	FnfBeginTextureMode(manager.TransitionTexture)
	rl.ClearBackground(rl.Color{0, 0, 0, 0})
	rl.BeginShaderMode(manager.MaskShader)

	rl.SetShaderValueTexture(
		manager.MaskShader, manager.MaskLoc, manager.MaskTexture.Texture)

	rl.SetShaderValueTexture(
		manager.MaskShader, manager.ImageLoc, manager.ImgTexture)

	rl.SetShaderValue(
		manager.MaskShader,
		manager.ScreenSizeLoc,
		[]float32{SCREEN_WIDTH, SCREEN_HEIGHT},
		rl.ShaderUniformVec2)

	rl.SetShaderValue(
		manager.MaskShader,
		manager.ImageSizeLoc,
		[]float32{f32(manager.ImgTexture.Width), f32(manager.ImgTexture.Height)},
		rl.ShaderUniformVec2)

	rl.DrawTexture(manager.MaskTexture.Texture, 0, 0, rl.Color{255, 255, 255, 255})

	rl.EndShaderMode()
	FnfEndTextureMode()
}

func ShowTransition(texture rl.Texture2D, callback func()) {
	TheTransitionManager.ShowTransition = true
	TheTransitionManager.AnimStartedAt = GlobalTimerNow()
	TheTransitionManager.ImgTexture = texture
	TheTransitionManager.Callback = callback

	DisableInputGlobal()
}

func HideTransition() {
	TheTransitionManager.ShowTransition = false
	TheTransitionManager.AnimStartedAt = GlobalTimerNow()

	ClearGlobalInputDisable()
}

func IsTransitionOn() bool {
	return TheTransitionManager.ShowTransition
}
