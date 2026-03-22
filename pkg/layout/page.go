package layout

import (
	"github.com/ismaelvodacom/goxml2pdf/pkg/parser"
	"github.com/ismaelvodacom/goxml2pdf/pkg/style"
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
		pl.headerHeight = pl.headerBox.Height +
			pl.headerBox.Style.PaddingTop + pl.headerBox.Style.PaddingBottom +
			pl.headerBox.Style.BorderTopWidth + pl.headerBox.Style.BorderBottomWidth +
			pl.headerBox.Style.MarginTop + pl.headerBox.Style.MarginBottom
	}
	if pl.footerBox != nil {
		pl.footerBox.Width = contentWidth
		layoutBlockChildren(pl.ctx, pl.footerBox, contentWidth)
		pl.footerHeight = pl.footerBox.Height +
			pl.footerBox.Style.PaddingTop + pl.footerBox.Style.PaddingBottom +
			pl.footerBox.Style.BorderTopWidth + pl.footerBox.Style.BorderBottomWidth +
			pl.footerBox.Style.MarginTop + pl.footerBox.Style.MarginBottom
	}

	// Available content height per page
	pageContentH := ps.Height - ps.MarginTop - ps.MarginBottom - pl.headerHeight - pl.footerHeight

	// Paginate: walk body's direct children sequentially
	pl.newPage()
	curPageIdx := 0
	curPageY := 0.0 // cursor Y within the current page content area

	for _, child := range root.Children {
		// Forced page break
		if child.Type == PageBreakBox {
			curPageIdx++
			pl.newPage()
			curPageY = 0
			continue
		}

		childH := child.OuterHeight()

		// Auto-break if child overflows current page (but not if it's too tall to fit on any page)
		if curPageY+childH > pageContentH && curPageY > 0 && childH <= pageContentH {
			curPageIdx++
			pl.newPage()
			curPageY = 0
		}

		cs := child.Style

		// Assign absolute position on the page
		child.Page = curPageIdx
		child.AbsX = ps.MarginLeft + cs.MarginLeft
		child.AbsY = ps.MarginTop + pl.headerHeight + curPageY + cs.MarginTop

		// Only add top-level boxes to page; rendering recurses into children
		pl.Pages[curPageIdx].Boxes = append(pl.Pages[curPageIdx].Boxes, child)

		curPageY += childH
	}

	// Fill in page numbers
	total := len(pl.Pages)
	for i, p := range pl.Pages {
		p.Number = i + 1
		p.Total = total
	}
}

// newPage appends a fresh page to the list.
func (pl *PageLayout) newPage() {
	ps := pl.PageStyle
	page := &Page{
		Width:        ps.Width,
		Height:       ps.Height,
		MarginTop:    ps.MarginTop,
		MarginRight:  ps.MarginRight,
		MarginBottom: ps.MarginBottom,
		MarginLeft:   ps.MarginLeft,
		Header:       pl.headerBox,
		Footer:       pl.footerBox,
	}
	pl.Pages = append(pl.Pages, page)
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
