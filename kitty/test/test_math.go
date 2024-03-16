package main

import (
	"image/color"
	"github.com/hajimehoshi/ebiten/v2"
	"com/game/kitty"
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
	screen.Fill(color.NRGBA{100,100,100,255})

	curX, curY := ebiten.CursorPosition()

	from := kitty.Vec2{10, 12}
	to := kitty.Vec2{100, 50}
	point := kitty.I_V2(curX, curY)

	kitty.DrawLine(screen, from, to, 3, kitty.Color255(255,0,0,255))

	projection := kitty.GetClosestPointToSegment(from, to, point)

	kitty.DrawCircle(screen, kitty.Circle{point.X, point.Y, 7}, kitty.Color255(0,255,0,255))
	kitty.DrawCircle(screen, kitty.Circle{projection.X, projection.Y, 7}, kitty.Color255(255,255,255,255))

	f1 := kitty.Vec2{100,110}
	t1 := kitty.Vec2{30,75}

	f2 := kitty.Vec2{120, 100}
	t2 := point

	if kitty.SegmentIntersects(f1, t1, f2, t2){
		kitty.DrawLine(screen, f1, t1, 3, kitty.Color255(255,0,0,255))
		kitty.DrawLine(screen, f2, t2, 3, kitty.Color255(255,0,0,255))
	}else{
		kitty.DrawLine(screen, f1, t1, 3, kitty.Color255(255,255,255,255))
		kitty.DrawLine(screen, f2, t2, 3, kitty.Color255(255,255,255,255))
	}
	
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("Math Test")

	var err error
	if err = ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}