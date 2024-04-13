package main

import (
	"math"
	"time"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type TransitionManager struct{
	DiamonWidth float32
	DiamonHeight float32

	ShowTransition bool

	AnimStartedAt time.Duration
	AnimDuration time.Duration
} 

var TheTransitionManager TransitionManager

func init(){
	TheTransitionManager.DiamonWidth = 80
	TheTransitionManager.DiamonHeight = 100

	TheTransitionManager.AnimStartedAt = time.Duration(math.MaxInt64) / 2
	TheTransitionManager.AnimDuration = time.Millisecond * 300
}

func DrawTransition(){
	timeT :=  float32(GlobalTimerNow() - TheTransitionManager.AnimStartedAt)
	timeT /= float32(TheTransitionManager.AnimDuration)

	if timeT < 0 || timeT > 1{
		if TheTransitionManager.ShowTransition{
			rl.DrawRectangle(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT, rl.Color{0,0,0,255})
		}
		return
	}


	diaW := TheTransitionManager.DiamonWidth
	diaH :=TheTransitionManager.DiamonHeight

	intW := int(SCREEN_WIDTH / diaW)
	intH := int(SCREEN_HEIGHT / (diaH * 0.5))

	diaNx1 := intW + 2
	diaNx2 := diaNx1 + 1

	diaNy := intH + 3

	count := 0

	if diaNy%2 == 1{
		count = diaNx1 * (diaNy/2 + 1) + diaNx2 * diaNy/2
	}else{
		count = diaNx1 * diaNy/2 + diaNx2 * diaNy/2
	}

	diaTotalW1 := f32(diaNx1) * diaW
	diaTotalW2 := f32(diaNx2) * diaW

	diaTotalH := f32(diaNy) * diaH * 0.5

	xStart1 := -(diaTotalW1 - SCREEN_WIDTH) * 0.5
	xStart2 := -(diaTotalW2 - SCREEN_WIDTH) * 0.5

	yStart := -(diaTotalH - SCREEN_HEIGHT) * 0.5

	points := make([]rl.Vector2, 4)

	index := 0

	for yi := 0; yi < diaNy; yi++{
		xEnd := diaNx1
		xStart := xStart1

		if yi%2 == 1{
			xEnd = diaNx2
			xStart = xStart2
		}

		y := yStart + f32(yi) * diaH * 0.5

		for xi := 0; xi < xEnd; xi++{
			x := xStart + f32(xi) * diaW

			scale := float32(1.0)
			
			t := (f32(index) + Lerp(f32(-count +1), f32(count - 1), 1 - timeT)) / f32(count - 1)
			t = Clamp(t,0,1)
			t = 1 - t
			t = t * t
			scale = t

			if !TheTransitionManager.ShowTransition{
				scale = 1-scale
			}

			points[0] = rl.Vector2{x,                      y - diaH * scale * 0.5}
			points[1] = rl.Vector2{x - diaW * scale * 0.5, y }
			points[2] = rl.Vector2{x + diaW * scale * 0.5, y }
			points[3] = rl.Vector2{x,                      y + diaH * scale * 0.5}

			rl.DrawTriangleStrip(points, rl.Color{0, 0, 0,255})
			index++
		}
	}
}

func ShowTransition(){
	TheTransitionManager.ShowTransition = true
	TheTransitionManager.AnimStartedAt = GlobalTimerNow()
}

func IsShowTransitionDone() bool{
	return GlobalTimerNow() - TheTransitionManager.AnimStartedAt > TheTransitionManager.AnimDuration
}

func HideTransition(){
	TheTransitionManager.ShowTransition = false
	TheTransitionManager.AnimStartedAt = GlobalTimerNow()
}

func IsHideTransitionDone() bool{
	return GlobalTimerNow() - TheTransitionManager.AnimStartedAt > TheTransitionManager.AnimDuration
}

func IsTransitionOn() bool{
	return TheTransitionManager.ShowTransition
}
