package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"fnf-practice/unitext"
)

func InitUnitext() {
	// set unitext CacheDir
	cacheDir, err := RelativePath("./fnf-font-cache")
	if err != nil {
		// TODO : handle error
		ErrorLogger.Fatal(err)
	}

	unitext.CacheDir = cacheDir

	unitext.Logger = FnfLogger
}

func RenderUnicodeText(
	text string,
	desiredFont unitext.DesiredFont, fontSize float32, textColor Color,
) *rl.Image {
	img, err := unitext.RenderUnicodeText(text, desiredFont, fontSize, textColor.ToImageColor())

	if err != nil {
		// TODO : handle error
		ErrorLogger.Fatal(err)
	}

	return rl.NewImageFromImagePro(img, rl.Color{0, 0, 0, 0}, false)
}
