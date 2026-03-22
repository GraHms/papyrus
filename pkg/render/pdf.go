package render

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/grahms/pdfml/pkg/layout"
	"github.com/grahms/pdfml/pkg/style"
	"github.com/signintech/gopdf"
)

// Options holds rendering options.
type Options struct {
	Debug    bool
	BasePath string // base directory for resolving relative file paths
	Meta     map[string]string
}

// Renderer renders laid-out pages to PDF.
type Renderer struct {
	pdf         *gopdf.GoPdf
	fm          *FontManager
	opts        Options
	pages       []*layout.Page
	measure     layout.TextMeasurer
	currentPage *layout.Page // page currently being rendered (for page-number substitution)
}

// NewRenderer creates a new PDF renderer.
func NewRenderer(pages []*layout.Page, opts Options, extraFonts map[string]string) *Renderer {
	pdf := &gopdf.GoPdf{}
	fm := NewFontManager(pdf)

	for family, path := range extraFonts {
		fm.RegisterFont(family, path)
	}

	return &Renderer{
		pdf:   pdf,
		fm:    fm,
		opts:  opts,
		pages: pages,
	}
}

// Render generates the PDF and writes it to w.
func (r *Renderer) Render(w io.Writer) error {
	if len(r.pages) == 0 {
		return fmt.Errorf("render: no pages to render")
	}

	firstPage := r.pages[0]
	r.pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: firstPage.Width, H: firstPage.Height},
		Unit:     gopdf.Unit_PT,
	})

	if _, err := r.fm.EnsureFont("Liberation Sans", false, false); err != nil {
		return fmt.Errorf("render: failed to load default font: %w", err)
	}

	r.measure = MeasureText(r.pdf, r.fm)

	if r.opts.Meta != nil {
		if title, ok := r.opts.Meta["title"]; ok {
			r.pdf.SetInfo(gopdf.PdfInfo{Title: title})
		}
	}

	for i, page := range r.pages {
		if i > 0 {
			r.pdf.AddPageWithOption(gopdf.PageOption{
				PageSize: &gopdf.Rect{W: page.Width, H: page.Height},
			})
		} else {
			r.pdf.AddPage()
		}
		r.renderPage(page)
	}

	_, err := r.pdf.WriteTo(w)
	return err
}

// renderPage renders all boxes on a single page.
func (r *Renderer) renderPage(page *layout.Page) {
	r.currentPage = page

	contentWidth := page.Width - page.MarginLeft - page.MarginRight

	if page.Header != nil {
		r.renderBox(page.Header, page.MarginLeft, page.MarginTop, contentWidth)
	}

	if page.Footer != nil {
		footerH := page.Footer.Height +
			page.Footer.Style.PaddingTop + page.Footer.Style.PaddingBottom +
			page.Footer.Style.BorderTopWidth + page.Footer.Style.BorderBottomWidth
		footerY := page.Height - page.MarginBottom - footerH
		r.renderBox(page.Footer, page.MarginLeft, footerY, contentWidth)
	}

	for _, box := range page.Boxes {
		r.renderBox(box, box.AbsX, box.AbsY, box.Width)
	}
}

// renderBox renders a single box at the given absolute position.
func (r *Renderer) renderBox(box *layout.Box, absX, absY, availWidth float64) {
	if box == nil {
		return
	}

	cs := box.Style
	// box.Width is the border-box width (content + padding + border); box.Height is content-only.
	borderBoxW := box.Width
	borderBoxH := box.Height + cs.PaddingTop + cs.PaddingBottom + cs.BorderTopWidth + cs.BorderBottomWidth

	if cs.BackgroundColor.A > 0 {
		drawBackground(r.pdf, absX, absY, borderBoxW, borderBoxH, cs.BackgroundColor)
	}

	// For cells inside a border-collapse:collapse table, borders are drawn
	// by the table-level grid pass — skip individual cell borders here.
	if !box.InCollapsedTable {
		drawBorders(r.pdf, absX, absY, borderBoxW, borderBoxH, cs)
	}

	contentX := absX + cs.BorderLeftWidth + cs.PaddingLeft
	contentY := absY + cs.BorderTopWidth + cs.PaddingTop

	if r.opts.Debug {
		drawDebugOutline(r.pdf, absX, absY, borderBoxW, borderBoxH, style.Color{R: 255, G: 0, B: 0, A: 255})
	}

	// Content width = border-box width minus left/right padding and border
	contentWidth := box.Width - cs.PaddingLeft - cs.PaddingRight - cs.BorderLeftWidth - cs.BorderRightWidth
	if contentWidth < 0 {
		contentWidth = 0
	}

	switch box.Type {
	case layout.TableBox:
		// Render table children first, then draw collapsed-border grid on top.
		r.renderBlockContent(box, contentX, contentY, contentWidth)
		if box.Style.BorderCollapse == "collapse" && len(box.ColXPositions) > 0 {
			drawCollapsedTableBorders(r.pdf, contentX, contentY, box.ColXPositions, box.RowYPositions, box.Style)
		}
		return

	case layout.HRBox:
		drawHR(r.pdf, contentX, contentY, contentWidth, cs)

	case layout.ImageBox:
		basePath := r.opts.BasePath
		if basePath == "" {
			basePath = "."
		}
		imgW := box.ImgWidth
		imgH := box.ImgHeight
		if imgW <= 0 {
			imgW = contentWidth
		}
		if imgH <= 0 {
			imgH = box.Height
		}
		if err := drawImage(r.pdf, box.ImageSrc, contentX, contentY, imgW, imgH, basePath); err != nil {
			r.drawImagePlaceholder(contentX, contentY, imgW, imgH, box.ImageSrc)
		}

	case layout.PageBreakBox:
		// nothing

	case layout.TextBox:
		// rendered via parent's InlineRuns

	default:
		// Draw list marker (bullet or number) before content
		if box.ListMarker != "" {
			r.drawListMarker(box.ListMarker, absX-cs.PaddingLeft, absY+cs.BorderTopWidth+cs.PaddingTop, cs)
		}
		r.renderBlockContent(box, contentX, contentY, contentWidth)
	}
}

// renderBlockContent renders inline runs or block children.
func (r *Renderer) renderBlockContent(box *layout.Box, x, y, width float64) {
	if len(box.InlineRuns) > 0 {
		r.renderInlineRuns(box.InlineRuns, x, y, width, box.Style)
		return
	}
	for _, child := range box.Children {
		if child == nil {
			continue
		}
		r.renderBox(child, x+child.X, y+child.Y, child.Width)
	}
}

// renderInlineRuns draws inline content of a box.
func (r *Renderer) renderInlineRuns(runs []layout.InlineRun, x, y, width float64, cs style.ComputedStyle) {
	var contentRuns []layout.InlineRun
	for _, run := range runs {
		// Substitute page-number/page-count markers (keep \n markers intact — they encode layout line breaks)
		text := r.substitutePageMarkers(run.Text)
		if text != run.Text {
			run = layout.InlineRun{Text: text, Style: run.Style, Node: run.Node}
		}
		contentRuns = append(contentRuns, run)
	}
	if len(contentRuns) == 0 {
		return
	}
	drawTextRuns(r.pdf, r.fm, contentRuns, x+cs.TextIndent, y, width-cs.TextIndent, cs)
}

// substitutePageMarkers replaces {{PAGE}} and {{PAGES}} with actual values.
func (r *Renderer) substitutePageMarkers(text string) string {
	if r.currentPage == nil {
		return text
	}
	pageNum := fmt.Sprintf("%d", r.currentPage.Number)
	pageTotal := fmt.Sprintf("%d", r.currentPage.Total)
	text = strings.ReplaceAll(text, "{{PAGE}}", pageNum)
	text = strings.ReplaceAll(text, "{{PAGES}}", pageTotal)
	return text
}

// drawImagePlaceholder draws a placeholder when an image cannot be loaded.
func (r *Renderer) drawImagePlaceholder(x, y, w, h float64, src string) {
	drawBackground(r.pdf, x, y, w, h, style.Color{R: 220, G: 220, B: 220, A: 255})
	if err := r.fm.SetFont("Liberation Sans", false, false, 7); err == nil {
		setTextColor(r.pdf, style.Color{R: 120, G: 120, B: 120, A: 255})
		r.pdf.SetXY(x+2, y+h/2-3.5)
		label := filepath.Base(src)
		if len(label) > 20 {
			label = label[:20] + "..."
		}
		_ = r.pdf.Text(label)
	}
}

// drawListMarker renders a bullet or ordered list marker to the left of the list item.
func (r *Renderer) drawListMarker(marker string, x, y float64, cs style.ComputedStyle) {
	if err := r.fm.SetFont(cs.FontFamily, cs.IsBold(), cs.IsItalic(), cs.FontSize); err != nil {
		return
	}
	setTextColor(r.pdf, cs.Color)
	// Measure marker width to right-align it just before content
	mw, _ := r.pdf.MeasureTextWidth(marker)
	markerX := x - mw - 2 // 2pt gap
	ascent := cs.FontSize * 0.75
	lineH := cs.LineHeight
	if lineH <= 0 {
		lineH = cs.FontSize * 1.2
	}
	r.pdf.SetXY(markerX, y+lineH-ascent-cs.FontSize*0.15)
	_ = r.pdf.Text(marker)
}

// MeasureForLayout returns a TextMeasurer for use during layout.
func MeasureForLayout(extraFonts map[string]string) (layout.TextMeasurer, func(), error) {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: 595.28, H: 841.89},
		Unit:     gopdf.Unit_PT,
	})
	pdf.AddPage()

	fm := NewFontManager(pdf)
	for family, path := range extraFonts {
		fm.RegisterFont(family, path)
	}

	if _, err := fm.EnsureFont("Liberation Sans", false, false); err != nil {
		return nil, nil, fmt.Errorf("render: failed to load default font for measurement: %w", err)
	}

	return MeasureText(pdf, fm), func() {}, nil
}

// PageSizeFromString parses a page size string and returns (width, height) in pt.
func PageSizeFromString(size string) (float64, float64) {
	switch strings.ToUpper(size) {
	case "A4":
		return 595.28, 841.89
	case "A4 LANDSCAPE":
		return 841.89, 595.28
	case "LETTER":
		return 612, 792
	case "LEGAL":
		return 612, 1008
	case "A3":
		return 841.89, 1190.55
	case "A5":
		return 419.53, 595.28
	}
	return 595.28, 841.89
}
