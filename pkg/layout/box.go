// Package layout handles box model computation and page layout for Papyrus.
package layout

import (
	"github.com/grahms/papyrus/pkg/parser"
	"github.com/grahms/papyrus/pkg/style"
)

// BoxType categorises a box in the layout tree.
type BoxType int

const (
	BlockBox     BoxType = iota // block-level container
	InlineBox                   // inline content
	TextBox                     // leaf text node
	TableBox                    // table container
	TableRowBox                 // table row
	TableCellBox                // table cell
	AnonymousBox                // anonymous block / inline wrapper
	HRBox                       // horizontal rule
	ImageBox                    // image element
	PageBreakBox                // forced page break
)

// TableSection identifies which section of a table a row belongs to.
type TableSection int

const (
	TableSectionBody TableSection = iota
	TableSectionHead
	TableSectionFoot
)

// Box represents a node in the layout tree.
// All positions and sizes are in points.
type Box struct {
	Type  BoxType
	Node  *parser.Node // originating DOM node (may be nil for anonymous boxes)
	Style style.ComputedStyle

	// Geometry (set during layout)
	X, Y          float64 // position relative to content area of parent
	Width, Height float64 // content box dimensions

	// Children in the layout tree
	Children []*Box

	// Text content for TextBox
	Text string

	// For inline text: runs of text with their style
	InlineRuns []InlineRun

	// Page this box resides on (0-based)
	Page int

	// Absolute position on the page (set during pagination)
	AbsX, AbsY float64

	// For image boxes
	ImageSrc            string
	ImgWidth, ImgHeight float64

	// ListMarker is the bullet/number text drawn before a list item (e.g., "•", "1.").
	ListMarker string

	// HREF holds the URL for <a> link elements. Empty for non-link boxes.
	HREF string

	// TableSectionType marks a table row as belonging to thead, tbody, or tfoot.
	TableSectionType TableSection

	// InCollapsedTable is true for cells inside a border-collapse:collapse table.
	// The renderer skips individual cell borders and draws the table grid instead.
	InCollapsedTable bool

	// ColXPositions and RowYPositions store column/row edge coordinates for collapsed
	// table grid rendering (relative to the table content area origin).
	ColXPositions []float64
	RowYPositions []float64

	// TheadBox holds a reference to the <thead> rows box for table pagination.
	// When a table spans pages, this box is cloned and prepended on each new page.
	TheadBox *Box

	// TfootBox holds a reference to the <tfoot> rows box for table pagination.
	TfootBox *Box

	// InlineLineCount stores the number of laid-out text lines in this box.
	// Set during inline layout for use by orphans/widows pagination logic.
	InlineLineCount int
}

// InlineRun is a segment of inline text with associated style.
type InlineRun struct {
	Text  string
	Style style.ComputedStyle
	Node  *parser.Node
	HREF  string // non-empty when the run is inside an <a> element
}

// MarginBox returns the full outer extent including margin.
func (b *Box) MarginBox() (x, y, w, h float64) {
	s := b.Style
	x = b.X - s.MarginLeft
	y = b.Y - s.MarginTop
	w = b.Width + s.MarginLeft + s.MarginRight + s.BorderLeftWidth + s.BorderRightWidth + s.PaddingLeft + s.PaddingRight
	h = b.Height + s.MarginTop + s.MarginBottom + s.BorderTopWidth + s.BorderBottomWidth + s.PaddingTop + s.PaddingBottom
	return
}

// BorderBox returns the border box (includes border + padding + content).
func (b *Box) BorderBoxWidth() float64 {
	s := b.Style
	return b.Width + s.BorderLeftWidth + s.BorderRightWidth + s.PaddingLeft + s.PaddingRight
}

// BorderBoxHeight returns the border box height.
func (b *Box) BorderBoxHeight() float64 {
	s := b.Style
	return b.Height + s.BorderTopWidth + s.BorderBottomWidth + s.PaddingTop + b.Style.PaddingBottom
}

// ContentX returns the X offset of the content area within the border box.
func (b *Box) ContentX() float64 {
	return b.X + b.Style.PaddingLeft + b.Style.BorderLeftWidth
}

// ContentY returns the Y offset of the content area within the border box.
func (b *Box) ContentY() float64 {
	return b.Y + b.Style.PaddingTop + b.Style.BorderTopWidth
}

// OuterHeight is total vertical space including margin, border, padding, content.
func (b *Box) OuterHeight() float64 {
	s := b.Style
	return b.Height + s.MarginTop + s.MarginBottom + s.BorderTopWidth + s.BorderBottomWidth + s.PaddingTop + s.PaddingBottom
}

// OuterWidth is total horizontal space including margin, border, padding, content.
func (b *Box) OuterWidth() float64 {
	s := b.Style
	return b.Width + s.MarginLeft + s.MarginRight + s.BorderLeftWidth + s.BorderRightWidth + s.PaddingLeft + s.PaddingRight
}
