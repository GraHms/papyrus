package layout

import (
	"github.com/ismaelvodacom/goxml2pdf/pkg/style"
)

// Context holds layout state.
type Context struct {
	// Page content area dimensions in pt
	PageWidth  float64
	PageHeight float64

	// DPI for px conversion
	DPI float64

	// TextMeasurer is used to measure text widths
	Measure TextMeasurer
}

// LayoutBlock performs block layout on a box, computing widths and heights.
// availWidth is the available content width (inside parent's padding/border).
// The box's X, Y, Width, Height are all set relative to the parent's content origin.
func LayoutBlock(ctx *Context, box *Box, availWidth float64, startY float64) float64 {
	if box == nil {
		return startY
	}

	cs := box.Style

	// Resolve box width
	boxWidth := resolveWidth(cs, availWidth)
	box.Width = boxWidth

	// Inner content width (what children get)
	innerWidth := boxWidth - cs.PaddingLeft - cs.PaddingRight - cs.BorderLeftWidth - cs.BorderRightWidth
	if innerWidth < 0 {
		innerWidth = 0
	}

	// Layout children
	switch box.Type {
	case TableBox:
		layoutTable(ctx, box, innerWidth, startY)
	case HRBox:
		box.Height = cs.BorderTopWidth
		if box.Height < 1 {
			box.Height = 1
		}
	case ImageBox:
		layoutImage(ctx, box, innerWidth)
	case PageBreakBox:
		box.Height = 0
	default:
		layoutBlockChildren(ctx, box, innerWidth)
	}

	return startY + box.OuterHeight()
}

// layoutBlockChildren lays out the children of a block box.
func layoutBlockChildren(ctx *Context, parent *Box, innerWidth float64) {
	if len(parent.Children) == 0 {
		parent.Height = 0
		return
	}

	// Determine if this box has only inline content
	hasBlock := false
	for _, child := range parent.Children {
		if child.Type == BlockBox || child.Type == TableBox || child.Type == HRBox || child.Type == ImageBox {
			hasBlock = true
			break
		}
	}

	if !hasBlock {
		// Inline formatting context
		layoutInlineBlock(ctx, parent, innerWidth)
		return
	}

	// Block formatting context
	curY := 0.0
	prevMarginBottom := 0.0

	for _, child := range parent.Children {
		if child.Type == PageBreakBox {
			child.Y = curY
			child.Width = innerWidth
			child.Height = 0
			continue
		}

		cs := child.Style

		// Margin collapsing: collapse top margin with previous bottom margin
		topMargin := cs.MarginTop
		if prevMarginBottom > topMargin {
			topMargin = prevMarginBottom
		}

		child.X = cs.MarginLeft
		child.Y = curY + topMargin

		LayoutBlock(ctx, child, innerWidth, 0)

		curY = child.Y + child.OuterHeight() - cs.MarginTop // advance by effective height
		prevMarginBottom = cs.MarginBottom
	}

	// Apply last margin
	parent.Height = curY + prevMarginBottom
	if parent.Height < 0 {
		parent.Height = 0
	}
}

// layoutInlineBlock lays out inline content in the block.
func layoutInlineBlock(ctx *Context, parent *Box, innerWidth float64) {
	cs := parent.Style

	// Collect all inline runs
	runs := CollectInlineRuns(parent, cs)
	if len(runs) == 0 {
		parent.Height = 0
		return
	}

	// Break into lines
	lines := BreakIntoLines(runs, innerWidth, ctx.Measure)

	// Compute total height
	totalHeight := 0.0
	for _, line := range lines {
		h := line.Height
		if h <= 0 {
			h = cs.LineHeight
			if h <= 0 {
				h = cs.FontSize * 1.2
			}
		}
		totalHeight += h
	}

	parent.Height = totalHeight

	// Store lines in the box for rendering
	// We flatten lines back into inline runs on the box
	var flatRuns []InlineRun
	for _, line := range lines {
		flatRuns = append(flatRuns, line.Runs...)
		// Add a newline marker between lines
		if line.Height > 0 {
			flatRuns = append(flatRuns, InlineRun{Text: "\n", Style: cs})
		}
	}
	parent.InlineRuns = flatRuns
}

// resolveWidth computes the content width of a block box.
func resolveWidth(cs style.ComputedStyle, availWidth float64) float64 {
	var w float64

	if cs.Width.IsAuto() {
		// Block boxes with auto width stretch to fill available width
		w = availWidth - cs.MarginLeft - cs.MarginRight
	} else {
		w = cs.Width.ToPoints(availWidth, cs.FontSize, 10, 96)
	}

	// Apply min/max constraints
	if cs.MinWidth > 0 && w < cs.MinWidth {
		w = cs.MinWidth
	}
	if cs.MaxWidth >= 0 && w > cs.MaxWidth {
		w = cs.MaxWidth
	}

	if w < 0 {
		w = 0
	}
	return w
}

// layoutImage computes image box dimensions.
func layoutImage(ctx *Context, box *Box, availWidth float64) {
	w := box.ImgWidth
	h := box.ImgHeight

	if w <= 0 && h <= 0 {
		// Default to available width, maintain aspect ratio later
		w = availWidth
		h = availWidth * 0.75 // placeholder ratio
	} else if w <= 0 {
		w = availWidth
	} else if h <= 0 {
		h = w * 0.75
	}

	box.Width = w
	box.Height = h
}

// layoutTable is a simple table layout: equal column widths, auto rows.
func layoutTable(ctx *Context, table *Box, availWidth float64, startY float64) {
	if len(table.Children) == 0 {
		table.Height = 0
		return
	}

	// Collect all rows (through thead/tbody/tfoot groups)
	var rows []*Box
	for _, group := range table.Children {
		if group.Type == BlockBox || group.Type == TableRowBox {
			if group.Node != nil && (group.Node.Tag == "thead" || group.Node.Tag == "tbody" || group.Node.Tag == "tfoot") {
				for _, row := range group.Children {
					rows = append(rows, row)
				}
			} else if group.Node != nil && group.Node.Tag == "tr" {
				rows = append(rows, group)
			}
		}
	}

	if len(rows) == 0 {
		table.Height = 0
		return
	}

	// Count max columns
	maxCols := 0
	for _, row := range rows {
		cols := 0
		for _, cell := range row.Children {
			colspan := 1
			if cell.Node != nil {
				if cs, ok := cell.Node.GetAttribute("colspan"); ok {
					n := 0
					if _, err := parseIntAttr(cs, &n); err == nil {
						colspan = n
					}
				}
			}
			cols += colspan
		}
		if cols > maxCols {
			maxCols = cols
		}
	}

	if maxCols == 0 {
		maxCols = 1
	}

	colWidth := availWidth / float64(maxCols)

	// Layout each row
	curY := 0.0
	for _, row := range rows {
		rowY := curY
		rowHeight := 0.0
		curX := 0.0

		for _, cell := range row.Children {
			cs := cell.Style
			colspan := 1
			if cell.Node != nil {
				if csAttr, ok := cell.Node.GetAttribute("colspan"); ok {
					n := 0
					if _, err := parseIntAttr(csAttr, &n); err == nil && n > 1 {
						colspan = n
					}
				}
			}

			cellWidth := colWidth*float64(colspan) - cs.MarginLeft - cs.MarginRight
			cell.X = curX + cs.MarginLeft
			cell.Y = rowY + cs.MarginTop

			innerW := cellWidth - cs.PaddingLeft - cs.PaddingRight - cs.BorderLeftWidth - cs.BorderRightWidth
			if innerW < 0 {
				innerW = 0
			}
			cell.Width = cellWidth

			// Layout cell content
			layoutBlockChildren(ctx, cell, innerW)

			cellH := cell.Height + cs.PaddingTop + cs.PaddingBottom + cs.BorderTopWidth + cs.BorderBottomWidth
			if cellH > rowHeight {
				rowHeight = cellH
			}

			curX += colWidth * float64(colspan)
		}

		// Set all cells to same row height
		for _, cell := range row.Children {
			cell.Height = rowHeight - cell.Style.PaddingTop - cell.Style.PaddingBottom - cell.Style.BorderTopWidth - cell.Style.BorderBottomWidth
		}

		row.X = 0
		row.Y = rowY
		row.Width = availWidth
		row.Height = rowHeight

		curY += rowHeight
	}

	table.Width = availWidth
	table.Height = curY
}

// parseIntAttr parses an integer from a string attribute.
func parseIntAttr(s string, out *int) (string, error) {
	var n int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	*out = n
	return s, nil
}
