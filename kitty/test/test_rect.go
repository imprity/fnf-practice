package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"fmt"

	"kitty"
)

const (
	SCREEN_WIDTH  = 500
	SCREEN_HEIGHT = 500
)

var Rect1 = kitty.Fr(0,0, 200, 100)
var Rect2 = kitty.IntsToFr(SCREEN_WIDTH - 100, SCREEN_HEIGHT - 200, 100, 200)

var Inset float64

type Game struct {}

func (g *Game) Update() error {
	const moveSpeed float64 = 5.0

	dir1 := kitty.V(0,0)
	dir2 := kitty.V(0,0)

	_= dir2

	if ebiten.IsKeyPressed(ebiten.KeyW){dir1.Y -= moveSpeed}
	if ebiten.IsKeyPressed(ebiten.KeyS){dir1.Y += moveSpeed}
	if ebiten.IsKeyPressed(ebiten.KeyD){dir1.X += moveSpeed}
	if ebiten.IsKeyPressed(ebiten.KeyA){dir1.X -= moveSpeed}

	if ebiten.IsKeyPressed(ebiten.KeyUp){dir2.Y -= moveSpeed}
	if ebiten.IsKeyPressed(ebiten.KeyDown){dir2.Y += moveSpeed}
	if ebiten.IsKeyPressed(ebiten.KeyRight){dir2.X += moveSpeed}
	if ebiten.IsKeyPressed(ebiten.KeyLeft){dir2.X -= moveSpeed}

	Rect1 = Rect1.Add(dir1)
	Rect2 = Rect2.Add(dir2)

	_, wheelY := ebiten.Wheel()
	Inset += float64(wheelY * 3)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	union := Rect1.Union(Rect2)
	kitty.DrawRect(screen, union, kitty.Color255(170, 170, 0, 255))

	kitty.DrawRect(screen, Rect1, kitty.Color255(170, 0, 0, 255))
	kitty.DrawRect(screen, Rect2, kitty.Color255(0, 0, 170, 255))

	intersect := Rect1.Intersect(Rect2)
	kitty.DrawRect(screen, intersect, kitty.Color255(170, 0, 170, 255))

	insetRect := intersect.Inset(Inset)
	kitty.DrawRect(screen, insetRect, kitty.Color255(255, 255, 255, 90))

	ebitenutil.DebugPrint(screen,
		fmt.Sprintf(
			"Move rects with\n" +
			"wasd and arrow keys\n" +
			"\n" +
			"Inset Value : %.2f",
		Inset),
	)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("Rect Test")

	var err error
	if err = ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
