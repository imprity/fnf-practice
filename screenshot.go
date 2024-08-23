package fnf

import (
	"fmt"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var TheScreenshotManager struct {
	screenshotCounter int
	InputId           InputGroupId
}

func InitScreenshot() {
	ts := &TheScreenshotManager

	ts.InputId = NewInputGroupId()
}

func takeScreenshot() {
	ts := &TheScreenshotManager

	dirPath, err := RelativePath("./")
	if err != nil {
		ErrorLogger.Printf("failed to take screenshot: %v", err)
		DisplayAlert("failed to take screenshot")
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		ErrorLogger.Printf("failed to take screenshot: %v", err)
		DisplayAlert("failed to take screenshot")
		return
	}

	const fmtStr = "screenshot-%03d.png"

	screenshotName := fmt.Sprintf(fmtStr, ts.screenshotCounter)

	for _, entry := range entries {
		if entry.Name() == screenshotName {
			ts.screenshotCounter += 1
			screenshotName = fmt.Sprintf(fmtStr, ts.screenshotCounter)
		}
	}

	// actually take screenshot
	img := rl.LoadImageFromTexture(TheRenderTexture.Texture)
	if !rl.IsImageReady(img) {
		ErrorLogger.Printf("failed to take screenshot: failed to load image from render texture")
		DisplayAlert("failed to take screenshot")
		return
	}
	defer rl.UnloadImage(img)

	rl.ImageFlipVertical(img)

	data := rl.ExportImageToMemory(*img, ".png")

	fullPath := filepath.Join(dirPath, screenshotName)

	err = os.WriteFile(fullPath, data, 0664)
	if err != nil {
		ErrorLogger.Printf("failed to take screenshot: %v", err)
		DisplayAlert("failed to take screenshot")
		return
	}

	FnfLogger.Printf("saved screenshot as %s", fullPath)
	DisplayAlert(fmt.Sprintf("saved screenshot as %s", screenshotName))

	ts.screenshotCounter += 1
}

func UpdateScreenshot() {
	ts := &TheScreenshotManager

	if AreKeysPressed(ts.InputId, TheKM[ScreenshotKey]) {
		takeScreenshot()
	}
}
