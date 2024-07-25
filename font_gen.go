//go:build ignore

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqweek/dialog"

	rl "github.com/gen2brain/raylib-go/raylib"

	fnf "fnf-practice"
)

type FontGenScreen struct {
	InputId fnf.InputGroupId

	font         fnf.FnfFont
	sdfFont      fnf.FnfFont
	isFontLoaded bool

	fontData []byte

	fontGenSize int32

	sdfPadding        int32
	sdfOnEdgeValue    uint8
	sdfPixelDistScale float32

	needToReloadFont bool

	fontRenderSize float32
	strokeThick    float32

	fontName string

	texturePos rl.Vector2

	hiddenItems []fnf.MenuItemId

	menu *fnf.MenuDrawer
}

func NewFontGenScreen() *FontGenScreen {
	fg := new(FontGenScreen)

	fg.InputId = fnf.NewInputGroupId()

	fg.fontGenSize = 64
	fg.sdfPadding = 21
	fg.sdfOnEdgeValue = 200
	fg.sdfPixelDistScale = 10

	fg.fontRenderSize = 64
	fg.strokeThick = 5

	fg.menu = fnf.NewMenuDrawer()

	// add menu itmes
	{
		newItem := func() *fnf.MenuItem {
			item := fnf.NewMenuItem()
			item.Color = fnf.FnfColor{0, 0, 0, 220}
			item.ColorSelected = fnf.FnfColor{0, 0, 0, 255}

			item.StrokeColor = fnf.FnfColor{255, 255, 255, 220}
			item.StrokeColorSelected = fnf.FnfColor{255, 255, 255, 255}

			item.StrokeWidth = 0
			item.StrokeWidthSelected = 6

			return item
		}

		loadFont := newItem()
		loadFont.Name = "Load Font"
		loadFont.Type = fnf.MenuItemTrigger
		fg.menu.AddItems(loadFont)

		padding := newItem()
		padding.Type = fnf.MenuItemNumber
		padding.Name = "sdfPadding"
		padding.NValueMin = 0
		padding.NValueMax = 50
		padding.NValue = float32(fg.sdfPadding)
		padding.NValueInterval = 1
		padding.NValueFmtString = "%1.f"
		padding.NumberCallback = func(v float32) {
			fg.sdfPadding = int32(v)
			fg.needToReloadFont = true
		}
		fg.menu.AddItems(padding)

		onEdge := newItem()
		onEdge.Type = fnf.MenuItemNumber
		onEdge.Name = "sdfOnEdgeValue"
		onEdge.NValueMin = 0
		onEdge.NValueMax = 255
		onEdge.NValue = float32(fg.sdfOnEdgeValue)
		onEdge.NValueInterval = 1
		onEdge.NValueFmtString = "%1.f"
		onEdge.NumberCallback = func(v float32) {
			fg.sdfOnEdgeValue = uint8(v)
			fg.needToReloadFont = true
		}
		fg.menu.AddItems(onEdge)

		scale := newItem()
		scale.Type = fnf.MenuItemNumber
		scale.Name = "sdfPixelDistScale"
		scale.NValueMin = 0
		scale.NValueMax = 255
		scale.NValueInterval = 1
		scale.NValue = float32(fg.sdfPixelDistScale)
		scale.NValueFmtString = "%1.f"
		scale.NumberCallback = func(v float32) {
			fg.sdfPixelDistScale = (v)
			fg.needToReloadFont = true
		}
		fg.menu.AddItems(scale)

		ftg := newItem()
		ftg.Type = fnf.MenuItemNumber
		ftg.Name = "font generation size"
		ftg.NValueMin = 0
		ftg.NValueMax = 1000
		ftg.NValueInterval = 1
		ftg.NValue = float32(fg.fontGenSize)
		ftg.NValueFmtString = "%1.f"
		ftg.NumberCallback = func(v float32) {
			fg.fontGenSize = int32(v)
			fg.needToReloadFont = true
		}
		fg.menu.AddItems(ftg)

		ftr := newItem()
		ftr.Type = fnf.MenuItemNumber
		ftr.Name = "font render size"
		ftr.NValueMin = 0
		ftr.NValueMax = 1000
		ftr.NValueInterval = 1
		ftr.NValue = float32(fg.fontRenderSize)
		ftr.NValueFmtString = "%1.f"
		ftr.NumberCallback = func(v float32) {
			fg.fontRenderSize = v
		}
		fg.menu.AddItems(ftr)

		thick := newItem()
		thick.Type = fnf.MenuItemNumber
		thick.Name = "stroke"
		thick.NValueMin = 0
		thick.NValueMax = 1000
		thick.NValueInterval = 1
		thick.NValue = float32(fg.strokeThick)
		thick.NValueFmtString = "%1.f"
		thick.NumberCallback = func(v float32) {
			fg.strokeThick = v
		}
		fg.menu.AddItems(thick)

		saveFontImpl := func(saveSdf bool) {
			fnf.ShowTransition(fnf.BlackPixel, func() {
				fnf.HideTransition()
				factory := dialog.File().Title("choose where to save font file")

				fontName := strings.Trim(fg.fontName, " ")
				fontName = strings.ReplaceAll(fg.fontName, " ", "-")

				if saveSdf {
					fontName += "-SDF"
				}

				fontName += fmt.Sprintf("-%d", fg.fontGenSize)

				factory.SetStartFile(fontName)
				factory.SetStartDir(".")

				fontPath, dialogErr := factory.Save()

				if dialogErr != nil && !errors.Is(dialogErr, dialog.ErrCancelled) {
					fnf.ErrorLogger.Fatal(dialogErr)
				}

				if errors.Is(dialogErr, dialog.ErrCancelled) {
					return
				}

				var data []byte
				var err error

				if saveSdf {
					data, err = fnf.SerializeSdfFont(
						fg.fontData, int32(fg.fontGenSize),
						fg.sdfPadding, fg.sdfOnEdgeValue, fg.sdfPixelDistScale)
				} else {
					data, err = fnf.SerializeFont(fg.fontData, int32(fg.fontGenSize))
				}

				if err != nil {
					fnf.DisplayAlert("Failed to serialize font!")
					fnf.ErrorLogger.Printf("failed to serialize font %v", err)
					return
				}

				if err = os.WriteFile(fontPath, data, 0664); err != nil {
					fnf.DisplayAlert("Failed to save serialized font!")
					fnf.ErrorLogger.Printf("Failed to save serialized font %v", err)
					return
				}

				fnf.DisplayAlert(fmt.Sprintf("saved to %v", filepath.Base(fontPath)))
			})
		}

		saveFont := newItem()
		saveFont.Name = "Save Font"
		saveFont.Type = fnf.MenuItemTrigger
		saveFont.TriggerCallback = func() {
			saveFontImpl(false)
		}
		fg.menu.AddItems(saveFont)

		saveFontSdf := newItem()
		saveFontSdf.Name = "Save Font As SDF"
		saveFontSdf.Type = fnf.MenuItemTrigger
		saveFontSdf.TriggerCallback = func() {
			saveFontImpl(true)
		}
		fg.menu.AddItems(saveFontSdf)

		fg.hiddenItems = []fnf.MenuItemId{
			padding.Id,
			onEdge.Id,
			scale.Id,
			ftg.Id,
			ftr.Id,
			thick.Id,
			saveFont.Id,
			saveFontSdf.Id,
		}

		loadFont.TriggerCallback = func() {
			fnf.ShowTransition(fnf.BlackPixel, func() {
				defer fnf.HideTransition()

				factory := dialog.File().Title("choose font file to load")
				factory.Filter("font files", "ttf", "otf")
				factory.SetStartDir(".")

				fontPath, dialogErr := factory.Load()

				if dialogErr != nil && !errors.Is(dialogErr, dialog.ErrCancelled) {
					fnf.ErrorLogger.Fatal(dialogErr)
				}

				if errors.Is(dialogErr, dialog.ErrCancelled) {
					return
				}

				name := filepath.Base(fontPath)
				// remove extension
				if index := strings.IndexByte(name, '.'); index >= 0 {
					name = name[0:index]
				}

				fg.fontName = name

				if data, err := os.ReadFile(fontPath); err == nil {
					fg.fontData = data
					fg.ReloadFont()
				} else {
					fnf.ErrorLogger.Printf("failed to load font : %v", err)
					fnf.DisplayAlert("Failed to load font")
				}
			})
		}
	}

	return fg
}

func (fg *FontGenScreen) ReloadFont() {
	newSdfFont := fnf.LoadSdfFontFromMemory(
		fg.fontData, int32(fg.fontGenSize), nil,
		fg.sdfPadding, fg.sdfOnEdgeValue, fg.sdfPixelDistScale,
	)

	newFont := fnf.LoadFontAlphaPremultiply(fg.fontData, int32(fg.fontGenSize), nil)
	rl.GenTextureMipmaps(&newFont.Font.Texture)
	rl.SetTextureFilter(newFont.Font.Texture, rl.FilterTrilinear)

	if rl.IsFontReady(newSdfFont.Font) && rl.IsFontReady(newFont.Font) {
		rl.UnloadFont(fg.font.Font)
		rl.UnloadFont(fg.sdfFont.Font)

		fg.isFontLoaded = true
		fg.font = newFont
		fg.sdfFont = newSdfFont
	} else {
		fnf.DisplayAlert("Failed to load font")
	}
}

func (fg *FontGenScreen) Update(deltaTime time.Duration) {
	if anyDown, _ := fnf.AnyKeyDown(fg.InputId); !anyDown {
		if fg.needToReloadFont && fg.isFontLoaded {
			fg.ReloadFont()
			fg.needToReloadFont = false
		}
	}

	for _, id := range fg.hiddenItems {
		fg.menu.SetItemHidden(id, !fg.isFontLoaded)
	}

	if fg.isFontLoaded && fnf.IsMouseButtonDown(fg.InputId, rl.MouseButtonLeft) {
		delta := fnf.MouseDelta()
		fg.texturePos.X += delta.X
		fg.texturePos.Y += delta.Y
	}

	if fg.isFontLoaded && fnf.AreKeysPressed(fg.InputId, rl.KeyEscape) {
		fg.menu.IsHidden = !fg.menu.IsHidden
	}

	fg.menu.Update(deltaTime)
}

func (fg *FontGenScreen) Draw() {
	rl.ClearBackground(fnf.ToRlColor(fnf.FnfColor{128, 128, 128, 255}))

	if fg.isFontLoaded {
		rl.BeginBlendMode(rl.BlendAlpha)
		rl.DrawTexture(fg.sdfFont.Font.Texture, int32(fg.texturePos.X), int32(fg.texturePos.Y), rl.White)
		fnf.FnfEndBlendMode()

		// draw font name
		rl.BeginBlendMode(rl.BlendAlpha)
		rl.DrawText(fg.fontName, 9, 11, 30, rl.White)
		rl.DrawText(fg.fontName, 10, 10, 30, rl.Black)
		fnf.FnfEndBlendMode()

		const str = "Hello World!"
		const margin = 20

		var pos rl.Vector2
		pos.Y = 20

		size := MeasureText(fg.font, str, fg.fontRenderSize, 0)
		pos.X = fnf.SCREEN_WIDTH - size.X - 20
		fnf.DrawText(fg.font, str, pos, fg.fontRenderSize, 0, rl.Black)
		pos.Y += size.Y + 20

		size = MeasureText(fg.sdfFont, str, fg.fontRenderSize, 0)
		pos.X = fnf.SCREEN_WIDTH - size.X - 20
		fnf.DrawText(fg.sdfFont, str, pos, fg.fontRenderSize, 0, rl.Black)
		pos.Y += size.Y + 20

		fnf.DrawTextOutlined(fg.sdfFont, str, pos, fg.fontRenderSize, 0, rl.Black, rl.White, fg.strokeThick)
	}

	fg.menu.Draw()

	// draw help message
	if fg.isFontLoaded {
		const str = "Press ESC to hide menu"
		rl.BeginBlendMode(rl.BlendAlpha)
		rl.DrawText(str, 10, fnf.SCREEN_HEIGHT-50, 30, rl.Red)
		fnf.FnfEndBlendMode()
	}
}

func (fg *FontGenScreen) BeforeScreenTransition() {
	fg.menu.BeforeScreenTransition()
}

func (fg *FontGenScreen) BeforeScreenEnd() {
	// pass
}

func (fg *FontGenScreen) Free() {
	fg.menu.Free()

	rl.UnloadFont(fg.font.Font)
	rl.UnloadFont(fg.sdfFont.Font)
}

func main() {
	fnf.OverrideFirstScreen(func() fnf.Screen {
		return NewFontGenScreen()
	})

	fnf.RunApplication()
}
