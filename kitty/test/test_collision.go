package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"kitty"
)

const (
	SCREEN_WIDTH  = 500
	SCREEN_HEIGHT = 500
)

type Game struct {}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	rect := kitty.FRect{30,40, 100, 130}

	curX, curY := ebiten.CursorPosition()

	circle := kitty.Circle{R: 100}
	circle.X = float64(curX)
	circle.Y = float64(curY)

	circle2 := kitty.Circle{X : 200, Y: 100, R: 30}

	if kitty.CollisionRectCircle(rect, circle){
		kitty.DrawRect(screen, rect, kitty.Color255(255,0,0,255))
	}else{
		kitty.DrawRect(screen, rect, kitty.Color255(0,255,0,255))
	}

	if kitty.CollsionCircleCircle(circle, circle2){
		kitty.DrawCircle(screen, circle2, kitty.Color255(255,0,0,255))
	}else{
		kitty.DrawCircle(screen, circle2, kitty.Color255(0,255,0,255))
	}

	kitty.DrawCircle(screen, circle, kitty.Color255(0,0,255,100))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("Collsion Test")

	var err error
	if err = ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}