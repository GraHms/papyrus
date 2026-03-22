package render

import (
	"github.com/grahms/papyrus/pkg/style"
	"github.com/signintech/gopdf"
)

// drawBackground draws a filled background rectangle for a box.
func drawBackground(pdf *gopdf.GoPdf, x, y, w, h float64, color style.Color) {
	if color.A == 0 {
		return // transparent
	}
	setFillColor(pdf, color)
	pdf.Rectangle(x, y, x+w, y+h, "F", 0, 0)
}

// drawBorders draws the four CSS borders of a box.
func drawBorders(pdf *gopdf.GoPdf, x, y, w, h float64, cs style.ComputedStyle) {
	// Top border
	if cs.BorderTopWidth > 0 && cs.BorderTopStyle != "none" {
		setStrokeColor(pdf, cs.BorderTopColor)
		pdf.SetLineWidth(cs.BorderTopWidth)
		pdf.SetLineType(borderLineType(cs.BorderTopStyle))
		pdf.Line(x, y+cs.BorderTopWidth/2, x+w, y+cs.BorderTopWidth/2)
	}

	// Bottom border
	if cs.BorderBottomWidth > 0 && cs.BorderBottomStyle != "none" {
		setStrokeColor(pdf, cs.BorderBottomColor)
		pdf.SetLineWidth(cs.BorderBottomWidth)
		pdf.SetLineType(borderLineType(cs.BorderBottomStyle))
		by := y + h - cs.BorderBottomWidth/2
		pdf.Line(x, by, x+w, by)
	}

	// Left border
	if cs.BorderLeftWidth > 0 && cs.BorderLeftStyle != "none" {
		setStrokeColor(pdf, cs.BorderLeftColor)
		pdf.SetLineWidth(cs.BorderLeftWidth)
		pdf.SetLineType(borderLineType(cs.BorderLeftStyle))
		pdf.Line(x+cs.BorderLeftWidth/2, y, x+cs.BorderLeftWidth/2, y+h)
	}

	// Right border
	if cs.BorderRightWidth > 0 && cs.BorderRightStyle != "none" {
		setStrokeColor(pdf, cs.BorderRightColor)
		pdf.SetLineWidth(cs.BorderRightWidth)
		pdf.SetLineType(borderLineType(cs.BorderRightStyle))
		rx := x + w - cs.BorderRightWidth/2
		pdf.Line(rx, y, rx, y+h)
	}
}

// drawDebugOutline draws a colored outline for debug mode.
func drawDebugOutline(pdf *gopdf.GoPdf, x, y, w, h float64, color style.Color) {
	setStrokeColor(pdf, color)
	pdf.SetLineWidth(0.5)
	pdf.SetLineType("")
	pdf.Rectangle(x, y, x+w, y+h, "D", 0, 0)
}

// setFillColor sets the PDF fill color.
func setFillColor(pdf *gopdf.GoPdf, c style.Color) {
	pdf.SetFillColor(uint8(c.R), uint8(c.G), uint8(c.B))
}

// setStrokeColor sets the PDF stroke color.
func setStrokeColor(pdf *gopdf.GoPdf, c style.Color) {
	pdf.SetStrokeColor(uint8(c.R), uint8(c.G), uint8(c.B))
}

// borderLineType converts a CSS border-style to a gopdf line type string.
func borderLineType(style string) string {
	switch style {
	case "dashed":
		return "dashed"
	case "dotted":
		return "dotted"
	default:
		return ""
	}
}

// drawHR draws a horizontal rule.
func drawHR(pdf *gopdf.GoPdf, x, y, w float64, cs style.ComputedStyle) {
	color := cs.BorderTopColor
	if color.A == 0 {
		color = style.Color{R: 204, G: 204, B: 204, A: 255}
	}
	lineWidth := cs.BorderTopWidth
	if lineWidth <= 0 {
		lineWidth = 1
	}
	setStrokeColor(pdf, color)
	pdf.SetLineWidth(lineWidth)
	pdf.SetLineType("")
	pdf.Line(x, y+lineWidth/2, x+w, y+lineWidth/2)
}

// setTextColor sets the PDF text color.
func setTextColor(pdf *gopdf.GoPdf, c style.Color) {
	pdf.SetTextColor(uint8(c.R), uint8(c.G), uint8(c.B))
}

// drawCollapsedTableBorders draws a merged border grid for a border-collapse:collapse table.
// tableX/tableY are the absolute content-area origin; colXs and rowYs are relative to that origin.
// It draws the outer table border plus all internal grid lines using the table's own border style
// (falling back to a default thin black line if none is set).
func drawCollapsedTableBorders(pdf *gopdf.GoPdf, tableX, tableY float64, colXs, rowYs []float64, cs style.ComputedStyle) {
	if len(colXs) < 2 || len(rowYs) < 2 {
		return
	}

	lineColor := cs.BorderTopColor
	if lineColor.A == 0 {
		// No explicit table border set — derive from cell border defaults.
		lineColor = style.Color{R: 0, G: 0, B: 0, A: 255}
	}
	lineWidth := cs.BorderTopWidth
	if lineWidth <= 0 {
		lineWidth = 0.5
	}
	lineStyle := cs.BorderTopStyle
	if lineStyle == "none" || lineStyle == "" {
		lineStyle = "solid"
	}

	setStrokeColor(pdf, lineColor)
	pdf.SetLineWidth(lineWidth)
	pdf.SetLineType(borderLineType(lineStyle))

	tableW := colXs[len(colXs)-1]
	tableH := rowYs[len(rowYs)-1]

	// Outer border
	pdf.Line(tableX, tableY, tableX+tableW, tableY)
	pdf.Line(tableX, tableY+tableH, tableX+tableW, tableY+tableH)
	pdf.Line(tableX, tableY, tableX, tableY+tableH)
	pdf.Line(tableX+tableW, tableY, tableX+tableW, tableY+tableH)

	// Internal horizontal lines (between rows)
	for _, ry := range rowYs[1 : len(rowYs)-1] {
		pdf.Line(tableX, tableY+ry, tableX+tableW, tableY+ry)
	}

	// Internal vertical lines (between columns)
	for _, cx := range colXs[1 : len(colXs)-1] {
		pdf.Line(tableX+cx, tableY, tableX+cx, tableY+tableH)
	}
}
