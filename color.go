package fnf

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
)

type FnfColor rl.Color

var FnfWhite = FnfColor{255, 255, 255, 255}

func Col(r, g, b, a uint8) FnfColor {
	return FnfColor{r, g, b, a}
}

func Col01(r, g, b, a float32) FnfColor {
	return FnfColor(rl.ColorFromNormalized(rl.Vector4{r, g, b, a}))
}

func Col01Vec4(v rl.Vector4) FnfColor {
	return FnfColor(rl.ColorFromNormalized(v))
}

func ColFromHSV(h, s, v float32) FnfColor {
	return FnfColor(rl.ColorFromHSV(h, s, v))
}

func ColorNormalize(c FnfColor) rl.Vector4 {
	return rl.Vector4{f32(c.R) / 255, f32(c.G) / 255, f32(c.B) / 255, f32(c.A) / 255}
}

func ColToHSV(c FnfColor) rl.Vector3 {
	return rl.Vector3(rl.ColorToHSV(rl.Color(c)))
}

func LerpRGB(c1, c2 FnfColor, t float64) FnfColor {
	return FnfColor{
		uint8(Lerp(f64(c1.R), f64(c2.R), t)),
		uint8(Lerp(f64(c1.G), f64(c2.G), t)),
		uint8(Lerp(f64(c1.B), f64(c2.B), t)),
		255,
	}
}

func LerpRGBA(c1, c2 FnfColor, t float64) FnfColor {
	return FnfColor{
		uint8(Lerp(f64(c1.R), f64(c2.R), t)),
		uint8(Lerp(f64(c1.G), f64(c2.G), t)),
		uint8(Lerp(f64(c1.B), f64(c2.B), t)),
		uint8(Lerp(f64(c1.A), f64(c2.A), t)),
	}
}

func ToRlColor(c FnfColor) rl.Color {
	norm := rl.ColorNormalize(rl.Color(c))
	return rl.Color{
		uint8(norm.X * norm.W * 255),
		uint8(norm.Y * norm.W * 255),
		uint8(norm.Z * norm.W * 255),
		uint8(norm.W * 255),
	}
}

func ToRlColorNoPremultiply(c FnfColor) rl.Color {
	return rl.Color(c)
}

func TintColor(target, tint FnfColor) FnfColor {
	targetV := ColorNormalize(target)
	tintV := ColorNormalize(tint)
	return Col01(
		targetV.X*tintV.X,
		targetV.Y*tintV.Y,
		targetV.Z*tintV.Z,
		targetV.W*tintV.W,
	)
}

// Copy pasted from https://bottosson.github.io/posts/oklab/
func ColToOkLab(c FnfColor) rl.Vector3 {
	norm := rl.ColorNormalize(rl.Color(c))
	l := 0.4122214708*norm.X + 0.5363325363*norm.Y + 0.0514459929*norm.Z
	m := 0.2119034982*norm.X + 0.6806995451*norm.Y + 0.1073969566*norm.Z
	s := 0.0883024619*norm.X + 0.2817188376*norm.Y + 0.6299787005*norm.Z

	l_ := f32(math.Cbrt(f64(l)))
	m_ := f32(math.Cbrt(f64(m)))
	s_ := f32(math.Cbrt(f64(s)))

	return rl.Vector3{
		0.2104542553*l_ + 0.7936177850*m_ - 0.0040720468*s_,
		1.9779984951*l_ - 2.4285922050*m_ + 0.4505937099*s_,
		0.0259040371*l_ + 0.7827717662*m_ - 0.8086757660*s_,
	}
}

// Copy pasted from https://bottosson.github.io/posts/oklab/
func ColFromOkLab(lab rl.Vector3) FnfColor {
	l_ := lab.X + 0.3963377774*lab.Y + 0.2158037573*lab.Z
	m_ := lab.X - 0.1055613458*lab.Y - 0.0638541728*lab.Z
	s_ := lab.X - 0.0894841775*lab.Y - 1.2914855480*lab.Z

	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_

	cv := rl.Vector4{
		+4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s,
		1,
	}

	cv.X = Clamp(cv.X, 0, 1)
	cv.Y = Clamp(cv.Y, 0, 1)
	cv.Z = Clamp(cv.Z, 0, 1)

	return Col01Vec4(cv)
}

func FnfEndBlendMode() {
	rl.SetBlendMode(i32(rl.BlendAlphaPremultiply))
}
