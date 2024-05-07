package unitext

import (
	meta "github.com/go-text/typesetting/opentype/api/metadata"
	"github.com/go-text/typesetting/fontscan"
)

// this is just a copy paste of constants from typesetting packages
// I thought it would be nicer if it wasn't scattered all over the place

// Generic families as defined by
// https://www.w3.org/TR/css-fonts-4/#generic-font-families
const (
	FontFamilyFantasy   = fontscan.Fantasy
	FontFamilyMath      = fontscan.Math
	FontFamilyEmoji     = fontscan.Emoji
	FontFamilySerif     = fontscan.Serif
	FontFamilySansSerif = fontscan.SansSerif
	FontFamilyCursive   = fontscan.Cursive
	FontFamilyMonospace = fontscan.Monospace
)

// Style (also called slant) allows italic or oblique faces to be selected.
type FontStyle   = meta.Style

// Weight is the degree of blackness or stroke thickness of a font. This value
// ranges from 100.0 to 900.0, with 400.0 as normal.
type FontWeight  = meta.Weight

// Stretch is the width of a font as an approximate fraction of the normal
// width. Widths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width.
type FontStretch = meta.Stretch

type DesiredFont struct{
	Families []string

	Style   FontStyle
	Weight  FontWeight
	Stretch FontStretch
}

func MakeDesiredFont() DesiredFont{
	df := DesiredFont{}
	df.Style = StyleNormal
	df.Weight = WeightNormal
	df.Stretch = StretchNormal

	return df
}

const (
	// A face that is neither italic not obliqued.
	StyleNormal FontStyle = meta.StyleNormal
	// A form that is generally cursive in nature or slanted.
	// This groups what is usually called Italic or Oblique.
	StyleItalic FontStyle = meta.StyleItalic
)

const (
	// Thin weight (100), the thinnest value.
	WeightThin FontWeight = meta.WeightThin
	// Extra light weight (200).
	WeightExtraLight FontWeight = meta.WeightExtraLight
	// Light weight (300).
	WeightLight FontWeight = meta.WeightLight
	// Normal (400).
	WeightNormal FontWeight = meta.WeightNormal
	// Medium weight (500, higher than normal).
	WeightMedium FontWeight = meta.WeightMedium
	// Semibold weight (600).
	WeightSemibold FontWeight = meta.WeightSemibold
	// Bold weight (700).
	WeightBold FontWeight = meta.WeightBold
	// Extra-bold weight (800).
	WeightExtraBold FontWeight = meta.WeightExtraBold
	// Black weight (900), the thickest value.
	WeightBlack FontWeight = meta.WeightBlack
)

const (
	// Ultra-condensed width (50%), the narrowest possible.
	StretchUltraCondensed FontStretch = meta.StretchUltraCondensed
	// Extra-condensed width (62.5%).
	StretchExtraCondensed FontStretch = meta.StretchExtraCondensed
	// Condensed width (75%).
	StretchCondensed FontStretch = meta.StretchCondensed
	// Semi-condensed width (87.5%).
	StretchSemiCondensed FontStretch = meta.StretchSemiCondensed
	// Normal width (100%).
	StretchNormal FontStretch = meta.StretchNormal
	// Semi-expanded width (112.5%).
	StretchSemiExpanded FontStretch = meta.StretchSemiExpanded
	// Expanded width (125%).
	StretchExpanded FontStretch = meta.StretchExpanded
	// Extra-expanded width (150%).
	StretchExtraExpanded FontStretch = meta.StretchExtraExpanded
	// Ultra-expanded width (200%), the widest possible.
	StretchUltraExpanded FontStretch = meta.StretchUltraExpanded
)

