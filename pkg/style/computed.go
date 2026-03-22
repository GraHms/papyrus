package style

// ComputedStyle holds all resolved CSS property values for a single DOM node.
// All length values have been converted to points.
// All color values have been parsed to Color structs.
type ComputedStyle struct {
	// Typography
	FontFamily     string
	FontSize       float64 // pt
	FontWeight     string  // "normal", "bold", "100"-"900"
	FontStyle      string  // "normal", "italic"
	Color          Color
	LineHeight     float64 // pt (absolute, resolved from ratio/length)
	TextAlign      string  // "left", "right", "center", "justify"
	TextDecoration string  // "none", "underline", "line-through"
	LetterSpacing  float64 // pt
	TextTransform  string  // "none", "uppercase", "lowercase", "capitalize"
	WhiteSpace     string  // "normal", "nowrap", "pre"
	TextIndent     float64 // pt

	// Box model (all in pt)
	Width     Length
	Height    Length
	MinWidth  float64
	MaxWidth  float64 // -1 means none
	MinHeight float64
	MaxHeight float64 // -1 means none

	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64

	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64

	BorderTopWidth    float64
	BorderRightWidth  float64
	BorderBottomWidth float64
	BorderLeftWidth   float64

	BorderTopColor    Color
	BorderRightColor  Color
	BorderBottomColor Color
	BorderLeftColor   Color

	BorderTopStyle    string // "none", "solid", "dashed", "dotted"
	BorderRightStyle  string
	BorderBottomStyle string
	BorderLeftStyle   string

	// Colors and backgrounds
	BackgroundColor Color

	// Layout
	Display       string // "block", "inline", "table", "table-row", "table-cell", "none"
	VerticalAlign string // "top", "middle", "bottom", "baseline"
	Overflow      string // "visible", "hidden"

	// Table
	BorderCollapse string // "collapse", "separate"
	BorderSpacing  float64
	TableLayout    string // "auto", "fixed"

	// Page break
	PageBreakBefore string // "auto", "always", "avoid"
	PageBreakAfter  string
	PageBreakInside string
	Orphans         int
	Widows          int
}

// DefaultStyle returns a ComputedStyle with all CSS initial values.
func DefaultStyle(dpi float64) ComputedStyle {
	return ComputedStyle{
		FontFamily:     "Liberation Sans",
		FontSize:       10,
		FontWeight:     "normal",
		FontStyle:      "normal",
		Color:          Color{R: 0, G: 0, B: 0, A: 255},
		LineHeight:     12, // 1.2 * 10pt
		TextAlign:      "left",
		TextDecoration: "none",
		LetterSpacing:  0,
		TextTransform:  "none",
		WhiteSpace:     "normal",
		TextIndent:     0,

		Width:     Auto,
		Height:    Auto,
		MinWidth:  0,
		MaxWidth:  -1,
		MinHeight: 0,
		MaxHeight: -1,

		MarginTop: 0, MarginRight: 0, MarginBottom: 0, MarginLeft: 0,
		PaddingTop: 0, PaddingRight: 0, PaddingBottom: 0, PaddingLeft: 0,

		BorderTopWidth: 0, BorderRightWidth: 0, BorderBottomWidth: 0, BorderLeftWidth: 0,
		BorderTopColor:    Color{A: 255},
		BorderRightColor:  Color{A: 255},
		BorderBottomColor: Color{A: 255},
		BorderLeftColor:   Color{A: 255},
		BorderTopStyle:    "none",
		BorderRightStyle:  "none",
		BorderBottomStyle: "none",
		BorderLeftStyle:   "none",

		BackgroundColor: Color{A: 0}, // transparent

		Display:       "block",
		VerticalAlign: "top",
		Overflow:      "visible",

		BorderCollapse: "separate",
		BorderSpacing:  0,
		TableLayout:    "auto",

		PageBreakBefore: "auto",
		PageBreakAfter:  "auto",
		PageBreakInside: "auto",
		Orphans:         2,
		Widows:          2,
	}
}

// IsBold returns true if font-weight is bold or >= 700.
func (cs *ComputedStyle) IsBold() bool {
	switch cs.FontWeight {
	case "bold", "bolder":
		return true
	case "700", "800", "900":
		return true
	}
	return false
}

// IsItalic returns true if font-style is italic or oblique.
func (cs *ComputedStyle) IsItalic() bool {
	return cs.FontStyle == "italic" || cs.FontStyle == "oblique"
}

// HorizontalBorderWidth returns the total horizontal border width.
func (cs *ComputedStyle) HorizontalBorderWidth() float64 {
	return cs.BorderLeftWidth + cs.BorderRightWidth
}

// VerticalBorderWidth returns the total vertical border width.
func (cs *ComputedStyle) VerticalBorderWidth() float64 {
	return cs.BorderTopWidth + cs.BorderBottomWidth
}

// HorizontalPadding returns the total horizontal padding.
func (cs *ComputedStyle) HorizontalPadding() float64 {
	return cs.PaddingLeft + cs.PaddingRight
}

// VerticalPadding returns the total vertical padding.
func (cs *ComputedStyle) VerticalPadding() float64 {
	return cs.PaddingTop + cs.PaddingBottom
}

// HorizontalMargin returns the total horizontal margin.
func (cs *ComputedStyle) HorizontalMargin() float64 {
	return cs.MarginLeft + cs.MarginRight
}

// VerticalMargin returns the total vertical margin.
func (cs *ComputedStyle) VerticalMargin() float64 {
	return cs.MarginTop + cs.MarginBottom
}
