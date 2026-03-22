package layout

import (
	"github.com/grahms/papyrus/pkg/parser"
	"github.com/grahms/papyrus/pkg/style"
)

// Page represents a single output page with its laid-out boxes.
type Page struct {
	Width, Height float64
	// TopLevelBoxes holds only the direct body-level boxes for this page.
	// Rendering recurses into their children.
	Boxes []*Box

	// Header and footer boxes (rendered on every page)
	Header *Box
	Footer *Box

	// Margins
	MarginTop, MarginRight, MarginBottom, MarginLeft float64

	// Number (1-based) and total count — filled after all pages are created
	Number int
	Total  int
}

// PageLayout manages the pagination of boxes across multiple pages.
type PageLayout struct {
	PageStyle style.PageStyle
	Pages     []*Page

	headerBox    *Box
	footerBox    *Box
	headerHeight float64
	footerHeight float64

	// First-page header/footer (optional)
	firstHeaderBox    *Box
	firstFooterBox    *Box
	firstHeaderHeight float64
	firstFooterHeight float64

	ctx *Context
}

// NewPageLayout creates a PageLayout from a page style.
func NewPageLayout(ps style.PageStyle, ctx *Context) *PageLayout {
	return &PageLayout{
		PageStyle: ps,
		ctx:       ctx,
	}
}

// SetHeader sets the page header box (rendered on every page).
func (pl *PageLayout) SetHeader(box *Box) {
	pl.headerBox = box
}

// SetFooter sets the page footer box (rendered on every page).
func (pl *PageLayout) SetFooter(box *Box) {
	pl.footerBox = box
}

// SetFirstHeader sets a header that appears only on the first page.
func (pl *PageLayout) SetFirstHeader(box *Box) {
	pl.firstHeaderBox = box
}

// SetFirstFooter sets a footer that appears only on the first page.
func (pl *PageLayout) SetFirstFooter(box *Box) {
	pl.firstFooterBox = box
}

// Layout performs layout and paginates the root body box across pages.
func (pl *PageLayout) Layout(root *Box) {
	ps := pl.PageStyle
	contentWidth := ps.Width - ps.MarginLeft - ps.MarginRight

	pl.ctx.PageWidth = contentWidth
	pl.ctx.PageHeight = ps.Height - ps.MarginTop - ps.MarginBottom

	// Layout the root body box
	root.Width = contentWidth
	layoutBlockChildren(pl.ctx, root, contentWidth)

	// Layout header and footer, measure their heights
	if pl.headerBox != nil {
		pl.headerBox.Width = contentWidth
		layoutBlockChildren(pl.ctx, pl.headerBox, contentWidth)
		pl.headerHeight = boxOuterHeightWithStyle(pl.headerBox)
	}
	if pl.footerBox != nil {
		pl.footerBox.Width = contentWidth
		layoutBlockChildren(pl.ctx, pl.footerBox, contentWidth)
		pl.footerHeight = boxOuterHeightWithStyle(pl.footerBox)
	}
	if pl.firstHeaderBox != nil {
		pl.firstHeaderBox.Width = contentWidth
		layoutBlockChildren(pl.ctx, pl.firstHeaderBox, contentWidth)
		pl.firstHeaderHeight = boxOuterHeightWithStyle(pl.firstHeaderBox)
	}
	if pl.firstFooterBox != nil {
		pl.firstFooterBox.Width = contentWidth
		layoutBlockChildren(pl.ctx, pl.firstFooterBox, contentWidth)
		pl.firstFooterHeight = boxOuterHeightWithStyle(pl.firstFooterBox)
	}

	// Available content height per page (standard pages)
	pageContentH := ps.Height - ps.MarginTop - ps.MarginBottom - pl.headerHeight - pl.footerHeight

	// Paginate: walk body's direct children sequentially
	pl.newPage()
	curPageIdx := 0
	curPageY := 0.0 // cursor Y within the current page content area
	pendingBreakAfter := false

	for _, child := range root.Children {
		// Forced page break (element)
		if child.Type == PageBreakBox {
			curPageIdx++
			pl.newPage()
			curPageY = 0
			pendingBreakAfter = false
			continue
		}

		// Deferred page-break-after from previous child
		if pendingBreakAfter {
			curPageIdx++
			pl.newPage()
			curPageY = 0
			pendingBreakAfter = false
		}

		// CSS page-break-before: always
		if child.Style.PageBreakBefore == "always" && curPageY > 0 {
			curPageIdx++
			pl.newPage()
			curPageY = 0
		}

		childH := child.OuterHeight()

		// Effective page content height (first page may differ)
		effectiveH := pageContentH
		if curPageIdx == 0 && (pl.firstHeaderBox != nil || pl.firstFooterBox != nil) {
			hh := pl.headerHeight
			fh := pl.footerHeight
			if pl.firstHeaderBox != nil {
				hh = pl.firstHeaderHeight
			}
			if pl.firstFooterBox != nil {
				fh = pl.firstFooterHeight
			}
			effectiveH = ps.Height - ps.MarginTop - ps.MarginBottom - hh - fh
		}

		// page-break-inside: avoid — if the child doesn't fit but would fit on a fresh page
		if child.Style.PageBreakInside == "avoid" && curPageY > 0 {
			if curPageY+childH > effectiveH && childH <= pageContentH {
				curPageIdx++
				pl.newPage()
				curPageY = 0
				effectiveH = pageContentH
			}
		}

		// Auto-break if child overflows current page (but not if it's too tall for any page)
		if curPageY+childH > effectiveH && curPageY > 0 && childH <= effectiveH {
			// Orphans/widows check: if this is a text block, see if we should
			// push the whole thing to the next page to satisfy orphans/widows
			if shouldPushForOrphansWidows(child, curPageY, effectiveH) {
				curPageIdx++
				pl.newPage()
				curPageY = 0
				effectiveH = pageContentH
			} else {
				curPageIdx++
				pl.newPage()
				curPageY = 0
				effectiveH = pageContentH
			}
		}

		cs := child.Style

		// Determine current header height for placement
		currentHeaderH := pl.headerHeight
		if curPageIdx == 0 && pl.firstHeaderBox != nil {
			currentHeaderH = pl.firstHeaderHeight
		}

		// Assign absolute position on the page
		child.Page = curPageIdx
		child.AbsX = ps.MarginLeft + cs.MarginLeft
		child.AbsY = ps.MarginTop + currentHeaderH + curPageY + cs.MarginTop

		// Only add top-level boxes to page; rendering recurses into children
		pl.Pages[curPageIdx].Boxes = append(pl.Pages[curPageIdx].Boxes, child)

		curPageY += childH

		// CSS page-break-after: always
		if child.Style.PageBreakAfter == "always" {
			pendingBreakAfter = true
		}
	}

	// Fill in page numbers
	total := len(pl.Pages)
	for i, p := range pl.Pages {
		p.Number = i + 1
		p.Total = total
	}
}

// shouldPushForOrphansWidows returns true if pushing the entire block to the
// next page would better satisfy orphans/widows constraints than splitting it.
func shouldPushForOrphansWidows(child *Box, curPageY, pageContentH float64) bool {
	if child.InlineLineCount == 0 {
		return false
	}

	orphans := child.Style.Orphans
	widows := child.Style.Widows
	if orphans <= 0 {
		orphans = 2 // CSS default
	}
	if widows <= 0 {
		widows = 2 // CSS default
	}

	totalLines := child.InlineLineCount
	remaining := pageContentH - curPageY
	lineH := child.OuterHeight() / float64(totalLines)
	if lineH <= 0 {
		return false
	}

	linesFit := int(remaining / lineH)
	if linesFit < 0 {
		linesFit = 0
	}

	linesOnNext := totalLines - linesFit

	// If we can't satisfy orphans at bottom of current page, push entire block
	if linesFit < orphans && linesFit < totalLines {
		return true
	}
	// If we can't satisfy widows at top of next page, push entire block
	if linesOnNext > 0 && linesOnNext < widows {
		return true
	}

	return false
}

// newPage appends a fresh page to the list.
func (pl *PageLayout) newPage() {
	ps := pl.PageStyle
	pageIdx := len(pl.Pages)

	header := pl.headerBox
	footer := pl.footerBox

	// Use first-page header/footer for page 0
	if pageIdx == 0 {
		if pl.firstHeaderBox != nil {
			header = pl.firstHeaderBox
		}
		if pl.firstFooterBox != nil {
			footer = pl.firstFooterBox
		}
	}

	page := &Page{
		Width:        ps.Width,
		Height:       ps.Height,
		MarginTop:    ps.MarginTop,
		MarginRight:  ps.MarginRight,
		MarginBottom: ps.MarginBottom,
		MarginLeft:   ps.MarginLeft,
		Header:       header,
		Footer:       footer,
	}
	pl.Pages = append(pl.Pages, page)
}

// boxOuterHeightWithStyle computes the full outer height of a box including
// padding, border, and margin.
func boxOuterHeightWithStyle(box *Box) float64 {
	s := box.Style
	return box.Height +
		s.PaddingTop + s.PaddingBottom +
		s.BorderTopWidth + s.BorderBottomWidth +
		s.MarginTop + s.MarginBottom
}

// BuildHeaderFooter extracts page header and footer from the document.
func BuildHeaderFooter(doc *parser.Document, styles map[*parser.Node]style.ComputedStyle) (header, footer *Box) {
	if doc == nil || doc.Root == nil {
		return nil, nil
	}

	body := parser.FindElement(doc.Root, "body")
	if body == nil {
		return nil, nil
	}

	var headerNode, footerNode *parser.Node
	for _, child := range body.Children {
		if child.Type == parser.ElementNode {
			switch child.Tag {
			case "page-header":
				headerNode = child
			case "page-footer":
				footerNode = child
			}
		}
	}

	if headerNode != nil {
		cs := styles[headerNode]
		header = &Box{Type: BlockBox, Node: headerNode, Style: cs}
		for _, child := range headerNode.Children {
			buildNode(child, header, styles)
		}
	}

	if footerNode != nil {
		cs := styles[footerNode]
		footer = &Box{Type: BlockBox, Node: footerNode, Style: cs}
		for _, child := range footerNode.Children {
			buildNode(child, footer, styles)
		}
	}

	return header, footer
}

// BuildFirstPageHeaderFooter extracts first-page-only header/footer from the document.
func BuildFirstPageHeaderFooter(doc *parser.Document, styles map[*parser.Node]style.ComputedStyle) (header, footer *Box) {
	if doc == nil || doc.Root == nil {
		return nil, nil
	}

	body := parser.FindElement(doc.Root, "body")
	if body == nil {
		return nil, nil
	}

	var headerNode, footerNode *parser.Node
	for _, child := range body.Children {
		if child.Type == parser.ElementNode {
			switch child.Tag {
			case "first-header":
				headerNode = child
			case "first-footer":
				footerNode = child
			}
		}
	}

	if headerNode != nil {
		cs := styles[headerNode]
		header = &Box{Type: BlockBox, Node: headerNode, Style: cs}
		for _, child := range headerNode.Children {
			buildNode(child, header, styles)
		}
	}

	if footerNode != nil {
		cs := styles[footerNode]
		footer = &Box{Type: BlockBox, Node: footerNode, Style: cs}
		for _, child := range footerNode.Children {
			buildNode(child, footer, styles)
		}
	}

	return header, footer
}
