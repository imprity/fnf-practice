package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"time"
	"fmt"
)

var _=fmt.Printf

type SustainTestScreen struct {
	InputId InputGroupId
}

func NewSustainTestScreen() *SustainTestScreen {
	mt := new(SustainTestScreen)
	mt.InputId = NewInputGroupId()
	return mt
}

func (mt *SustainTestScreen) Update(deltaTime time.Duration) {
}

var _width float32 = 10

var _vertices [4]rl.Vector2
var _verticesCount = 0

func (mt *SustainTestScreen) Draw() {
	rl.ClearBackground(rl.Color{10,10,10,255})
	from := rl.Vector2{
		X : SCREEN_WIDTH * 0.5, Y : SCREEN_HEIGHT* 0.5,
	}

	to := MouseV()

	wheel := rl.GetMouseWheelMove()
	_width += wheel

	if wheel != 0{
		FnfLogger.Printf("width : %.3f", _width)
	}

	//DrawSustainBar(from, to, _width, rl.Color{255,255,255,255})
	_=to
	DrawSustainBar(from, to, _width, rl.Color{255,255,255,255})

	rl.DrawCircleV(from, _width * 0.5, rl.Color{255, 0, 0, 100})
	rl.DrawCircleV(to, _width * 0.5, rl.Color{0, 255, 0, 100})

	if IsMouseButtonPressed(mt.InputId, rl.MouseButtonLeft){
		index := _verticesCount % 4
		_vertices[index] = MouseV()

		_verticesCount += 1

	}

	if _verticesCount >= 4{
		DrawTextureUvVertices(
			DirSelectScreen,
			[4]rl.Vector2{
				rl.Vector2{0,0},
				rl.Vector2{0,1},
				rl.Vector2{1,1},
				rl.Vector2{1,0},
			},
			_vertices,
			rl.Color{255,255,255,255},
		)
	}

	for i := range min(_verticesCount, 4){
		rl.DrawCircleV(_vertices[i], 20, rl.Color{255, 255, 255, 100})
		rl.DrawText(
			fmt.Sprintf("%v", i),
			i32(_vertices[i].X),
			i32(_vertices[i].Y),
			20,
			rl.Color{255,0,0,255},
		)
	}
}

func (mt *SustainTestScreen) BeforeScreenTransition() {
}

func (mt *SustainTestScreen) Free() {
}

func DrawSustainBar(from, to rl.Vector2, width float32, color rl.Color){
	if width < 1 {
		return
	}	

	f2t := rl.Vector2Subtract(from, to)
	
	tipHeight := float32(SustainTex.Width) * 0.5

	topSrcRect := rl.Rectangle{
		X : 0, Y : 0,
		Width : f32(SustainTex.Width), Height : tipHeight,
	}

	bottomSrcRect := rl.Rectangle{
		X : 0, Y : f32(SustainTex.Height) - tipHeight,
		Width : f32(SustainTex.Width), Height : tipHeight,
	}

	scale := width / topSrcRect.Width
	angle := f32(math.Atan2(f64(f2t.Y), f64(f2t.X)))

	topVertices := [4]rl.Vector2{
		rl.Vector2{-width * 0.5, -tipHeight * scale}, 
		rl.Vector2{-width * 0.5, 0}, 
		rl.Vector2{width * 0.5,  0}, 
		rl.Vector2{width * 0.5,  -tipHeight * scale}, 
	}

	bottomVertices := [4]rl.Vector2{
		rl.Vector2{-width * 0.5, 0}, 
		rl.Vector2{-width * 0.5, tipHeight * scale}, 
		rl.Vector2{width * 0.5,  tipHeight * scale}, 
		rl.Vector2{width * 0.5,  0}, 
	}

	for i, v := range topVertices{
		v = rl.Vector2Rotate(v, angle + math.Pi * 0.5)
		v.X += from.X
		v.Y += from.Y
		topVertices[i] = v
	}

	for i, v := range bottomVertices{
		v = rl.Vector2Rotate(v, angle + math.Pi * 0.5)
		v.X += to.X
		v.Y += to.Y
		bottomVertices[i] = v
	}

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)

	DrawTextureVertices(
		SustainTex, topSrcRect, topVertices, color,
	)

	// draw the middle part
	{
		middleUvs := [4]rl.Vector2{}

		// calculate middle uvs
		marginNormalized := tipHeight / f32(SustainTex.Height)

		middleUvs[0] = rl.Vector2{0, marginNormalized}
		middleUvs[1] = rl.Vector2{0, 1 - marginNormalized}
		middleUvs[2] = rl.Vector2{1, 1 - marginNormalized}
		middleUvs[3] = rl.Vector2{1, marginNormalized}

		middlePartLength := (f32(SustainTex.Height) - tipHeight * 2) * scale

		t2b0 := rl.Vector2Subtract(bottomVertices[0], topVertices[1])
		t2b1 := rl.Vector2Subtract(bottomVertices[3], topVertices[2])

		t2bLen := rl.Vector2Length(t2b0) // t2b0 and t2b1 has the same length

		partDrawn := float32(0)

		t2b0 = rl.Vector2Scale(t2b0, middlePartLength / t2bLen)
		t2b1 = rl.Vector2Scale(t2b1, middlePartLength / t2bLen)

		start0 := topVertices[1]
		start1 := topVertices[2]

		partCounter := 0

		for partDrawn + middlePartLength < t2bLen{
			end0 := rl.Vector2Add(start0, t2b0)
			end1 := rl.Vector2Add(start1, t2b1)

			if partCounter % 2 == 0{
				DrawTextureUvVertices(
					SustainTex, 
					middleUvs,
					[4]rl.Vector2{
						start0,
						end0,
						end1,
						start1,
					}, 
					color,
				)
			}else{
				DrawTextureUvVertices(
					SustainTex, 
					[4]rl.Vector2{
						middleUvs[1],
						middleUvs[0],
						middleUvs[3],
						middleUvs[2],
					},
					[4]rl.Vector2{
						start0,
						end0,
						end1,
						start1,
					}, 
					color,
				)
			}

			start0 = end0
			start1 = end1

			partDrawn += middlePartLength
			partCounter++
		}

		restOfMiddle := t2bLen - partDrawn

		middleEndUvHeight := (restOfMiddle / scale) / f32(SustainTex.Height)
		uvBegin := 1 - (marginNormalized + middleEndUvHeight)
		uvEnd := 1 - marginNormalized
		middleEndUvs := [4]rl.Vector2{
			rl.Vector2{0, uvBegin},
			rl.Vector2{0, uvEnd},
			rl.Vector2{1, uvEnd},
			rl.Vector2{1, uvBegin},
		}

		DrawTextureUvVertices(
			SustainTex, 
			middleEndUvs,
			[4]rl.Vector2{
				start0,
				bottomVertices[0],
				bottomVertices[3],
				start1,
			}, 
			color,
		)
	}

	DrawTextureVertices(
		SustainTex, bottomSrcRect, bottomVertices, color,
	)

	rl.EndBlendMode()
}
