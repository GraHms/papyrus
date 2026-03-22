package layout

import (
	"github.com/grahms/pdfml/pkg/style"
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

	// Enforce min-height / max-height constraints
	if cs.MinHeight > 0 && box.Height < cs.MinHeight {
		box.Height = cs.MinHeight
	}
	if cs.MaxHeight >= 0 && box.Height > cs.MaxHeight {
		box.Height = cs.MaxHeight
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
	parent.InlineLineCount = len(lines)

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

// layoutTable performs full table layout: column sizing, rowspan/colspan, border-spacing, collapse.
func layoutTable(ctx *Context, table *Box, availWidth float64, startY float64) {
	if len(table.Children) == 0 {
		table.Height = 0
		return
	}

	collapsed := table.Style.BorderCollapse == "collapse"
	spacing := table.Style.BorderSpacing
	if collapsed {
		spacing = 0
	}

	// Separate caption from row groups; classify thead/tbody/tfoot rows.
	var captionBox *Box
	var theadRows, tbodyRows, tfootRows []*Box

	for _, child := range table.Children {
		tag := ""
		if child.Node != nil {
			tag = child.Node.Tag
		}
		switch tag {
		case "caption":
			captionBox = child
		case "thead":
			for _, row := range child.Children {
				if row.Node != nil && row.Node.Tag == "tr" {
					row.TableSectionType = TableSectionHead
					theadRows = append(theadRows, row)
				}
			}
		case "tfoot":
			for _, row := range child.Children {
				if row.Node != nil && row.Node.Tag == "tr" {
					row.TableSectionType = TableSectionFoot
					tfootRows = append(tfootRows, row)
				}
			}
		case "tbody":
			for _, row := range child.Children {
				if row.Node != nil && row.Node.Tag == "tr" {
					tbodyRows = append(tbodyRows, row)
				}
			}
		case "tr":
			tbodyRows = append(tbodyRows, child)
		}
	}

	allRows := make([]*Box, 0, len(theadRows)+len(tbodyRows)+len(tfootRows))
	allRows = append(allRows, theadRows...)
	allRows = append(allRows, tbodyRows...)
	allRows = append(allRows, tfootRows...)

	if len(allRows) == 0 {
		table.Height = 0
		return
	}

	maxCols := countTableColumns(allRows)
	if maxCols == 0 {
		maxCols = 1
	}

	// Compute column widths
	colWidths := computeColumnWidths(ctx, table, allRows, maxCols, availWidth, spacing)

	// Mark cells that are inside a collapsed table
	if collapsed {
		for _, row := range allRows {
			for _, cell := range row.Children {
				cell.InCollapsedTable = true
			}
		}
	}

	// Build rowspan/colspan grid.
	// grid[row][col] holds the cell box that starts at this slot; occupied marks all slots.
	type gridEntry struct {
		cell    *Box
		rowspan int
		colspan int
	}
	grid := make([][]gridEntry, len(allRows))
	for i := range grid {
		grid[i] = make([]gridEntry, maxCols)
	}
	type rc struct{ r, c int }
	occupied := make(map[rc]bool)

	for rowIdx, row := range allRows {
		colIdx := 0
		for _, cell := range row.Children {
			// Advance past occupied slots
			for colIdx < maxCols && occupied[rc{rowIdx, colIdx}] {
				colIdx++
			}
			if colIdx >= maxCols {
				break
			}
			cs, rs := getCellSpans(cell)
			// Mark spanned slots as occupied (all except the origin)
			for r := rowIdx; r < rowIdx+rs && r < len(allRows); r++ {
				for c := colIdx; c < colIdx+cs && c < maxCols; c++ {
					if r != rowIdx || c != colIdx {
						occupied[rc{r, c}] = true
					}
				}
			}
			grid[rowIdx][colIdx] = gridEntry{cell: cell, rowspan: rs, colspan: cs}
			colIdx += cs
		}
	}

	// First pass: layout non-rowspan cells and compute row heights.
	rowHeights := make([]float64, len(allRows))
	for rowIdx := range allRows {
		for colIdx := 0; colIdx < maxCols; colIdx++ {
			e := grid[rowIdx][colIdx]
			if e.cell == nil || e.rowspan > 1 {
				continue
			}
			cellW := cellContentWidth(e.cell, e.colspan, colIdx, colWidths, maxCols, spacing)
			innerW := cellW - e.cell.Style.PaddingLeft - e.cell.Style.PaddingRight
			if !collapsed {
				innerW -= e.cell.Style.BorderLeftWidth + e.cell.Style.BorderRightWidth
			}
			if innerW < 0 {
				innerW = 0
			}
			e.cell.Width = cellW
			layoutBlockChildren(ctx, e.cell, innerW)

			borderH := e.cell.Style.PaddingTop + e.cell.Style.PaddingBottom
			if !collapsed {
				borderH += e.cell.Style.BorderTopWidth + e.cell.Style.BorderBottomWidth
			}
			cellH := e.cell.Height + borderH + e.cell.Style.MarginTop + e.cell.Style.MarginBottom
			if cellH > rowHeights[rowIdx] {
				rowHeights[rowIdx] = cellH
			}
		}
	}

	// Second pass: layout rowspan > 1 cells and expand row heights as needed.
	for rowIdx := range allRows {
		for colIdx := 0; colIdx < maxCols; colIdx++ {
			e := grid[rowIdx][colIdx]
			if e.cell == nil || e.rowspan <= 1 {
				continue
			}
			cellW := cellContentWidth(e.cell, e.colspan, colIdx, colWidths, maxCols, spacing)
			innerW := cellW - e.cell.Style.PaddingLeft - e.cell.Style.PaddingRight
			if !collapsed {
				innerW -= e.cell.Style.BorderLeftWidth + e.cell.Style.BorderRightWidth
			}
			if innerW < 0 {
				innerW = 0
			}
			e.cell.Width = cellW
			layoutBlockChildren(ctx, e.cell, innerW)

			borderH := e.cell.Style.PaddingTop + e.cell.Style.PaddingBottom
			if !collapsed {
				borderH += e.cell.Style.BorderTopWidth + e.cell.Style.BorderBottomWidth
			}
			cellH := e.cell.Height + borderH + e.cell.Style.MarginTop + e.cell.Style.MarginBottom

			// Sum of heights for rows this cell spans
			spanEnd := rowIdx + e.rowspan
			if spanEnd > len(allRows) {
				spanEnd = len(allRows)
			}
			spannedH := 0.0
			for r := rowIdx; r < spanEnd; r++ {
				spannedH += rowHeights[r]
			}
			if e.rowspan > 1 && spacing > 0 {
				spannedH += spacing * float64(e.rowspan-1)
			}
			if cellH > spannedH {
				extra := cellH - spannedH
				perRow := extra / float64(spanEnd-rowIdx)
				for r := rowIdx; r < spanEnd; r++ {
					rowHeights[r] += perRow
				}
			}
		}
	}

	// Layout caption before rows.
	curY := 0.0
	if captionBox != nil {
		captionBox.Width = availWidth
		captionBox.X = 0
		captionBox.Y = curY
		layoutBlockChildren(ctx, captionBox, availWidth)
		curY += captionBox.OuterHeight()
	}

	// Leading spacing
	if spacing > 0 {
		curY += spacing
	}

	// Build column X positions (for collapsed grid rendering).
	colXs := make([]float64, maxCols+1)
	xOff := spacing
	for c := 0; c < maxCols; c++ {
		colXs[c] = xOff
		xOff += colWidths[c]
		if spacing > 0 {
			xOff += spacing
		}
	}
	colXs[maxCols] = xOff

	// Build row Y positions (for collapsed grid rendering).
	rowYs := make([]float64, len(allRows)+1)
	rowStart := curY
	for r, h := range rowHeights {
		rowYs[r] = curY - rowStart
		curY += h
		if spacing > 0 {
			curY += spacing
		}
	}
	rowYs[len(allRows)] = curY - rowStart

	// Store grid positions on the table box for collapsed-border rendering.
	table.ColXPositions = colXs
	table.RowYPositions = rowYs

	// Third pass: assign final positions to all cells.
	curY = rowStart
	for rowIdx, row := range allRows {
		rowY := curY
		for colIdx := 0; colIdx < maxCols; colIdx++ {
			if occupied[rc{rowIdx, colIdx}] {
				continue
			}
			e := grid[rowIdx][colIdx]
			if e.cell == nil {
				continue
			}
			ccs := e.cell.Style
			e.cell.X = colXs[colIdx] + ccs.MarginLeft
			e.cell.Y = rowY + ccs.MarginTop

			if e.rowspan > 1 {
				spanEnd := rowIdx + e.rowspan
				if spanEnd > len(allRows) {
					spanEnd = len(allRows)
				}
				spannedH := 0.0
				for r := rowIdx; r < spanEnd; r++ {
					spannedH += rowHeights[r]
				}
				if e.rowspan > 1 && spacing > 0 {
					spannedH += spacing * float64(e.rowspan-1)
				}
				borderH := ccs.PaddingTop + ccs.PaddingBottom
				if !collapsed {
					borderH += ccs.BorderTopWidth + ccs.BorderBottomWidth
				}
				e.cell.Height = spannedH - borderH - ccs.MarginTop - ccs.MarginBottom
			} else {
				borderH := ccs.PaddingTop + ccs.PaddingBottom
				if !collapsed {
					borderH += ccs.BorderTopWidth + ccs.BorderBottomWidth
				}
				e.cell.Height = rowHeights[rowIdx] - borderH - ccs.MarginTop - ccs.MarginBottom
			}
			if e.cell.Height < 0 {
				e.cell.Height = 0
			}
		}

		row.X = 0
		row.Y = rowY
		row.Width = availWidth
		row.Height = rowHeights[rowIdx]
		curY += rowHeights[rowIdx]
		if spacing > 0 {
			curY += spacing
		}
	}

	table.Width = availWidth
	table.Height = curY
}

// countTableColumns returns the maximum number of columns across all rows (accounting for colspan).
func countTableColumns(rows []*Box) int {
	max := 0
	for _, row := range rows {
		cols := 0
		for _, cell := range row.Children {
			cs, _ := getCellSpans(cell)
			cols += cs
		}
		if cols > max {
			max = cols
		}
	}
	return max
}

// getCellSpans returns the colspan and rowspan of a cell box.
func getCellSpans(cell *Box) (colspan, rowspan int) {
	colspan, rowspan = 1, 1
	if cell.Node == nil {
		return
	}
	if v, ok := cell.Node.GetAttribute("colspan"); ok {
		n := 0
		if _, err := parseIntAttr(v, &n); err == nil && n > 1 {
			colspan = n
		}
	}
	if v, ok := cell.Node.GetAttribute("rowspan"); ok {
		n := 0
		if _, err := parseIntAttr(v, &n); err == nil && n > 1 {
			rowspan = n
		}
	}
	return
}

// cellContentWidth computes the border-box width of a cell given its colspan and column widths.
func cellContentWidth(cell *Box, colspan, colIdx int, colWidths []float64, maxCols int, spacing float64) float64 {
	w := 0.0
	for c := colIdx; c < colIdx+colspan && c < maxCols; c++ {
		w += colWidths[c]
	}
	if colspan > 1 && spacing > 0 {
		w += spacing * float64(colspan-1)
	}
	ccs := cell.Style
	w -= ccs.MarginLeft + ccs.MarginRight
	if w < 0 {
		w = 0
	}
	return w
}

// computeColumnWidths determines column widths based on table-layout property.
func computeColumnWidths(ctx *Context, table *Box, rows []*Box, numCols int, availWidth, spacing float64) []float64 {
	// Effective width after subtracting all spacing gaps.
	effectiveW := availWidth
	if spacing > 0 {
		effectiveW -= spacing * float64(numCols+1)
	}
	if effectiveW < 0 {
		effectiveW = 0
	}

	colWidths := make([]float64, numCols)

	switch table.Style.TableLayout {
	case "fixed":
		// Use explicit widths from the first row; fill remainder equally.
		if len(rows) > 0 {
			col := 0
			for _, cell := range rows[0].Children {
				if col >= numCols {
					break
				}
				cs, _ := getCellSpans(cell)
				if !cell.Style.Width.IsAuto() {
					w := cell.Style.Width.ToPoints(effectiveW, cell.Style.FontSize, 10, 96)
					perCol := w / float64(cs)
					for c := col; c < col+cs && c < numCols; c++ {
						colWidths[c] = perCol
					}
				}
				col += cs
			}
		}
		total, unset := 0.0, 0
		for _, w := range colWidths {
			total += w
		}
		for _, w := range colWidths {
			if w == 0 {
				unset++
			}
		}
		if unset > 0 {
			rem := effectiveW - total
			if rem < 0 {
				rem = 0
			}
			per := rem / float64(unset)
			for i, w := range colWidths {
				if w == 0 {
					colWidths[i] = per
				}
			}
		}

	default: // "auto"
		// Measure natural (one-line) content width for each column.
		naturalWidths := make([]float64, numCols)

		// Build column assignments for the first pass
		type rc struct{ r, c int }
		occupied := make(map[rc]bool)
		for rowIdx, row := range rows {
			colIdx := 0
			for _, cell := range row.Children {
				for colIdx < numCols && occupied[rc{rowIdx, colIdx}] {
					colIdx++
				}
				if colIdx >= numCols {
					break
				}
				cs, rs := getCellSpans(cell)
				for r := rowIdx; r < rowIdx+rs && r < len(rows); r++ {
					for c := colIdx; c < colIdx+cs && c < numCols; c++ {
						if r != rowIdx || c != colIdx {
							occupied[rc{r, c}] = true
						}
					}
				}
				// Measure cell's natural width
				cellNatW := measureCellNaturalWidth(ctx, cell)
				cellNatW += cell.Style.PaddingLeft + cell.Style.PaddingRight
				if table.Style.BorderCollapse != "collapse" {
					cellNatW += cell.Style.BorderLeftWidth + cell.Style.BorderRightWidth
				}
				cellNatW += cell.Style.MarginLeft + cell.Style.MarginRight

				// Attribute natural width to first column of span (approximation)
				if cellNatW/float64(cs) > naturalWidths[colIdx] {
					naturalWidths[colIdx] = cellNatW / float64(cs)
				}
				colIdx += cs
			}
		}

		// Scale to fit available width proportionally.
		total := 0.0
		for _, w := range naturalWidths {
			total += w
		}
		if total <= 0 {
			// No content — equal distribution
			for i := range colWidths {
				colWidths[i] = effectiveW / float64(numCols)
			}
		} else if total <= effectiveW {
			// Natural widths fit — use them and distribute remainder equally
			remainder := effectiveW - total
			extra := remainder / float64(numCols)
			for i, w := range naturalWidths {
				colWidths[i] = w + extra
			}
		} else {
			// Scale down proportionally
			for i, w := range naturalWidths {
				colWidths[i] = w / total * effectiveW
			}
		}
	}

	return colWidths
}

// measureCellNaturalWidth returns the sum of inline run widths for a cell
// (the max-content width approximation: all text on one line).
func measureCellNaturalWidth(ctx *Context, cell *Box) float64 {
	if ctx.Measure == nil {
		return 0
	}
	runs := CollectInlineRuns(cell, cell.Style)
	total := 0.0
	for _, run := range runs {
		if run.Text != "\n" {
			total += ctx.Measure(run.Text, run.Style)
		}
	}
	return total
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
