// Package style handles CSS style resolution for pdfml.
package style

import (
	"fmt"
	"strconv"
	"strings"
)

// Unit represents a CSS length unit.
type Unit int

const (
	UnitPt  Unit = iota // points (1/72 inch) — canonical internal unit
	UnitPx              // pixels (converted via DPI)
	UnitMm              // millimetres
	UnitCm              // centimetres
	UnitIn              // inches
	UnitEm              // relative to current font-size
	UnitRem             // relative to root font-size
	UnitPct             // percentage
	UnitAuto            // auto
)

// Length is a CSS length value with magnitude and unit.
type Length struct {
	Value float64
	Unit  Unit
}

// Auto is the CSS 'auto' length.
var Auto = Length{Unit: UnitAuto}

// Pt returns a length in points.
func Pt(v float64) Length { return Length{Value: v, Unit: UnitPt} }

// Mm returns a length in millimetres.
func Mm(v float64) Length { return Length{Value: v, Unit: UnitMm} }

// Pct returns a percentage length.
func Pct(v float64) Length { return Length{Value: v, Unit: UnitPct} }

// IsAuto returns true if the length is 'auto'.
func (l Length) IsAuto() bool { return l.Unit == UnitAuto }

// ToPoints converts a Length to points given a context for relative units.
// parentPt is the parent dimension in points (for %).
// fontSizePt is the current element's font-size in points (for em).
// rootFontSizePt is the root font-size in points (for rem).
// dpi is the screen DPI for px conversion.
func (l Length) ToPoints(parentPt, fontSizePt, rootFontSizePt, dpi float64) float64 {
	switch l.Unit {
	case UnitPt:
		return l.Value
	case UnitPx:
		// 1px = 72/dpi pt
		return l.Value * 72.0 / dpi
	case UnitMm:
		return l.Value * 72.0 / 25.4
	case UnitCm:
		return l.Value * 72.0 / 2.54
	case UnitIn:
		return l.Value * 72.0
	case UnitEm:
		return l.Value * fontSizePt
	case UnitRem:
		return l.Value * rootFontSizePt
	case UnitPct:
		return l.Value / 100.0 * parentPt
	case UnitAuto:
		return 0
	}
	return l.Value
}

// ParseLength parses a CSS length string into a Length.
// Returns an error if the value cannot be parsed.
func ParseLength(s string) (Length, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Length{}, fmt.Errorf("style: empty length value")
	}
	if strings.EqualFold(s, "auto") {
		return Auto, nil
	}
	if strings.EqualFold(s, "normal") {
		return Length{Value: 1.2, Unit: UnitEm}, nil
	}

	// Try percentage
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil {
			return Length{}, fmt.Errorf("style: invalid percentage %q", s)
		}
		return Length{Value: v, Unit: UnitPct}, nil
	}

	// Try known units
	units := []struct {
		suffix string
		unit   Unit
	}{
		{"pt", UnitPt},
		{"px", UnitPx},
		{"mm", UnitMm},
		{"cm", UnitCm},
		{"in", UnitIn},
		{"em", UnitEm},
		{"rem", UnitRem},
	}

	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			numStr := strings.TrimSuffix(s, u.suffix)
			v, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return Length{}, fmt.Errorf("style: invalid length %q", s)
			}
			return Length{Value: v, Unit: u.unit}, nil
		}
	}

	// Bare number — treat as pt
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Length{}, fmt.Errorf("style: cannot parse length %q", s)
	}
	return Length{Value: v, Unit: UnitPt}, nil
}

// MustParseLength parses a length or returns a zero-pt on error.
func MustParseLength(s string) Length {
	l, err := ParseLength(s)
	if err != nil {
		return Pt(0)
	}
	return l
}

// ParseColor parses a CSS color string into an RGBA value (0-255).
// Supports: #rgb, #rrggbb, named colors (limited set), rgb(), rgba()
func ParseColor(s string) (Color, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Color{}, fmt.Errorf("style: empty color value")
	}

	if strings.HasPrefix(s, "#") {
		return parseHexColor(s)
	}

	if strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba(") {
		return parseRGBColor(s)
	}

	// Named colors
	if c, ok := namedColors[strings.ToLower(s)]; ok {
		return c, nil
	}

	return Color{}, fmt.Errorf("style: unknown color %q", s)
}

// Color holds RGBA components (0-255 for R,G,B; 0-255 for A where 255=opaque).
type Color struct {
	R, G, B uint8
	A       uint8 // 255 = fully opaque
}

// IsZero returns true for an unset/zero color.
func (c Color) IsZero() bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// WithAlpha returns a copy of the color with the given alpha (0-255).
func (c Color) WithAlpha(a uint8) Color {
	c.A = a
	return c
}

func parseHexColor(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	var r, g, b uint8
	switch len(s) {
	case 3:
		// #rgb → #rrggbb
		rv, err := strconv.ParseUint(string([]byte{s[0], s[0]}), 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("style: invalid hex color #%s", s)
		}
		gv, _ := strconv.ParseUint(string([]byte{s[1], s[1]}), 16, 8)
		bv, _ := strconv.ParseUint(string([]byte{s[2], s[2]}), 16, 8)
		r, g, b = uint8(rv), uint8(gv), uint8(bv)
	case 6:
		rv, err := strconv.ParseUint(s[0:2], 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("style: invalid hex color #%s", s)
		}
		gv, _ := strconv.ParseUint(s[2:4], 16, 8)
		bv, _ := strconv.ParseUint(s[4:6], 16, 8)
		r, g, b = uint8(rv), uint8(gv), uint8(bv)
	default:
		return Color{}, fmt.Errorf("style: invalid hex color #%s", s)
	}
	return Color{R: r, G: g, B: b, A: 255}, nil
}

func parseRGBColor(s string) (Color, error) {
	s = strings.TrimSpace(s)
	isRGBA := strings.HasPrefix(s, "rgba(")
	inner := strings.TrimPrefix(s, "rgb(")
	inner = strings.TrimPrefix(inner, "rgba(")
	inner = strings.TrimSuffix(inner, ")")

	parts := strings.Split(inner, ",")
	if len(parts) < 3 {
		return Color{}, fmt.Errorf("style: invalid rgb() color %q", s)
	}

	parse := func(p string) (uint8, error) {
		p = strings.TrimSpace(p)
		if strings.HasSuffix(p, "%") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
			if err != nil {
				return 0, err
			}
			return uint8(v / 100.0 * 255), nil
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return 0, err
		}
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v), nil
	}

	r, err := parse(parts[0])
	if err != nil {
		return Color{}, fmt.Errorf("style: invalid rgb() color %q", s)
	}
	g, _ := parse(parts[1])
	b, _ := parse(parts[2])

	a := uint8(255)
	if isRGBA && len(parts) >= 4 {
		av, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err == nil {
			a = uint8(av * 255)
		}
	}

	return Color{R: r, G: g, B: b, A: a}, nil
}

// namedColors maps CSS named colors to Color values (common subset).
var namedColors = map[string]Color{
	"black":   {0, 0, 0, 255},
	"white":   {255, 255, 255, 255},
	"red":     {255, 0, 0, 255},
	"green":   {0, 128, 0, 255},
	"blue":    {0, 0, 255, 255},
	"yellow":  {255, 255, 0, 255},
	"orange":  {255, 165, 0, 255},
	"gray":    {128, 128, 128, 255},
	"grey":    {128, 128, 128, 255},
	"silver":  {192, 192, 192, 255},
	"navy":    {0, 0, 128, 255},
	"teal":    {0, 128, 128, 255},
	"purple":  {128, 0, 128, 255},
	"maroon":  {128, 0, 0, 255},
	"lime":    {0, 255, 0, 255},
	"aqua":    {0, 255, 255, 255},
	"cyan":    {0, 255, 255, 255},
	"fuchsia": {255, 0, 255, 255},
	"magenta": {255, 0, 255, 255},
	"pink":    {255, 192, 203, 255},
	"brown":   {165, 42, 42, 255},
	"coral":   {255, 127, 80, 255},
	"salmon":  {250, 128, 114, 255},
	"gold":    {255, 215, 0, 255},
	"tan":     {210, 180, 140, 255},
	"khaki":   {240, 230, 140, 255},
	"transparent": {0, 0, 0, 0},
}
