//go:build ignore

package main

import (
	"bufio"
	"fnf-practice/unitext"
	"image/png"
	"image/color"
	"log"
	"os"
)

func main() {
	println("start")
	logger := log.New(os.Stderr, "UNITEXT : ", log.Lshortfile)

	unitext.Logger = logger
	unitext.CacheDir = "font-cache"
	text := "Hello " + "تثذرزسشص" + " world" + "لمنهويء" + "「こんにちは世界」한글도 포함"

	desiredFont := unitext.MakeDesiredFont()
	desiredFont.Families = append(desiredFont.Families, unitext.FontFamilyCursive)

	img, err := unitext.RenderUnicodeText(text, desiredFont, 40, color.NRGBA{255,0,255,255})
	if err != nil{
		log.Fatal(err)
	}

	file, err := os.Create("example-text.png")
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	bufWriter := bufio.NewWriter(file)

	err = png.Encode(bufWriter, img)
	if err != nil {
		log.Fatal(err)
	}
	bufWriter.Flush()

	println("end")
}
