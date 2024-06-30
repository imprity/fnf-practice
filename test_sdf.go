package main

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var _ = fmt.Printf

/*
func init() {
	OverrideFirstScreen(func() Screen {
		return NewSdfTestScreen()
	})
}
*/

type SdfTestScreen struct {
	InputId InputGroupId

	fontSizeRender float32

	sdfFont SdfFont

	sdfShader rl.Shader

	uValuesLoc int32
	uValues    []float32

	menu *MenuDrawer

	drawRegularTextOverly bool

	pinText bool
	pinPos  rl.Vector2

	pointerCircle float32

	needToReloadSdf bool
}

func (st *SdfTestScreen) LoadSdfFontShader() {
	rl.UnloadShader(st.sdfShader)

	st.sdfShader = rl.LoadShader("", "./shaders/test_sdf.fs")

	st.uValuesLoc = rl.GetShaderLocation(st.sdfShader, "uValues")
}

func (st *SdfTestScreen) ReloadSdf() {
	// everything will be unloaded (like textures and stuff) will be freed
	// alongside font
	rl.UnloadFont(st.sdfFont.Font)

	st.sdfFont = LoadSdfFontFromMemory(
		fontBoldData,
		st.sdfFont.Font.BaseSize,
		nil,
		st.sdfFont.SdfPadding,
		st.sdfFont.SdfOnEdgeValue,
		st.sdfFont.SdfPixelDistScale,
	)
}

func NewSdfTestScreen() *SdfTestScreen {
	st := new(SdfTestScreen)
	st.InputId = NewInputGroupId()

	st.fontSizeRender = 64

	st.sdfFont.Font.BaseSize = 64

	st.sdfFont.SdfPadding = 21
	st.sdfFont.SdfOnEdgeValue = 200
	st.sdfFont.SdfPixelDistScale = 10

	st.pointerCircle = 5

	st.uValues = make([]float32, 4)
	for i := range st.uValues {
		st.uValues[i] = 0.1
	}

	st.menu = NewMenuDrawer()

	{
		padding := NewMenuItem()
		padding.Type = MenuItemNumber
		padding.Name = "sdfPadding"
		padding.NValueMin = 0
		padding.NValueMax = 50
		padding.NValue = f32(st.sdfFont.SdfPadding)
		padding.NValueInterval = 1
		padding.NValueFmtString = "%1.f"
		padding.NumberCallback = func(v float32) {
			st.sdfFont.SdfPadding = i32(v)
			st.needToReloadSdf = true
		}
		st.menu.AddItems(padding)

		onEdge := NewMenuItem()
		onEdge.Type = MenuItemNumber
		onEdge.Name = "sdfOnEdgeValue"
		onEdge.NValueMin = 0
		onEdge.NValueMax = 255
		onEdge.NValue = f32(st.sdfFont.SdfOnEdgeValue)
		onEdge.NValueInterval = 1
		onEdge.NValueFmtString = "%1.f"
		onEdge.NumberCallback = func(v float32) {
			st.sdfFont.SdfOnEdgeValue = uint8(v)
			st.needToReloadSdf = true
		}
		st.menu.AddItems(onEdge)

		scale := NewMenuItem()
		scale.Type = MenuItemNumber
		scale.Name = "sdfPixelDistScale"
		scale.NValueMin = 0
		scale.NValueMax = 255
		scale.NValueInterval = 1
		scale.NValue = f32(st.sdfFont.SdfPixelDistScale)
		scale.NValueFmtString = "%1.f"
		scale.NumberCallback = func(v float32) {
			st.sdfFont.SdfPixelDistScale = (v)
			st.needToReloadSdf = true
		}
		st.menu.AddItems(scale)

		ftg := NewMenuItem()
		ftg.Type = MenuItemNumber
		ftg.Name = "font size for creating atlas"
		ftg.NValueMin = 0
		ftg.NValueMax = 1000
		ftg.NValueInterval = 1
		ftg.NValue = f32(st.sdfFont.Font.BaseSize)
		ftg.NValueFmtString = "%1.f"
		ftg.NumberCallback = func(v float32) {
			st.sdfFont.Font.BaseSize = i32(v)
			st.needToReloadSdf = true
		}
		st.menu.AddItems(ftg)

		ftr := NewMenuItem()
		ftr.Type = MenuItemNumber
		ftr.Name = "render font size"
		ftr.NValueMin = 0
		ftr.NValueMax = 1000
		ftr.NValueInterval = 1
		ftr.NValue = f32(st.fontSizeRender)
		ftr.NValueFmtString = "%1.f"
		ftr.NumberCallback = func(v float32) {
			st.fontSizeRender = v
		}
		st.menu.AddItems(ftr)

		for i, v := range st.uValues {
			uv := NewMenuItem()
			uv.Type = MenuItemNumber
			uv.Name = fmt.Sprintf("uValue %d", i)
			uv.NValueMin = 0
			uv.NValueMax = 1
			uv.NValueInterval = 0.005
			uv.NValue = f32(v)
			uv.NValueFmtString = "%0.3f"
			uv.NumberCallback = func(f float32) {
				st.uValues[i] = f
			}
			st.menu.AddItems(uv)
		}
	}

	st.ReloadSdf()
	st.LoadSdfFontShader()

	return st
}

func (st *SdfTestScreen) Update(deltaTime time.Duration) {
	st.menu.Update(deltaTime)

	if anyDown, _ := AnyKeyDown(st.InputId); st.needToReloadSdf && !anyDown {
		st.ReloadSdf()
		st.needToReloadSdf = false
	}

	if AreKeysPressed(st.InputId, rl.KeyT) {
		st.drawRegularTextOverly = !st.drawRegularTextOverly
	}

	if AreKeysPressed(st.InputId, rl.KeyR) {
		st.LoadSdfFontShader()
	}

	if IsMouseButtonPressed(st.InputId, rl.MouseButtonRight) {
		st.pinText = !st.pinText
		st.pinPos = MouseV()
	}

	st.pointerCircle += rl.GetMouseWheelMove()
}

func (st *SdfTestScreen) Draw() {
	rl.ClearBackground(rl.Color{100, 100, 100, 255})

	rl.DrawTexture(st.sdfFont.Font.Texture, 0, 0, rl.Color{255, 255, 255, 255})

	st.menu.Draw()

	textPos := MouseV()

	if st.pinText {
		textPos = st.pinPos
	}

	// draw sdf font ========
	rl.SetShaderValue(st.sdfShader, st.uValuesLoc, st.uValues[:], rl.ShaderUniformVec4)

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)
	rl.BeginShaderMode(st.sdfShader)
	rl.DrawTextEx(st.sdfFont.Font, "Hello World!", textPos, st.fontSizeRender, 0, rl.Color{255, 255, 255, 255})
	rl.EndShaderMode()
	rl.EndBlendMode()
	// ===============================

	if st.drawRegularTextOverly {
		rl.DrawTextEx(FontBold, "Hello World!", textPos, st.fontSizeRender, 0, rl.Color{0, 0, 0, 100})
	}

	rl.DrawCircleV(MouseV(), st.pointerCircle*0.5, rl.Color{255, 0, 0, 100})

	rl.DrawText(fmt.Sprintf("%.2f", st.pointerCircle), 20, SCREEN_HEIGHT-50, 30, rl.Color{255, 0, 0, 255})
}

func (st *SdfTestScreen) BeforeScreenTransition() {
	st.menu.BeforeScreenTransition()
}

func (st *SdfTestScreen) Free() {
	rl.UnloadFont(st.sdfFont.Font)
	st.menu.Free()
}
