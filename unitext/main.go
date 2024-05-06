//go:build ignore

package main

import (
	"bufio"
	"fnf-practice/unitext"
	"image/png"
	"log"
	"os"
)

func main() {
	println("start")
	logger := log.New(os.Stderr, "UNITEXT : ", log.Lshortfile)
	img := unitext.RenderUnicodeText("한글과  English and 💁👌🎍😍", logger)

	file, err := os.Create("test-text.png")
	defer file.Close()
	if err != nil {
		return
	}

	bufWriter := bufio.NewWriter(file)

	err = png.Encode(bufWriter, img)
	if err != nil {
		return
	}
	bufWriter.Flush()

	println("end")
}
