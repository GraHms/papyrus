// Package document provides the top-level API for pdfml.
//
// Usage:
//
//	err := pdfml.GenerateFromFile("invoice.xml", "invoice.pdf")
//
// Or with options:
//
//	f, _ := os.Open("invoice.xml")
//	out, _ := os.Create("invoice.pdf")
//	err := pdfml.Generate(f, out, pdfml.WithDebug(), pdfml.WithDPI(150))
package document

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/grahms/pdfml/pkg/layout"
	"github.com/grahms/pdfml/pkg/parser"
	"github.com/grahms/pdfml/pkg/render"
	"github.com/grahms/pdfml/pkg/style"
)

// Document represents a parsed and validated pdfml document
// that is ready to be rendered to PDF.
type Document struct {
	options    Options
	parsed     *parser.Document
	cssRules   []parser.Rule
	styles     map[*parser.Node]style.ComputedStyle
	pageLayout *layout.PageLayout
	resolver   *style.ResolverContext
}

// Generate reads an XML document from r, processes it through the full
// pipeline (parse → style → layout → render), and writes PDF to w.
func Generate(r io.Reader, w io.Writer, opts ...Option) error {
	doc, err := Parse(r)
	if err != nil {
		return err
	}
	return doc.Render(w, opts...)
}

// GenerateFromFile is a convenience function that reads from inputPath
// and writes to outputPath.
func GenerateFromFile(inputPath, outputPath string, opts ...Option) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("pdfml: cannot open %q: %w", inputPath, err)
	}
	defer f.Close()

	// Add base path option so relative file paths (images) resolve correctly
	basePath := filepath.Dir(inputPath)
	opts = append(opts, func(o *Options) {
		if o.Fonts == nil {
			o.Fonts = make(map[string]string)
		}
		if o.FontsBytes == nil {
			o.FontsBytes = make(map[string][]byte)
		}
		o.BasePath = basePath
	})

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("pdfml: cannot create %q: %w", outputPath, err)
	}
	defer out.Close()

	return Generate(f, out, opts...)
}

// Parse reads and validates an XML document without rendering it.
// Useful for validation or inspection, or for reusing a compiled template.
func Parse(r io.Reader) (*Document, error) {
	doc, err := parser.ParseXML(r)
	if err != nil {
		return nil, fmt.Errorf("pdfml: parse error: %w", err)
	}
	if err := parser.ValidateDocument(doc); err != nil {
		return nil, fmt.Errorf("pdfml: validation error: %w", err)
	}

	// Parse CSS from <style> blocks
	var rules []parser.Rule
	if doc.Styles != "" {
		rules, err = parser.ParseCSS(doc.Styles)
		if err != nil {
			return nil, fmt.Errorf("pdfml: CSS parse error: %w", err)
		}
	}

	return &Document{
		options:  defaultOptions(),
		parsed:   doc,
		cssRules: rules,
	}, nil
}

// Render generates PDF from an already-parsed document using the provided options.
func (d *Document) Render(w io.Writer, opts ...Option) error {
	if d.parsed == nil {
		return fmt.Errorf("pdfml: document not parsed")
	}

	options := d.options
	for _, opt := range opts {
		opt(&options)
	}

	// Phase 2 — Resolve styles
	resolver := style.NewResolver(d.cssRules, options.DPI)

	// Override page size if specified in options
	if options.PageSize != "" {
		w2, h := render.PageSizeFromString(options.PageSize)
		resolver.PageStyle.Width = w2
		resolver.PageStyle.Height = h
	}

	// Set up measurement before layout
	measure, cleanup, err := render.MeasureForLayout(options.Fonts, options.FontsBytes)
	if err != nil {
		return fmt.Errorf("pdfml: font initialization error: %w", err)
	}
	defer cleanup()

	// Resolve all node styles
	nodeStyles := resolver.ResolveTree(d.parsed.Root)

	// Phase 3 — Layout
	ctx := &layout.Context{
		PageWidth:  resolver.PageStyle.Width - resolver.PageStyle.MarginLeft - resolver.PageStyle.MarginRight,
		PageHeight: resolver.PageStyle.Height - resolver.PageStyle.MarginTop - resolver.PageStyle.MarginBottom,
		DPI:        options.DPI,
		Measure:    measure,
	}

	// Build box tree from body
	rootBox := layout.BuildBoxTree(d.parsed, nodeStyles)
	if rootBox == nil {
		return fmt.Errorf("pdfml: no body element found")
	}

	// Build header/footer boxes
	headerBox, footerBox := layout.BuildHeaderFooter(d.parsed, nodeStyles)

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
		return fmt.Errorf("pdfml: layout produced no pages")
	}

	// Phase 4 — Render to PDF
	renderOpts := render.Options{
		Debug:    options.Debug,
		BasePath: options.BasePath,
		Meta:     d.parsed.Meta,
	}

	renderer := render.NewRenderer(pageLayout.Pages, renderOpts, options.Fonts, options.FontsBytes)
	return renderer.Render(w)
}

// GenerateFromBytes resolves PDF generation directly from a byte slice.
func GenerateFromBytes(data []byte, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	if err := Generate(bytes.NewReader(data), &buf, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateFromString resolves PDF generation directly from a string.
func GenerateFromString(data string, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	if err := Generate(strings.NewReader(data), &buf, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
