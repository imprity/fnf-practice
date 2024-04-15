package main

import (
	"golang.org/x/exp/constraints"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func BoolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func IntToBool[N constraints.Integer](n N) bool {
	if n == 0 {
		return false
	} else {
		return true
	}
}

func AbsI[N constraints.Signed](n N) N {
	if n < 0 {
		return n * -1
	}
	return n
}

func SameSign[N constraints.Signed](n1, n2 N) bool {
	return (n1 < 0) == (n2 < 0)
}

func Clamp[N constraints.Integer | constraints.Float](n, minN, maxN N) N {
	n = min(n, maxN)
	n = max(n, minN)

	return n
}

// copied from https://www.febucci.com/2018/08/easing-functions/

func EaseIn[F constraints.Float](t F) F {
	return t * t
}

func EaseOut[F constraints.Float](t F) F {
	return 1.0 - (t-1.0)*(t-1.0)
}

func EaseInAndOut[F constraints.Float](t F) F {
	return Lerp(EaseIn(t), EaseOut(t), t)
}

func Lerp[F constraints.Float](a, b, t F) F {
	return a + (b-a)*t
}

type CircularQueue[T any] struct {
	End    int
	Start  int
	Length int
	Data   []T
}

func (q *CircularQueue[T]) IsFull() bool {
	return q.Length >= len(q.Data)
}

func (q *CircularQueue[T]) IsEmpty() bool {
	return q.Length <= 0
}

func (q *CircularQueue[T]) Enqueue(item T) {
	index := q.End

	isFull := q.IsFull()

	if isFull {
		q.Start += 1
		q.Start = q.Start % len(q.Data)
		q.End += 1
		q.End = q.End % len(q.Data)
	} else {
		q.End += 1
		q.End = q.End % len(q.Data)
		q.Length += 1
	}

	q.Data[index] = item
}

func (q *CircularQueue[T]) Dequeue() T {
	if q.Length <= 0 {
		panic("CircularQueue:Dequeue: Dequeue on empty queue")
	}

	q.Length -= 1

	q.Start %= len(q.Data)
	returnIndex := q.Start
	q.Start += 1

	return q.Data[returnIndex]
}

func (q *CircularQueue[T]) At(index int) T {
	return q.Data[(q.Start+index)%len(q.Data)]
}

func (q *CircularQueue[T]) Clear() {
	q.Length = 0
	q.Start = 0
	q.End = 0
}

type ReadChannel[T any] struct {
	RequestChannel chan bool
	DataChannel    chan T
}

func (rc ReadChannel[T]) RequestRead() {
	rc.RequestChannel <- true
}

func (rc ReadChannel[T]) Read() T {
	return <-rc.DataChannel
}

type ReadManyChannel[T any] struct {
	RequestChannel chan bool
	SizeChannel    chan int
	DataChannel    chan T
}

func (rm ReadManyChannel[T]) RequestRead() {
	rm.RequestChannel <- true
}

func (rm ReadManyChannel[T]) ReadSize() int {
	return <-rm.SizeChannel
}

func (rm ReadManyChannel[T]) Read() T {
	return <-rm.DataChannel
}

func IsClockWise(v1, v2, v3 rl.Vector2) bool {
	return (v2.X-v1.X)*(v3.Y-v1.Y)-(v2.Y-v1.Y)*(v3.X-v1.X) < 0
}

// TODO : support rotation and scaling
func DrawPatternBackground(
	texture rl.Texture2D,
	offsetX, offsetY float32,
	tint rl.Color,
) {
	/*
		0 -- 3
		|    |
		|    |
		1 -- 2
	*/

	if texture.ID > 0 {
		rl.SetTextureWrap(texture, rl.WrapRepeat)

		uvEndX := float32(SCREEN_WIDTH) / float32(texture.Width)
		uvEndY := float32(SCREEN_HEIGHT) / float32(texture.Height)

		if uvEndX < 0 {
			uvEndX = 0
		}

		if uvEndY < 0 {
			uvEndY = 0
		}

		uvs := [4]rl.Vector2{}

		uvs[0] = rl.Vector2{0, 0}
		uvs[1] = rl.Vector2{0, uvEndY}
		uvs[2] = rl.Vector2{uvEndX, uvEndY}
		uvs[3] = rl.Vector2{uvEndX, 0}

		for i := range len(uvs) {
			uvs[i].X += offsetX
			uvs[i].Y += offsetY
		}

		rl.SetTexture(texture.ID)
		rl.Begin(rl.Quads)

		rl.Color4ub(tint.R, tint.G, tint.B, tint.A)
		rl.Normal3f(0, 0, 1.0)

		rl.TexCoord2f(uvs[0].X, uvs[0].Y)
		rl.Vertex2f(0, 0)

		rl.TexCoord2f(uvs[1].X, uvs[1].Y)
		rl.Vertex2f(0, SCREEN_HEIGHT)

		rl.TexCoord2f(uvs[2].X, uvs[2].Y)
		rl.Vertex2f(SCREEN_WIDTH, SCREEN_HEIGHT)

		rl.TexCoord2f(uvs[3].X, uvs[3].Y)
		rl.Vertex2f(SCREEN_WIDTH, 0)

		rl.End()
		rl.SetTexture(0)
	}
}

func DrawTextureTransfromed(
	texture rl.Texture2D,
	srcRect rl.Rectangle,
	mat rl.Matrix,
	tint rl.Color,
) {
	/*
		0 -- 3
		|    |
		|    |
		1 -- 2
	*/

	if texture.ID > 0 {
		//normalize src rect
		texW := float32(texture.Width)
		texH := float32(texture.Height)

		uv0 := rl.Vector2{srcRect.X / texW, srcRect.Y / texH}
		uv1 := rl.Vector2{srcRect.X / texW, (srcRect.Y + srcRect.Height) / texH}
		uv2 := rl.Vector2{(srcRect.X + srcRect.Width) / texW, (srcRect.Y + srcRect.Height) / texH}
		uv3 := rl.Vector2{(srcRect.X + srcRect.Width) / texW, srcRect.Y / texH}

		v0 := rl.Vector2{0, 0}
		v1 := rl.Vector2{0, srcRect.Height}
		v2 := rl.Vector2{srcRect.Width, srcRect.Height}
		v3 := rl.Vector2{srcRect.Width, 0}

		v0 = rl.Vector2Transform(v0, mat)
		v1 = rl.Vector2Transform(v1, mat)
		v2 = rl.Vector2Transform(v2, mat)
		v3 = rl.Vector2Transform(v3, mat)

		rl.SetTexture(texture.ID)
		rl.Begin(rl.Quads)

		rl.Color4ub(tint.R, tint.G, tint.B, tint.A)
		rl.Normal3f(0, 0, 1.0)

		if IsClockWise(v0, v1, v2) {
			rl.TexCoord2f(uv0.X, uv0.Y)
			rl.Vertex2f(v0.X, v0.Y)

			rl.TexCoord2f(uv1.X, uv1.Y)
			rl.Vertex2f(v1.X, v1.Y)

			rl.TexCoord2f(uv2.X, uv2.Y)
			rl.Vertex2f(v2.X, v2.Y)

			rl.TexCoord2f(uv3.X, uv3.Y)
			rl.Vertex2f(v3.X, v3.Y)

		} else {
			rl.TexCoord2f(uv0.X, uv0.Y)
			rl.Vertex2f(v0.X, v0.Y)

			rl.TexCoord2f(uv3.X, uv3.Y)
			rl.Vertex2f(v3.X, v3.Y)

			rl.TexCoord2f(uv2.X, uv2.Y)
			rl.Vertex2f(v2.X, v2.Y)

			rl.TexCoord2f(uv1.X, uv1.Y)
			rl.Vertex2f(v1.X, v1.Y)
		}

		rl.End()
		rl.SetTexture(0)
	}
}
