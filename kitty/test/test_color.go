package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"kitty"

	"fmt"
)

const (
	SCREEN_WIDTH  = 500
	SCREEN_HEIGHT = 500
)

var COLOR1 kitty.Color = kitty.Color255(255,0,0,255)
var COLOR2 kitty.Color = kitty.Color255(0,0,255,255)
var ALPHA_ON bool = false

type Game struct {}

func (g *Game) Update() error {
	changeColor := func(color kitty.Color, index int, delta float64) kitty.Color{
		switch index{
		case 0:
			color.R += delta
			color.R = kitty.Clamp(color.R, 0, 1)
		case 1:
			color.G += delta
			color.G = kitty.Clamp(color.G, 0, 1)
		case 2:
			color.B += delta
			color.B = kitty.Clamp(color.B, 0, 1)
		case 3:
			color.A += delta
			color.A = kitty.Clamp(color.A, 0, 1)
		default :
			panic("UNREACHABLE")
		}

		return color
	}
	color1Keys := []ebiten.Key {ebiten.KeyQ, ebiten.KeyW, ebiten.KeyE, ebiten.KeyR}
	color2Keys := []ebiten.Key {ebiten.KeyA, ebiten.KeyS, ebiten.KeyD, ebiten.KeyF}

	for i, key := range color1Keys{
		if ebiten.IsKeyPressed(key){
			_, wheel := ebiten.Wheel()
			COLOR1 = changeColor(COLOR1, i, wheel * 0.1)
		}
	}

	for i, key := range color2Keys{
		if ebiten.IsKeyPressed(key){
			_, wheel := ebiten.Wheel()
			COLOR2 = changeColor(COLOR2, i, wheel * 0.1)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ){
		ALPHA_ON = !ALPHA_ON
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	barWidth := 300
	for i:=0; i<barWidth; i++{
		t := float64(i) / float64(barWidth - 1)
		c := kitty.Color{}
		if ALPHA_ON{
			c = kitty.LerpRGBA(COLOR1, COLOR2, t)
		}else{
			c = kitty.LerpRGB(COLOR1, COLOR2, t)
		}

		kitty.DrawLine(
			screen,
			kitty.IntsToV(i, 0),
			kitty.IntsToV(i, 100),
			1,
			c,
		)
	}
	ebitenutil.DebugPrintAt(screen, "LerpRGB", 2, 100)

	for i:=0; i<barWidth; i++{
		t := float64(i) / float64(barWidth - 1)
		c := kitty.Color{}
		if ALPHA_ON{
			c = kitty.LerpHSVA(COLOR1, COLOR2, t)
		}else{
			c = kitty.LerpHSV(COLOR1, COLOR2, t)
		}

		kitty.DrawLine(
			screen,
			kitty.IntsToV(i, 120),
			kitty.IntsToV(i, 220),
			1,
			c,
		)
	}
	ebitenutil.DebugPrintAt(screen, "LerpHSV", 2, 220)

	for i:=0; i<barWidth; i++{
		t := float64(i) / float64(barWidth - 1)
		c := kitty.Color{}
		if ALPHA_ON{
			c = kitty.LerpOkLabA(COLOR1, COLOR2, t)
		}else{
			c = kitty.LerpOkLab(COLOR1, COLOR2, t)
		}

		kitty.DrawLine(
			screen,
			kitty.IntsToV(i, 240),
			kitty.IntsToV(i, 340),
			1,
			c,
		)
	}
	ebitenutil.DebugPrintAt(screen, "LerpOkLab", 2, 340)

	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf("color 1 : r : %.3f, g : %.3f, b : %.3f, a : %.3f\n", COLOR1.R, COLOR1.G, COLOR1.B, COLOR1.A) +
		fmt.Sprintf("color 2 : r : %.3f, g : %.3f, b : %.3f, a : %.3f\n", COLOR2.R, COLOR2.G, COLOR2.B, COLOR2.A) +
		fmt.Sprintf("alpha on (Z): %v\n", ALPHA_ON) +
		fmt.Sprintf("\n") +
		fmt.Sprintf("To chang the color hold key\n") +
		fmt.Sprintf("color 1 : r : Q, g : W, b : E, a : R\n") +
		fmt.Sprintf("color 2 : r : A, g : S, b : D, a : F\n") +
		fmt.Sprintf("and scroll the mouse wheel\n"),
		10,
		365,
	)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("Color Test")

	var a = -1
	a = kitty.Clamp(a, 0, 10)
	println(a)

	var err error
	if err = ebiten.RunGame(&Game{}); err != nil {
		panic(err)
	}
}
