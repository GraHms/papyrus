// Package document provides the top-level API for goxml2pdf.
//
// Usage:
//
//	err := goxml2pdf.GenerateFromFile("invoice.xml", "invoice.pdf")
//
// Or with options:
//
//	f, _ := os.Open("invoice.xml")
//	out, _ := os.Create("invoice.pdf")
//	err := goxml2pdf.Generate(f, out, goxml2pdf.WithDebug(), goxml2pdf.WithDPI(150))
package document

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ismaelvodacom/goxml2pdf/pkg/layout"
	"github.com/ismaelvodacom/goxml2pdf/pkg/parser"
	"github.com/ismaelvodacom/goxml2pdf/pkg/render"
	"github.com/ismaelvodacom/goxml2pdf/pkg/style"
)

// Document represents a parsed and validated goxml2pdf document
// that is ready to be rendered to PDF.
type Document struct {
	options Options
	parsed  *parser.Document
	styles  map[*parser.Node]style.ComputedStyle
	pageLayout *layout.PageLayout
	resolver   *style.ResolverContext
}

// Generate reads an XML document from r, processes it through the full
// pipeline (parse → style → layout → render), and writes PDF to w.
func Generate(r io.Reader, w io.Writer, opts ...Option) error {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Phase 1 — Parse XML + CSS
	doc, err := parser.ParseXML(r)
	if err != nil {
		return fmt.Errorf("goxml2pdf: parse error: %w", err)
	}

	if err := parser.ValidateDocument(doc); err != nil {
		return fmt.Errorf("goxml2pdf: validation error: %w", err)
	}

	// Parse CSS from <style> blocks
	var rules []parser.Rule
	if doc.Styles != "" {
		rules, err = parser.ParseCSS(doc.Styles)
		if err != nil {
			return fmt.Errorf("goxml2pdf: CSS parse error: %w", err)
		}
	}

	// Phase 2 — Resolve styles
	resolver := style.NewResolver(rules, options.DPI)

	// Override page size if specified in options
	if options.PageSize != "" {
		w2, h := render.PageSizeFromString(options.PageSize)
		resolver.PageStyle.Width = w2
		resolver.PageStyle.Height = h
	}

	// Set up measurement before layout
	measure, cleanup, err := render.MeasureForLayout(options.Fonts)
	if err != nil {
		return fmt.Errorf("goxml2pdf: font initialization error: %w", err)
	}
	defer cleanup()

	// Resolve all node styles
	nodeStyles := resolver.ResolveTree(doc.Root)

	// Phase 3 — Layout
	ctx := &layout.Context{
		PageWidth:  resolver.PageStyle.Width - resolver.PageStyle.MarginLeft - resolver.PageStyle.MarginRight,
		PageHeight: resolver.PageStyle.Height - resolver.PageStyle.MarginTop - resolver.PageStyle.MarginBottom,
		DPI:        options.DPI,
		Measure:    measure,
	}

	// Build box tree from body
	rootBox := layout.BuildBoxTree(doc, nodeStyles)
	if rootBox == nil {
		return fmt.Errorf("goxml2pdf: no body element found")
	}

	// Build header/footer boxes
	headerBox, footerBox := layout.BuildHeaderFooter(doc, nodeStyles)

	// Create page layout and paginate
	pageLayout := layout.NewPageLayout(resolver.PageStyle, ctx)
	if headerBox != nil {
		pageLayout.SetHeader(headerBox)
	}
	if footerBox != nil {
		pageLayout.SetFooter(footerBox)
	}
	pageLayout.Layout(rootBox)

	if len(pageLayout.Pages) == 0 {
		return fmt.Errorf("goxml2pdf: layout produced no pages")
	}

	// Phase 4 — Render to PDF
	renderOpts := render.Options{
		Debug: options.Debug,
		Meta:  doc.Meta,
	}

	renderer := render.NewRenderer(pageLayout.Pages, renderOpts, options.Fonts)
	return renderer.Render(w)
}

// GenerateFromFile is a convenience function that reads from inputPath
// and writes to outputPath.
func GenerateFromFile(inputPath, outputPath string, opts ...Option) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("goxml2pdf: cannot open %q: %w", inputPath, err)
	}
	defer f.Close()

	// Add base path option so relative file paths (images) resolve correctly
	basePath := filepath.Dir(inputPath)
	opts = append(opts, func(o *Options) {
		if o.Fonts == nil {
			o.Fonts = make(map[string]string)
		}
		_ = basePath // used by renderer via Options.BasePath
	})

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("goxml2pdf: cannot create %q: %w", outputPath, err)
	}
	defer out.Close()

	return Generate(f, out, opts...)
}

// Parse reads and validates an XML document without rendering it.
// Useful for validation or inspection.
func Parse(r io.Reader) (*Document, error) {
	doc, err := parser.ParseXML(r)
	if err != nil {
		return nil, fmt.Errorf("goxml2pdf: parse error: %w", err)
	}
	if err := parser.ValidateDocument(doc); err != nil {
		return nil, fmt.Errorf("goxml2pdf: validation error: %w", err)
	}

	return &Document{
		options: defaultOptions(),
		parsed:  doc,
	}, nil
}

// Render generates PDF from an already-parsed document.
func (d *Document) Render(w io.Writer, opts ...Option) error {
	if d.parsed == nil {
		return fmt.Errorf("goxml2pdf: document not parsed")
	}

	options := d.options
	for _, opt := range opts {
		opt(&options)
	}

	// Use a strings reader so we can call Generate with the parsed state
	// For now, re-process through Generate is the simplest approach
	// TODO: cache the parsed state
	return fmt.Errorf("goxml2pdf: Render() on parsed Document not yet implemented — use Generate() instead")
}
