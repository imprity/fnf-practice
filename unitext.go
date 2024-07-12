package fnf

import (
	_ "embed"
	"image"

	rl "github.com/gen2brain/raylib-go/raylib"

	"fnf-practice/unitext"
)

//go:embed fonts/dejavu-fonts-ttf-2.37/ttf/DejaVuSans.ttf
var dejavuFontData []byte

var unitextFailed bool = false

func InitUnitext() {
	// set unitext CacheDir
	cacheDir, err := RelativePath("./fnf-font-cache")

	if err != nil {
		ErrorLogger.Printf("failed to init unitext : %v\n", err)
		ErrorLogger.Printf("using dejavu font as backup\n")

		unitextFailed = true
	} else {
		unitext.CacheDir = cacheDir
		unitext.Logger = FnfLogger
	}
}

func RenderUnicodeText(
	text string,
	desiredFont unitext.DesiredFont, fontSize float32, textColor FnfColor,
) *rl.Image {

	var imageImg image.Image
	var err error

	if unitextFailed {
		goto UNITEXT_FAIL
	}

	imageImg, err = unitext.RenderUnicodeText(text, desiredFont, fontSize, rl.Color(textColor))

	if err != nil {
		ErrorLogger.Printf("failed to use unitext : %v\n", err)
		ErrorLogger.Printf("using dejavu font as backup\n")
		goto UNITEXT_FAIL
	} else {
		return rl.NewImageFromImagePro(imageImg, rl.Color{0, 0, 0, 0}, true)
	}

UNITEXT_FAIL:
	runes := unitext.StringToRunes(text)
	font := rl.LoadFontFromMemory(".ttf", dejavuFontData, i32(fontSize), runes)
	defer rl.UnloadFont(font)

	// this might seem weird not to use text argument
	// but unitext.StringToRunes actually replaces invalid codepoints with hexcode
	// like for example "Hello <0xff> <0xfe> <0xfd> World"
	// so it's correct to use runes
	return rl.ImageTextEx(font, string(runes), fontSize, 0, ToRlColor(textColor))
}
