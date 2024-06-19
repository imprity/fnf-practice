package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

var _ = fmt.Printf

type SustainTestScreen struct {
	LineWidth float32
	InputId   InputGroupId
}

func NewSustainTestScreen() *SustainTestScreen {
	mt := new(SustainTestScreen)
	mt.InputId = NewInputGroupId()
	mt.LineWidth = 10
	return mt
}

func (mt *SustainTestScreen) Update(deltaTime time.Duration) {
}

func (mt *SustainTestScreen) Draw() {
	rl.ClearBackground(rl.Color{10, 10, 10, 255})
	from := rl.Vector2{
		X: SCREEN_WIDTH * 0.5, Y: SCREEN_HEIGHT * 0.5,
	}

	to := MouseV()

	wheel := rl.GetMouseWheelMove()
	mt.LineWidth += wheel

	if wheel != 0 {
		FnfLogger.Printf("width : %.3f", mt.LineWidth)
	}

	//drawLineWithSustainTex(from, to, mt.LineWidth, rl.Color{255,255,255,255})
	//_=to
	drawLineWithSustainTex(from, to, mt.LineWidth, rl.Color{255, 255, 255, 255})

	rl.DrawCircleV(from, mt.LineWidth*0.5, rl.Color{255, 0, 0, 100})
	rl.DrawCircleV(to, mt.LineWidth*0.5, rl.Color{0, 255, 0, 100})
}

func (mt *SustainTestScreen) BeforeScreenTransition() {
}

func (mt *SustainTestScreen) Free() {
}
