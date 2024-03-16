package kitty

import (
	"golang.org/x/exp/constraints"

	"image"
	"math"
)

// ============================================================
// Vec2
// ============================================================

type Vec2 struct {
	X, Y float64
}

func V(x, y float64) Vec2 { return Vec2{x, y} }

func (v Vec2) Ints() (int, int) { return int(v.X), int(v.Y) }
func IntsToV(x, y int) Vec2     { return Vec2{X: float64(x), Y: float64(y)} }

func FromPt(pt image.Point) Vec2 { return IntsToV(pt.X, pt.Y) }
func (v Vec2) ToPt() image.Point { return image.Pt(int(v.X), int(v.Y)) }

func Dot(ihs Vec2, rhs Vec2) float64  { return ihs.X*rhs.X + ihs.Y*rhs.Y }
func (v Vec2) Dot(other Vec2) float64 { return v.X*other.X + v.Y*other.Y }

func Cross(ihs Vec2, rhs Vec2) float64  { return ihs.X*rhs.Y - ihs.Y*rhs.X }
func (v Vec2) Cross(other Vec2) float64 { return v.X*other.Y - v.Y*other.X }

func (v Vec2) Equals(other Vec2) bool { return v.X == other.X && v.Y == other.Y }

func (v Vec2) Length() float64  { return math.Sqrt(v.X*v.X + v.Y*v.Y) }
func (v Vec2) Length2() float64 { return v.X*v.X + v.Y*v.Y }

func (v Vec2) AddV(other Vec2) Vec2           { return Vec2{X: v.X + other.X, Y: v.Y + other.Y} }
func (v Vec2) Add1(s float64) Vec2            { return Vec2{X: v.X + s, Y: v.Y + s} }
func (v Vec2) Add2(x float64, y float64) Vec2 { return Vec2{X: v.X + x, Y: v.Y + y} }

func (v Vec2) SubV(other Vec2) Vec2           { return Vec2{X: v.X - other.X, Y: v.Y - other.Y} }
func (v Vec2) Sub1(s float64) Vec2            { return Vec2{X: v.X - s, Y: v.Y - s} }
func (v Vec2) Sub2(x float64, y float64) Vec2 { return Vec2{X: v.X - x, Y: v.Y - y} }

func (v Vec2) MulV(other Vec2) Vec2           { return Vec2{X: v.X * other.X, Y: v.Y * other.Y} }
func (v Vec2) Mul1(s float64) Vec2            { return Vec2{X: v.X * s, Y: v.Y * s} }
func (v Vec2) Mul2(x float64, y float64) Vec2 { return Vec2{X: v.X * x, Y: v.Y * y} }

func (v Vec2) DivV(other Vec2) Vec2           { return Vec2{X: v.X / other.X, Y: v.Y / other.Y} }
func (v Vec2) Div1(s float64) Vec2            { return Vec2{X: v.X / s, Y: v.Y / s} }
func (v Vec2) Div2(x float64, y float64) Vec2 { return Vec2{X: v.X / x, Y: v.Y / y} }

func LerpV(a Vec2, b Vec2, t float64) Vec2 {
	return Vec2{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
	}
}

func (v Vec2) Lerp(other Vec2, t float64) Vec2 {
	return Vec2{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
	}
}

func Distance(a Vec2, b Vec2) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func Distance2(a Vec2, b Vec2) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

func (v Vec2) Distance(other Vec2) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (v Vec2) Distance2(other Vec2) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return dx*dx + dy*dy
}

func Reflect(ihs Vec2, rhs Vec2) Vec2 {
	factor := -2.0 * Dot(ihs, rhs)
	return Vec2{
		X: factor*ihs.X + rhs.X,
		Y: factor*ihs.Y + rhs.Y,
	}
}

func (v Vec2) Reflect(other Vec2) Vec2 {
	factor := -2.0 * v.Dot(other)
	return Vec2{
		X: factor*v.X + other.X,
		Y: factor*v.Y + other.Y,
	}
}

func (v Vec2) Normalized() Vec2 {
	m := v.Length()

	if m > 0.00001 {
		return v.Div1(m)
	} else {
		return v
	}
}

func (v Vec2) LimitLength(l float64) Vec2 {
	if v.Length2() < l*l {
		return v
	}
	return v.Normalized().Mul1(l)
}

func (v Vec2) Rotate(theta float64) Vec2 {
	return Vec2{
		v.X*math.Cos(theta) - v.Y*math.Sin(theta),
		v.X*math.Sin(theta) + v.Y*math.Cos(theta),
	}
}

// ============================================================
// Rect
// ============================================================

type FRect struct {
	X, Y, W, H float64
}

func Fr(x, y, w, h float64) FRect { return FRect{x, y, w, h} }

func (r FRect) Ints() (int, int, int, int) { return int(r.X), int(r.Y), int(r.W), int(r.H) }
func IntsToFr(x, y, w, h int) FRect        { return FRect{float64(x), float64(y), float64(w), float64(h)} }

func FromIRect(ir image.Rectangle) FRect {
	canon := ir.Canon()
	return IntsToFr(canon.Min.X, canon.Min.Y, canon.Dx(), canon.Dy())
}

func (rect FRect) ToIRect() image.Rectangle {
	return image.Rect(
		int(rect.X), int(rect.Y), int(rect.X+rect.W), int(rect.Y+rect.H),
	)
}

func (rect FRect) ContainsV(v Vec2) bool {
	return (v.X >= rect.X && v.X <= rect.X+rect.W &&
		v.Y >= rect.Y && v.Y <= rect.Y+rect.H)
}

func (rect FRect) ContainsXY(x, y float64) bool {
	return (x >= rect.X && x <= rect.X+rect.W &&
		y >= rect.Y && y <= rect.Y+rect.H)
}

func (rect FRect) Center() Vec2 {
	return Vec2{
		X: rect.X + rect.W*0.5,
		Y: rect.Y + rect.H*0.5,
	}
}

func (rect FRect) CenteredAt(v Vec2) FRect {
	var m Vec2 = v.SubV(rect.Center())
	return FRect{
		rect.X + m.X, rect.Y + m.Y, rect.W, rect.H,
	}
}

func (rect FRect) Empty() bool {
	return rect.W <= 0 || rect.H <= 0
}

func (rect FRect) Add(v Vec2) FRect {
	return Fr(rect.X+v.X, rect.Y+v.Y, rect.W, rect.H)
}

func (rect FRect) Sub(v Vec2) FRect {
	return Fr(rect.X-v.X, rect.Y-v.Y, rect.W, rect.H)
}

func (rect FRect) In(other FRect) bool {
	return rect.X >= other.X &&
		rect.Y >= other.Y &&
		rect.X+rect.W <= other.X+other.W &&
		rect.Y+rect.H <= other.Y+other.H
}

func (rect FRect) Inset(n float64) FRect {
	newR := FRect{}

	if rect.W < n*2 {
		newR.X = rect.X + rect.W*0.5
		newR.W = 0.0
	} else {
		newR.X = rect.X + n
		newR.W = rect.W - n*2
	}

	if rect.H < n*2 {
		newR.Y = rect.Y + rect.H*0.5
		newR.H = 0.0
	} else {
		newR.Y = rect.Y + n
		newR.H = rect.H - n*2
	}

	return newR
}

func (rect FRect) Intersect(other FRect) FRect {
	newR := FRect{}

	newR.X = max(rect.X, other.X)
	newR.Y = max(rect.Y, other.Y)
	newR.W = min(rect.X+rect.W, other.X+other.W) - newR.X
	newR.H = min(rect.Y+rect.H, other.Y+other.H) - newR.Y

	if newR.Empty() {
		newR.X += newR.W * 0.5
		newR.Y += newR.H * 0.5
		newR.W, newR.H = 0.0, 0.0
	}

	return newR
}

func (rect FRect) Union(other FRect) FRect {
	minX := min(rect.X, other.X)
	minY := min(rect.Y, other.Y)

	maxX := max(rect.X+rect.W, other.X+other.W)
	maxY := max(rect.Y+rect.H, other.Y+other.H)

	return Fr(minX, minY, maxX-minX, maxY-minY)
}

func (rect FRect) Overlaps(other FRect) bool {
	intersect := rect.Intersect(other)
	return !intersect.Empty()
}

// ============================================================
// Circle
// ============================================================

type Circle struct {
	X, Y float64
	R    float64
}

func C(x, y, r float64) Circle { return Circle{x, y, r} }

func (c Circle) GetV() Vec2 { return V(c.X, c.Y) }
func (c Circle) At(v Vec2) Circle {
	c.X, c.Y = v.X, v.Y
	return c
}

func (c Circle) ContainsV(v Vec2) bool {
	return c.GetV().SubV(v).Length2() < c.R*c.R
}

func (c Circle) ContainsXY(x, y float64) bool {
	return c.GetV().Sub2(x, y).Length2() < c.R*c.R
}

// ============================================================
// Collsions
// ============================================================

func CollsionCircleCircle(c1, c2 Circle) bool {
	v := c1.GetV().SubV(c2.GetV())
	r := c1.R + c2.R

	return v.Length2() < r*r
}

// copy pasted from https://vband3d.tripod.com/visualbasic/tut_mixedcollisions.htm
func CollisionRectCircle(r FRect, c Circle) bool {
	// temporary variables to set edges for testing
	var testX float64 = c.X
	var testY float64 = c.Y

	// which edge is closest?
	if c.X < r.X {
		testX = r.X // left edge
	} else if c.X > r.X+r.W {
		testX = r.X + r.W // right edge
	}

	if c.Y < r.Y {
		testY = r.Y // top edge
	} else if c.Y > r.Y+r.H {
		testY = r.Y + r.H // bottom edge
	}

	// get distance from closest edges
	distX := c.X - testX
	distY := c.Y - testY
	distance := math.Sqrt((distX * distX) + (distY * distY))

	// if the distance is less than the radius, collision!
	return distance <= c.R
}

// ============================================================
// Other Stuffs
// ============================================================

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func Clamp[N constraints.Integer | constraints.Float](n, minN, maxN N) N {
	n = min(n, maxN)
	n = max(n, minN)

	return n
}

func GetClosestPointToSegment(from, to, point Vec2) Vec2 {
	len2 := from.Distance2(to)

	if math.Abs(len2) < 0.00001 {
		return from
	}

	t := point.SubV(from).Dot(to.SubV(from)) / len2

	t = min(t, 1)
	t = max(t, 0)

	projection := from.AddV(to.SubV(from).Mul1(t)) // Projection falls on the segment
	return projection
}

// copy pasted from https://jordano-jackson.tistory.com/27
func SegmentIntersects(from1, to1, from2, to2 Vec2) bool {
	dir := func(a, b, c Vec2) float64 {
		ca := a.SubV(c)
		cb := b.SubV(c)

		return ca.Cross(cb)
	}

	return dir(from1, to1, from2)*dir(from1, to1, to2) < 0 && dir(from2, to2, from1)*dir(from2, to2, to1) < 0
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
