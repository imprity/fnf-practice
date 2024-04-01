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

func Lerp[F constraints.Float](a, b, t F) F{
	return a + (b-a) * t
}

type CircularQueue[T any] struct {
	Length int
	Start  int
	Data   []T
}

func (q *CircularQueue[T]) IsFull() bool {
	return q.Length >= len(q.Data)
}

func (q *CircularQueue[T]) IsEmpty() bool {
	return q.Length <= 0
}

func (q *CircularQueue[T]) Enqueue(item T) {
	isFull := q.IsFull()
	if isFull {
		q.Start += 1
		q.Start = q.Start % q.Length
	} else {
		q.Length += 1
	}

	index := (q.Start + q.Length - 1) % len(q.Data)
	q.Data[index] = item
}

func (q *CircularQueue[T]) Dequeue() T {
	if q.Length <= 0 {
		panic("CircularQueue:Dequeue: Dequeue on empty queue")
	}

	q.Start %= len(q.Data)
	returnIndex := q.Start
	q.Start += 1

	return q.Data[returnIndex]
}

func (q *CircularQueue[T]) At(index int) T {
	return q.Data[(q.Start+index)%q.Length]
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

func IsClockWise(v1, v2, v3 rl.Vector2) bool{
	return (v2.X - v1.X) * (v3.Y - v1.Y) - (v2.Y - v1.Y) * (v3.X - v1.X) < 0
}

func DrawTextureTransfromed(
	texture rl.Texture2D,
	mat rl.Matrix,
	tint Color,
){
	/*
	0 -- 3
	|    |
	|    |
	1 -- 2
	*/
	if texture.ID > 0{
		v0 := rl.Vector2{0,                      0}
		v1 := rl.Vector2{0,                      float32(texture.Height)}
		v2 := rl.Vector2{float32(texture.Width), float32(texture.Height)}
		v3 := rl.Vector2{float32(texture.Width), 0}

		v0 = rl.Vector2Transform(v0, mat)
		v1 = rl.Vector2Transform(v1, mat)
		v2 = rl.Vector2Transform(v2, mat)
		v3 = rl.Vector2Transform(v3, mat)


		c := tint.ToImageRGBA()
		rl.SetTexture(texture.ID)
		rl.Begin(rl.Quads)

		rl.Color4ub(c.R, c.G, c.B, c.A)
		rl.Normal3f(0,0, 1.0)

		if IsClockWise(v0, v1, v2){
			rl.TexCoord2f(0,0)
			rl.Vertex2f(v0.X, v0.Y)

			rl.TexCoord2f(0,1)
			rl.Vertex2f(v1.X, v1.Y)

			rl.TexCoord2f(1,1)
			rl.Vertex2f(v2.X, v2.Y)

			rl.TexCoord2f(1,0)
			rl.Vertex2f(v3.X, v3.Y)

		}else {
			rl.TexCoord2f(0,0)
			rl.Vertex2f(v0.X, v0.Y)

			rl.TexCoord2f(1,0)
			rl.Vertex2f(v3.X, v3.Y)

			rl.TexCoord2f(1,1)
			rl.Vertex2f(v2.X, v2.Y)

			rl.TexCoord2f(0,1)
			rl.Vertex2f(v1.X, v1.Y)
		}

		rl.End()
		rl.SetTexture(0)
	}
}
