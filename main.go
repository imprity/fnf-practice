package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"log"
	"encoding/json"
	"os"
)

type RawFnfNote struct{
	MustHitSection bool
	SectionNotes [][]int
}

type RawFnfSong struct{
	Song string
	Notes []RawFnfNote
}

type RawFnfJson struct{
	Song RawFnfSong
}

const (
	ScreenWidth  = 320
	ScreenHeight = 240
)

type App struct {
}

func (app *App) Update() error {
	return nil
}

func (app *App) Draw(screen *ebiten.Image) {
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	const inputJsonPath string = "./tutorial.json" 
	var err error
	var jsonBlob []byte

	if jsonBlob, err = os.ReadFile(inputJsonPath); err != nil{
		log.Fatal(err)
	}

	var rawFnfJson RawFnfJson

	if err = json.Unmarshal(jsonBlob, &rawFnfJson); err != nil{
		log.Fatal(err)
	}
	
	app := new(App)
	
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("fnaf-practice")

	if err = ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}