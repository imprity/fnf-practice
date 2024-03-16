package kitty

import (
	"errors"
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"image/color"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ================================================
// test that performs image comparsions
//
// to update test images provide
//    -update-imgs
//
// to clean fail and diff images provide
//    -clean-imgs
//
// ================================================

func TestDraw(t *testing.T) {
	const imgW = 500
	const imgH = 500
	const gridSize float64 = 20

	var japanFlag *ebiten.Image

	japanFlag = ebiten.NewImage(200, 100)
	japanFlag.Fill(color.NRGBA{255, 255, 255, 255})
	DrawCircle(japanFlag, Circle{100, 50, 35}, Color255(255, 0, 0, 255))

	img := ebiten.NewImage(imgW, imgH)

	var offsetX, offsetY float64 = 0, 0
	_ = offsetY

	for offsetX < imgW+gridSize {
		from, to := Vec2{}, Vec2{}

		from.Y = 0
		to.Y = imgH

		from.X = offsetX
		to.X = offsetX

		DrawLine(img, from, to, 1, Color255(150, 150, 150, 255))

		offsetX += gridSize
	}

	for offsetY < imgW+gridSize {
		from, to := Vec2{}, Vec2{}

		from.X = 0
		to.X = imgW

		from.Y = offsetY
		to.Y = offsetY

		DrawLine(img, from, to, 1, Color255(150, 150, 150, 255))

		offsetY += gridSize
	}

	roundRect1 := FRect{2, 3, 10, 13}
	roundRect1.X *= gridSize
	roundRect1.Y *= gridSize
	roundRect1.W *= gridSize
	roundRect1.H *= gridSize

	DrawRoundRect(img, roundRect1, 30, Color255(150, 150, 150, 100))

	roundRect2 := FRect{8, 6, 10, 6}
	roundRect2.X *= gridSize
	roundRect2.Y *= gridSize
	roundRect2.W *= gridSize
	roundRect2.H *= gridSize

	StrokeRoundRect(img, roundRect2, 30, 5, Color255(230, 230, 230, 100))

	dstRect := FRect{280, 280, 200, 100}
	DrawImageOnImageRect(japanFlag, img, nil, &dstRect)

	dstRect.Y += dstRect.H
	dstRect.W *= 0.8
	srcRect := FRect{0, 0, 100, 50}
	DrawImageOnImageRect(japanFlag, img, &srcRect, &dstRect)

	atTestEnd(t, img, "TestDraw")
}

func TestTextDraw(t *testing.T) {
	const (
		imgW = 800
		imgH = 480
	)

	fontFace := GetDefaultFont()

	img := ebiten.NewImage(imgW, imgH)

	img.Fill(Color255(209, 241, 188, 255).ToImageColor())
	textColor := Color255(86, 111, 70, 255)

	var longText string = "a really really long text\n" +
		"so very very very very~~~ long\n" +
		"so so long\n" +
		"how long can it be?\n" +
		"as long as it wants\n"

	var rect1 FRect = FRect{100, 10, 200, 100}
	DrawRect(img, rect1, Color255(255, 255, 255, 255))
	FitTextInRect(img, rect1, longText, fontFace, textColor)

	var rect2 FRect = FRect{100, 120, 150, 60}
	DrawRect(img, rect2, Color255(255, 255, 255, 255))
	FitTextInRectVert(img, rect2, longText, fontFace, textColor)

	var rect3 FRect = FRect{100, 300, 200, 100}
	DrawRect(img, rect3, Color255(255, 255, 255, 255))
	FitTextInRectHoz(img, rect3, longText, fontFace, textColor)

	op := &ebiten.DrawImageOptions{}
	{
		op.GeoM.Translate(400, 40)
		op.ColorScale.ScaleWithColor(ToImageColor(textColor))
	}

	text.DrawWithOptions(img, longText, fontFace, op)

	atTestEnd(t, img, "TestTextDraw")
}

// ================================================
// function backedns
// ================================================

const testImgDir string = "testimgs"

var updateImages bool
var cleanImages bool

func init() {
	flag.BoolVar(&updateImages, "update-imgs", false, "update test case images")
	flag.BoolVar(&cleanImages, "clean-imgs", false, "delete fail and diff images")
}

func atTestEnd(t *testing.T, img *ebiten.Image, testName string) {
	dirOfThisFile, err := getDirOfThisFile()
	if err != nil {
		t.Errorf("failed to get directory path : %v", err)
		return
	}

	imgDir := filepath.Join(dirOfThisFile, testImgDir)

	exists, err := fileExists(imgDir)
	if err != nil {
		t.Errorf("failed to check if directory \"%v\" exists : %v", imgDir, err)
		return
	}

	if !exists {
		if err = os.Mkdir(imgDir, 0750); err != nil {
			t.Errorf("failed to create directory \"%v\" : %v", imgDir, err)
			return
		}
	}

	compImgPath := filepath.Join(imgDir, testName+".png")

	if updateImages {
		if err = SaveImageToPng(img, compImgPath); err != nil {
			t.Errorf("failed to save \"%v\" : %v", compImgPath, err)
			return
		}
	} else {
		exists, err = fileExists(compImgPath)
		if err != nil {
			t.Errorf("failed to check if file \"%v\" exists : %v", compImgPath, err)
			return
		}

		if !exists {
			t.Logf("image to compare doesn't exist, creating file \"%v\"", compImgPath)

			err = SaveImageToPng(img, compImgPath)

			if err != nil {
				t.Errorf("failed to save \"%v\" : %v", compImgPath, err)
				return
			}
		} else {
			compImg, err := LoadEbitenImage(compImgPath)
			if err != nil {
				t.Errorf("failed to load \"%v\" : %v", compImgPath, err)
				return
			}

			if !imageEq(img, compImg, 2, 0.005) {
				t.Errorf("resulting image is different from \"%v\"", compImgPath)

				failImgPath := filepath.Join(imgDir, testName+"_fail.png")
				diffImgPath := filepath.Join(imgDir, testName+"_diff.png")

				t.Logf("saving failed test image \"%v\"", failImgPath)

				err = SaveImageToPng(img, failImgPath)
				if err != nil {
					t.Errorf("failed to save \"%v\" : %v", failImgPath, err)
				}

				t.Logf("saving diff image \"%v\"", diffImgPath)

				diffImg := createDiffImage(img, compImg)

				err = SaveImageToPng(diffImg, diffImgPath)
				if err != nil {
					t.Errorf("failed to save \"%v\" : %v", diffImgPath, err)
				}
			}
		}
	}
}

func imageEq(img1, img2 *ebiten.Image, coloreTolerance int, diffTolerance float64) bool {
	if !img1.Bounds().Eq(img2.Bounds()) {
		return false
	}

	pixelCount := img1.Bounds().Dx() * img1.Bounds().Dy()
	bufSize := pixelCount * 4

	pixels1 := make([]byte, bufSize)
	pixels2 := make([]byte, bufSize)

	img1.ReadPixels(pixels1)
	img2.ReadPixels(pixels2)

	wrongPixelCount := 0

	for i := 0; i < bufSize; i += 4 {
		r1 := int(pixels1[i])
		g1 := int(pixels1[i+1])
		b1 := int(pixels1[i+2])
		a1 := int(pixels1[i+3])

		r2 := int(pixels2[i])
		g2 := int(pixels2[i+1])
		b2 := int(pixels2[i+2])
		a2 := int(pixels2[i+3])

		if !(AbsI(r1-r2) <= coloreTolerance &&
			AbsI(g1-g2) <= coloreTolerance &&
			AbsI(b1-b2) <= coloreTolerance &&
			AbsI(a1-a2) <= coloreTolerance) {

			wrongPixelCount += 1

			if float64(wrongPixelCount) > diffTolerance*float64(pixelCount) {
				return false
			}
		}
	}

	return true
}

var shaderCode []byte = []byte(
	`
package main

//kage:unit pixels

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4{
	c1 := imageSrc0At(srcPos)
	c2 := imageSrc1At(srcPos)

	return abs(c1 - c2)
}
`)

var diffShader *ebiten.Shader

func init() {
	var err error
	diffShader, err = ebiten.NewShader(shaderCode)
	if err != nil {
		panic(err)
	}
}

func createDiffImage(img1, img2 *ebiten.Image) *ebiten.Image {
	imgRect := img1.Bounds().Union(img2.Bounds())

	diffImg := ebiten.NewImage(imgRect.Dx(), imgRect.Dy())

	drawOp := &ebiten.DrawImageOptions{}

	shaderImg1 := ebiten.NewImage(imgRect.Dx(), imgRect.Dy())
	shaderImg2 := ebiten.NewImage(imgRect.Dx(), imgRect.Dy())

	shaderImg1.DrawImage(img1, drawOp)
	shaderImg2.DrawImage(img2, drawOp)

	shaderOp := &ebiten.DrawRectShaderOptions{}

	shaderOp.Images[0] = shaderImg1
	shaderOp.Images[1] = shaderImg2

	diffImg.DrawRectShader(imgRect.Dx(), imgRect.Dy(), diffShader, shaderOp)

	return diffImg
}

func getDirOfThisFile() (string, error) {
	_, file, _, ok := runtime.Caller(0)

	if !ok {
		return "", fmt.Errorf("rutime.Caller failed")
	}

	return filepath.Dir(file), nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	} else if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

// this is copy pasted
// from ebiten testing.go
type game struct {
	m    *testing.M
	code int
}

func (g *game) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*game) Draw(*ebiten.Image) {
}

func (*game) Layout(int, int) (int, int) {
	return 320, 240
}

func TestMain(m *testing.M) {
	// Run an Ebiten process so that (*Image).At is available.
	flag.Parse()

	var err error

	if cleanImages {
		log.Printf("deleting fail and diff images")

		err = deleteDiffAndFailImages()
		if err != nil {
			log.Printf("failed to delete fail and diff images : %v", err)
		}
	}

	g := &game{
		m:    m,
		code: 1,
	}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}

func deleteDiffAndFailImages() error {
	dirOfThisFile, err := getDirOfThisFile()
	if err != nil {
		return fmt.Errorf("failed to get directory path : %w", err)
	}

	imgDir := filepath.Join(dirOfThisFile, testImgDir)

	exists, err := fileExists(imgDir)
	if err != nil {
		return fmt.Errorf("failed to check if directory \"%v\" exists : %w", imgDir, err)
	}

	if exists {
		entries, err := os.ReadDir(imgDir)

		if err != nil {
			return fmt.Errorf("failed to open directory \"%v\" : %w", imgDir, err)
		}

		for _, entry := range entries {
			if !entry.Type().IsDir() && entry.Type().IsRegular() {
				if strings.HasSuffix(entry.Name(), "_diff.png") || strings.HasSuffix(entry.Name(), "_fail.png") {
					entryPath := filepath.Join(imgDir, entry.Name())
					err = os.Remove(entryPath)

					if err != nil {
						return fmt.Errorf("failed to remove \"%v\" : %w", entryPath, err)
					}
				}
			}
		}
	}

	return nil
}
