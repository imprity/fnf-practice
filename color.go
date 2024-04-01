package main

import (
	"image/color"
	"math"
)

// whether raylib's color is alpha premultiplied depends on context
// but raylib's go binding uses color.RGBA to pass it's color parameter
// so this type is here to make it clear that it only converted the color's range from
// 0 - 1 to 0 - 255
type RlColor = color.RGBA

type Color struct {
	R, G, B, A float64
}

func Col(r, g, b, a float64) Color {
	return Color{r, g, b, a}
}

func FromImagecolor(c color.Color) Color {
	r, g, b, a := c.RGBA()

	toReturn := Color{}

	toReturn.R = float64(r) / 0xFFFF
	toReturn.G = float64(g) / 0xFFFF
	toReturn.B = float64(b) / 0xFFFF
	toReturn.A = float64(a) / 0xFFFF

	return toReturn
}

func ToImageColor(c Color) color.Color {
	return color.NRGBA{
		uint8(math.Round(c.R * 255)),
		uint8(math.Round(c.G * 255)),
		uint8(math.Round(c.B * 255)),
		uint8(math.Round(c.A * 255)),
	}
}

func (c Color) ToImageColor() color.Color {
	return ToImageColor(c)
}

func ToImageRGBA(c Color) color.RGBA {
	multiplied := c.MultiplyAlpha()
	return color.RGBA{
		uint8(math.Round(multiplied.R * 0xFF)),
		uint8(math.Round(multiplied.G * 0xFF)),
		uint8(math.Round(multiplied.B * 0xFF)),
		uint8(math.Round(multiplied.A * 0xFF)),
	}
}

func (c Color) ToImageRGBA() color.RGBA {
	return ToImageRGBA(c)
}

func ToRlColor(c Color) RlColor{
	return color.RGBA{
		uint8(math.Round(multiplied.R * 0xFF)),
		uint8(math.Round(multiplied.G * 0xFF)),
		uint8(math.Round(multiplied.B * 0xFF)),
		uint8(math.Round(multiplied.A * 0xFF)),
	}
}

func (c Color)ToRlColor() RlColor{
	return ToRlColor(c)

func Color255(r, g, b, a uint8) Color {
	rf, gf, bf, af := float64(r), float64(g), float64(b), float64(a)
	return Color{
		rf / 255.0,
		gf / 255.0,
		bf / 255.0,
		af / 255.0,
	}
}

func (c Color) MultiplyAlpha() Color {
	return Color{
		c.R * c.A,
		c.G * c.A,
		c.B * c.A,
		c.A,
	}
}

func LerpRGB(c1, c2 Color, t float64) Color {
	return Color{
		Lerp(c1.R, c2.R, t),
		Lerp(c1.G, c2.G, t),
		Lerp(c1.B, c2.B, t),
		1.0,
	}
}

func LerpRGBA(c1, c2 Color, t float64) Color {
	return Color{
		Lerp(c1.R, c2.R, t),
		Lerp(c1.G, c2.G, t),
		Lerp(c1.B, c2.B, t),
		Lerp(c1.A, c2.A, t),
	}
}

// TODO : THIS IS FUCKING TERRIBLE
//
//	: Study more about colors to make it better!
func LerpHSV(c1, c2 Color, t float64) Color {
	hsv1 := ToHSV(c1)
	hsv2 := ToHSV(c2)

	if c1.R < 0.00001 && c1.G < 0.00001 && c1.B < 0.00001 && c1.A < 0.00001 {
		hsv1[0] = hsv2[0]
	} else if c1.R > 0.99999 && c1.G > 0.99999 && c1.B > 0.99999 && c1.A > 0.99999 {
		hsv1[0] = hsv2[0]
	}

	if c2.R < 0.00001 && c2.G < 0.00001 && c2.B < 0.00001 && c2.A < 0.00001 {
		hsv2[0] = hsv1[0]
	} else if c2.R > 0.99999 && c2.G > 0.99999 && c2.B > 0.99999 && c2.A > 0.99999 {
		hsv2[0] = hsv1[0]
	}

	var h, d, th float64

	d = hsv2[0] - hsv1[0]
	th = t

	if hsv1[0] > hsv2[0] {
		hsv1[0], hsv2[0] = hsv2[0], hsv1[0]

		d = -d
		th = 1 - t
	}

	if d > 180 {
		hsv1[0] += 360
		h = math.Mod(hsv1[0]+th*(hsv2[0]-hsv1[0]), 360)
	} else {
		h = hsv1[0] + th*d
	}

	return FromHSV(
		[]float64{
			h,
			Lerp(hsv1[1], hsv2[1], t),
			Lerp(hsv1[2], hsv2[2], t),
		},
	)
}

func LerpHSVA(c1, c2 Color, t float64) Color {
	c3 := LerpHSV(c1, c2, t)
	c3.A = Lerp(c1.A, c2.A, t)
	return c3
}

func LerpOkLab(c1, c2 Color, t float64) Color {
	lab1 := ToOkLab(c1)
	lab2 := ToOkLab(c2)
	lab3 := []float64{0, 0, 0}

	for i := 0; i < 3; i++ {
		lab3[i] = Lerp(lab1[i], lab2[i], t)
	}

	return FromOkLab(lab3)
}

func LerpOkLabA(c1, c2 Color, t float64) Color {
	c3 := LerpOkLab(c1, c2, t)
	c3.A = Lerp(c1.A, c2.A, t)
	return c3
}

// retured hsv range is
//
// h : 0 - 360
// s : 0 - 100
// v : 0 - 100
func ToHSV(color Color) []float64 {
	r, g, b, _ := color.R, color.G, color.B, color.A

	cMax := max(r, g, b)
	cMin := min(r, g, b)
	delta := cMax - cMin

	var h, s, v float64 = 0, 0, 0

	if cMax == cMin {
		h = 0
	} else if cMax == r {
		h = math.Mod((60.0*((g-b)/delta) + 360.0), 360.0)
	} else if cMax == g {
		h = math.Mod((60.0*((b-r)/delta) + 120.0), 360.0)
	} else {
		h = math.Mod((60.0*((r-g)/delta) + 240.0), 360.0)
	}

	if cMax == 0 {
		s = 0
	} else {
		s = (delta / cMax) * 100.0
	}

	v = cMax * 100.0

	return []float64{h, s, v}
}

func FromHSV(hsv []float64) Color {
	if len(hsv) != 3 {
		panic("hsv array should have 3 elements")
	}

	h := hsv[0]
	s := hsv[1] / 100.0
	v := hsv[2] / 100.0

	c := v * s
	x := c * (1.0 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := v - c

	var rt, gt, bt float64

	if h <= 60 {
		rt, gt, bt = c, x, 0
	} else if h <= 120 {
		rt, gt, bt = x, c, 0
	} else if h <= 180 {
		rt, gt, bt = 0, c, x
	} else if h <= 240 {
		rt, gt, bt = 0, x, c
	} else if h <= 300 {
		rt, gt, bt = x, 0, c
	} else {
		rt, gt, bt = c, 0, x
	}

	r, g, b := rt+m, gt+m, bt+m

	return Color{r, g, b, 1.0}
}

// Copy pasted from https://bottosson.github.io/posts/oklab/
func ToOkLab(c Color) []float64 {
	l := 0.4122214708*c.R + 0.5363325363*c.G + 0.0514459929*c.B
	m := 0.2119034982*c.R + 0.6806995451*c.G + 0.1073969566*c.B
	s := 0.0883024619*c.R + 0.2817188376*c.G + 0.6299787005*c.B

	l_ := math.Cbrt(l)
	m_ := math.Cbrt(m)
	s_ := math.Cbrt(s)

	return []float64{
		0.2104542553*l_ + 0.7936177850*m_ - 0.0040720468*s_,
		1.9779984951*l_ - 2.4285922050*m_ + 0.4505937099*s_,
		0.0259040371*l_ + 0.7827717662*m_ - 0.8086757660*s_,
	}
}

// Copy pasted from https://bottosson.github.io/posts/oklab/
func FromOkLab(lab []float64) Color {
	if len(lab) != 3 {
		panic("lab array should have 3 elements")
	}

	l_ := lab[0] + 0.3963377774*lab[1] + 0.2158037573*lab[2]
	m_ := lab[0] - 0.1055613458*lab[1] - 0.0638541728*lab[2]
	s_ := lab[0] - 0.0894841775*lab[1] - 1.2914855480*lab[2]

	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_

	c3 := Color{
		+4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s,
		1,
	}

	c3.R = Clamp(c3.R, 0, 1)
	c3.G = Clamp(c3.G, 0, 1)
	c3.B = Clamp(c3.B, 0, 1)

	return c3
}
