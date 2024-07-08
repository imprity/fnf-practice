package main

import (
	"math"
	"os"
	"path/filepath"
	"time"

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

func Lerp[F constraints.Float](a, b, t F) F {
	return a + (b-a)*t
}

func ExecutablePath() (string, error) {
	path, err := os.Executable()

	if err != nil {
		return "", err
	}

	evaled, err := filepath.EvalSymlinks(path)

	if err != nil {
		return "", err
	}

	return evaled, nil
}

func RelativePath(path string) (string, error) {
	exePath, err := ExecutablePath()

	if err != nil {
		return "", err
	}

	joined := filepath.Join(filepath.Dir(exePath), path)

	return joined, nil
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

func (q *CircularQueue[T]) PeekFirst() T {
	return q.Data[q.Start%len(q.Data)]
}

func (q *CircularQueue[T]) PeekLast() T {
	return q.Data[(q.End-1)%len(q.Data)]
}

func (q *CircularQueue[T]) Clear() {
	q.Length = 0
	q.Start = 0
	q.End = 0
}

type Queue[T any] struct {
	Data []T
}

func (q *Queue[T]) Length() int {
	return len(q.Data)
}

func (q *Queue[T]) IsEmpty() bool {
	return len(q.Data) <= 0
}

func (q *Queue[T]) Enqueue(item T) {
	q.Data = append(q.Data, item)
}

func (q *Queue[T]) Dequeue() T {
	toReturn := q.Data[0]

	for i := 0; i+1 < len(q.Data); i++ {
		q.Data[i] = q.Data[i+1]
	}

	q.Data = q.Data[:len(q.Data)-1]

	return toReturn
}

func (q *Queue[T]) At(index int) T {
	return q.Data[index]
}

func (q *Queue[T]) Set(index int, item T) {
	q.Data[index] = item
}

func (q *Queue[T]) PeekFirst() T {
	return q.Data[0]
}

func (q *Queue[T]) PeekLast() T {
	return q.Data[len(q.Data)-1]
}

func (q *Queue[T]) Clear() {
	q.Data = q.Data[:0]
}

func IsClockWise(v1, v2, v3 rl.Vector2) bool {
	return (v2.X-v1.X)*(v3.Y-v1.Y)-(v2.Y-v1.Y)*(v3.X-v1.X) < 0
}

func RectWH[N constraints.Integer | constraints.Float](width, height N) rl.Rectangle {
	return rl.Rectangle{
		X: 0, Y: 0,
		Width: float32(width), Height: float32(height),
	}
}

func IsRectEmpty(r rl.Rectangle) bool {
	return r.Width <= 0 || r.Height <= 0
}

func RectUnion(r1, r2 rl.Rectangle) rl.Rectangle {
	minX := min(r1.X, r2.X)
	minY := min(r1.Y, r2.Y)

	maxX := max(r1.X+r1.Width, r2.X+r2.Width)
	maxY := max(r1.Y+r1.Height, r2.Y+r2.Height)

	return rl.Rectangle{
		X: minX, Y: minY,
		Width: maxX - minX, Height: maxY - minY,
	}
}

func RectIntersect(r1, r2 rl.Rectangle) rl.Rectangle {
	newR := rl.Rectangle{}

	newR.X = max(r1.X, r2.X)
	newR.Y = max(r1.Y, r2.Y)
	newR.Width = min(r1.X+r1.Width, r2.X+r2.Width) - newR.X
	newR.Height = min(r1.Y+r1.Height, r2.Y+r2.Height) - newR.Y

	if IsRectEmpty(newR) {
		newR.X += newR.Width * 0.5
		newR.Y += newR.Height * 0.5
		newR.Width, newR.Height = 0.0, 0.0
	}

	return newR
}

func RectCenter(rect rl.Rectangle) rl.Vector2 {
	return rl.Vector2{
		X: rect.X + rect.Width*0.5,
		Y: rect.Y + rect.Height*0.5,
	}
}

func RectPos(rect rl.Rectangle) rl.Vector2 {
	return rl.Vector2{
		X: rect.X,
		Y: rect.Y,
	}
}

func RectEnd(rect rl.Rectangle) rl.Vector2 {
	return rl.Vector2{
		X: rect.X + rect.Width,
		Y: rect.Y + rect.Height,
	}
}

func RectCentered(rect rl.Rectangle, x, y float32) rl.Rectangle {
	rect.X = x - rect.Width*0.5
	rect.Y = y - rect.Height*0.5

	return rect
}

func RectExpand(rect rl.Rectangle, amount float32) rl.Rectangle {
	rect.X -= amount
	rect.Y -= amount
	rect.Width += amount * 2
	rect.Height += amount * 2

	return rect
}

func RectExpandPro(rect rl.Rectangle, left, right, top, bottom float32) rl.Rectangle {
	rect.X -= left
	rect.Y -= top
	rect.Width += left + right
	rect.Height += top + bottom

	return rect
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

func DrawTextureUvVertices(
	texture rl.Texture2D,
	uvs [4]rl.Vector2,
	vertices [4]rl.Vector2,
	tint rl.Color,
) {
	if texture.ID > 0 {

		rl.SetTexture(texture.ID)
		rl.Begin(rl.Quads)

		rl.Color4ub(tint.R, tint.G, tint.B, tint.A)
		rl.Normal3f(0, 0, 1.0)

		if IsClockWise(vertices[0], vertices[1], vertices[2]) {
			rl.TexCoord2f(uvs[0].X, uvs[0].Y)
			rl.Vertex2f(vertices[0].X, vertices[0].Y)

			rl.TexCoord2f(uvs[1].X, uvs[1].Y)
			rl.Vertex2f(vertices[1].X, vertices[1].Y)

			rl.TexCoord2f(uvs[2].X, uvs[2].Y)
			rl.Vertex2f(vertices[2].X, vertices[2].Y)

			rl.TexCoord2f(uvs[3].X, uvs[3].Y)
			rl.Vertex2f(vertices[3].X, vertices[3].Y)

		} else {
			rl.TexCoord2f(uvs[0].X, uvs[0].Y)
			rl.Vertex2f(vertices[0].X, vertices[0].Y)

			rl.TexCoord2f(uvs[3].X, uvs[3].Y)
			rl.Vertex2f(vertices[3].X, vertices[3].Y)

			rl.TexCoord2f(uvs[2].X, uvs[2].Y)
			rl.Vertex2f(vertices[2].X, vertices[2].Y)

			rl.TexCoord2f(uvs[1].X, uvs[1].Y)
			rl.Vertex2f(vertices[1].X, vertices[1].Y)
		}

		rl.End()
		rl.SetTexture(0)
	}
}

// Draw a texture with vertices.
//
// Vertices should be laid out thus.
//
//	0 -- 3
//	|    |
//	|    |
//	1 -- 2
func DrawTextureVertices(
	texture rl.Texture2D,
	srcRect rl.Rectangle,
	vertices [4]rl.Vector2,
	tint rl.Color,
) {
	if texture.ID > 0 {
		texW := float32(texture.Width)
		texH := float32(texture.Height)

		uvs := [4]rl.Vector2{
			{srcRect.X / texW, srcRect.Y / texH},
			{srcRect.X / texW, (srcRect.Y + srcRect.Height) / texH},
			{(srcRect.X + srcRect.Width) / texW, (srcRect.Y + srcRect.Height) / texH},
			{(srcRect.X + srcRect.Width) / texW, srcRect.Y / texH},
		}

		DrawTextureUvVertices(
			texture,
			uvs,
			vertices,
			tint,
		)
	}
}

func DrawTextureTransfromed(
	texture rl.Texture2D,
	srcRect rl.Rectangle,
	mat rl.Matrix,
	tint rl.Color,
) {
	vertices := [4]rl.Vector2{
		{0, 0},
		{0, srcRect.Height},
		{srcRect.Width, srcRect.Height},
		{srcRect.Width, 0},
	}

	vertices[0] = rl.Vector2Transform(vertices[0], mat)
	vertices[1] = rl.Vector2Transform(vertices[1], mat)
	vertices[2] = rl.Vector2Transform(vertices[2], mat)
	vertices[3] = rl.Vector2Transform(vertices[3], mat)

	DrawTextureVertices(texture, srcRect, vertices, tint)
}

func drawRectangleRoundedCornersImpl(
	rec rl.Rectangle,
	roundness [4]float32, segements [4]int32,
	col rl.Color, fill bool, lineThick float32,
) {

	for i, r := range roundness {
		roundness[i] = Clamp(r, 0, 1)
	}

	radiusMax := float32(0)

	if rec.Width < rec.Height {
		radiusMax = rec.Width * 0.5
	} else {
		radiusMax = rec.Height * 0.5
	}

	if radiusMax <= 0 {
		return
	}

	rectCenter := rl.Vector2{
		X: (rec.X + rec.Width*0.5),
		Y: (rec.Y + rec.Height*0.5),
	}

	/*
		r0----r1
		|     |
		|     |
		r3----r2
	*/

	r0 := rl.Vector2{X: rec.X, Y: rec.Y}
	r1 := rl.Vector2{X: rec.X + rec.Width, Y: rec.Y}
	r2 := rl.Vector2{X: rec.X + rec.Width, Y: rec.Y + rec.Height}
	r3 := rl.Vector2{X: rec.X, Y: rec.Y + rec.Height}

	radius := [4]float32{
		radiusMax * roundness[0],
		radiusMax * roundness[1],
		radiusMax * roundness[2],
		radiusMax * roundness[3],
	}

	// ==================================
	// draw circle segements
	// ==================================

	// circleCenters
	ccs := [4]rl.Vector2{}

	// top left circle center
	ccs[0] = rl.Vector2{
		X: r0.X + radius[0],
		Y: r0.Y + radius[0],
	}

	// top right circle center
	ccs[1] = rl.Vector2{
		X: r1.X - radius[1],
		Y: r1.Y + radius[1],
	}

	// bottom right circle center
	ccs[2] = rl.Vector2{
		X: r2.X - radius[2],
		Y: r2.Y - radius[2],
	}

	// bottom left circle center
	ccs[3] = rl.Vector2{
		X: r3.X + radius[3],
		Y: r3.Y - radius[3],
	}

	cAngles := [5]float32{-180, -90, 0, 90, 180}

	for i := 0; i < 4; i++ {
		start := cAngles[i]
		end := cAngles[i+1]
		c := ccs[i]

		r := radius[i]

		if fill {
			rl.DrawCircleSector(c, r, start, end, segements[i], col)
		} else {
			rl.DrawRing(c,
				r-lineThick*0.5, r+lineThick*0.5,
				start, end,
				segements[i], col)
		}
	}

	// ==================================
	// draw the rest
	// ==================================

	/*
			sigh...
		          02 _________ 03
		            |         |_ 05
		        00__|01     04  |
		          |             |
		          |             |
		          |             |
		        11|____         |
		            10|   07____|06
		              |_____|
		            09      08
	*/

	ps := [14]rl.Vector2{}

	ps[12] = rl.Vector2{r0.X, r0.Y + radius[0]}             // 0
	ps[11] = rl.Vector2{r0.X + radius[0], r0.Y + radius[0]} // 1
	ps[10] = rl.Vector2{r0.X + radius[0], r0.Y}             // 2

	ps[9] = rl.Vector2{r1.X - radius[1], r1.Y}             // 3
	ps[8] = rl.Vector2{r1.X - radius[1], r1.Y + radius[1]} // 4
	ps[7] = rl.Vector2{r1.X, r1.Y + radius[1]}             // 5

	ps[6] = rl.Vector2{r2.X, r2.Y - radius[2]}             // 6
	ps[5] = rl.Vector2{r2.X - radius[2], r2.Y - radius[2]} // 7
	ps[4] = rl.Vector2{r2.X - radius[2], r2.Y}             // 8

	ps[3] = rl.Vector2{r3.X + radius[3], r3.Y}             // 9
	ps[2] = rl.Vector2{r3.X + radius[3], r3.Y - radius[3]} // 10
	ps[1] = rl.Vector2{r3.X, r3.Y - radius[3]}             // 11

	ps[13] = ps[1]

	ps[0] = rectCenter

	if fill {
		rl.DrawTriangleFan(ps[:], col)
	} else {
		// yes we do some unnecessary calculations but it will be too dirty
		// if I don't
		rl.DrawLineEx(ps[1], ps[12], lineThick, col) // 11 - 00
		rl.DrawLineEx(ps[10], ps[9], lineThick, col) // 02 - 03
		rl.DrawLineEx(ps[7], ps[6], lineThick, col)  // 05 - 06
		rl.DrawLineEx(ps[4], ps[3], lineThick, col)  // 08 - 09
	}
}

func DrawRectangleRoundedCorners(
	rec rl.Rectangle,
	roundness [4]float32, segements [4]int32,
	col rl.Color,
) {
	drawRectangleRoundedCornersImpl(
		rec, roundness, segements,
		col, true, 0,
	)
}

func DrawRectangleRoundedCornersLines(
	rec rl.Rectangle,
	roundness [4]float32, segements [4]int32,
	lineThick float32, col rl.Color,
) {
	drawRectangleRoundedCornersImpl(
		rec, roundness, segements,
		col, false, lineThick,
	)
}

// ==================
// music stuff
// ==================

func CalculateCrochet(bpm float64) time.Duration {
	return time.Duration(f64(time.Minute) / bpm)
}

func BeatsToTime(beats float64, bpm float64) time.Duration {
	return time.Duration(beats * f64(time.Minute) / bpm)
}

func StepsToTime(steps float64, bpm float64) time.Duration {
	return time.Duration(steps * f64(time.Minute) / bpm / 4)
}

// ==================
// easing funcitons
// ==================

// copy pasted from https://easings.net/

func EaseInOutCubic(x float64) float64 {
	if x < 0.5 {
		return 4 * x * x * x
	} else {
		return 1 - math.Pow(-2*x+2, 3)/2
	}
}

func EaseOutQuint[F constraints.Float](x F) F {
	return 1 - F(math.Pow(1-f64(x), 5))
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
