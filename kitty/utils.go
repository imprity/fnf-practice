package kitty

import (
	"bufio"
	"os"

	"image"
	_ "image/jpeg"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

func SaveImageToPng(image image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := png.Encoder{
		CompressionLevel: png.NoCompression,
	}

	writer := bufio.NewWriter(file)

	if err = encoder.Encode(writer, image); err != nil {
		return err
	}

	if err = writer.Flush(); err != nil {
		return err
	}

	return nil
}

func LoadEbitenImage(path string) (*ebiten.Image, error) {
	img, err := LoadImage(path)
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(bufio.NewReader(file))
	if err != nil {
		return nil, err
	}

	return img, nil
}
