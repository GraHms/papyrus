# CLAUDE.md — PDFML Development Guide

## Project Identity

**pdfml** is a pure Go library that converts an XML+CSS document markup language into PDF files. No Chromium, no wkhtmltopdf, no CGO, no external binaries.

The vocabulary is a strict, paged-document subset of HTML. Developers who know HTML can write documents immediately — the same elements (`<html>`, `<head>`, `<body>`, `<header>`, `<footer>`, `<div>`, `<p>`, `<table>`, etc.) are valid here.

Read `SPEC.md` for the full specification before making any architectural decisions.

## Core Invariants (never violate these)

1. **Zero external binary dependencies.** The library must work with `go build` alone. No system packages, no C libraries, no shelling out to CLI tools.
2. **HTML-compatible vocabulary.** The allowed element set is a strict subset of HTML. `<html>` is a valid alias for `<document>`; `<header>` and `<footer>` (direct children of `<body>`) are aliases for `<page-header>` and `<page-footer>`. Unknown elements produce a clear parse error with line and column numbers. Never silently ignore unknown markup.
3. **Strict CSS subset.** Only properties defined in SPEC.md are valid. Unknown properties produce warnings (not errors) but are never applied.
4. **Page-first layout.** Everything is rendered onto fixed-size pages. There is no concept of an infinite scrollable canvas. Page dimensions are always known before layout begins.
5. **Deterministic output.** The same input must always produce the same PDF bytes. No timestamps in PDF metadata unless explicitly set by the user. No random IDs.

## Architecture Overview

The pipeline has four stages. Each stage has its own package under `pkg/`:

```
XML+CSS Input → parser/ → style/ → layout/ → render/ → PDF Output
```

- `parser/`: Tokenize and parse XML into a DOM tree, parse CSS into rule sets
- `style/`: Match selectors to DOM nodes, resolve cascade/specificity/inheritance, produce ComputedStyle per node
- `layout/`: Convert styled DOM into a box tree, run block/inline/table layout, paginate
- `render/`: Walk the page tree and emit PDF primitives

Each package exposes clean interfaces. No package should reach into the internals of another.

## Code Style

- Standard `gofmt` formatting
- All exported types and functions have doc comments
- Error messages include context: `fmt.Errorf("css: unknown property %q at line %d: %w", name, line, err)`
- No `panic()` in library code. All errors returned via `error` interface.
- Use `io.Reader` and `io.Writer` interfaces for input/output where possible
- Test files go in `*_test.go` alongside the code they test
- Table-driven tests preferred
- Benchmark critical paths (layout, text measurement, PDF emission)

## Package Conventions

### parser/
- `nodes.go` defines the DOM types: `Document`, `Element`, `TextNode`, `Attribute`
- `nodes.go` also defines `AllowedElements` (the full vocabulary) and `HTMLAliases` (e.g. `"html" → "document"`, `"header" → "page-header"`)
- `xml.go` parses XML into the DOM using `encoding/xml`; alias normalisation happens here during parsing — by the time the DOM reaches other packages, only canonical tag names exist
- `css.go` parses CSS into `[]Rule` where `Rule = { Selector, Declarations []Declaration }`
- `validation.go` checks elements/attributes against the allowed schema (after alias normalisation)
- All nodes carry `Line` and `Col` for error reporting

### style/
- `selectors.go` implements selector parsing and matching
- `resolver.go` implements the cascade: collect rules, sort by specificity, apply
- `computed.go` defines `ComputedStyle` — a flat struct with all resolved property values
- `properties.go` defines defaults and which properties inherit
- `units.go` handles unit parsing (`12pt`, `1.5em`, `50%`) and conversion to points

### layout/
- `tree.go` converts styled DOM into box tree (BlockBox, InlineBox, TableBox, etc.)
- `block.go` runs block formatting context: width resolution → child layout → height resolution
- `inline.go` runs inline formatting: text shaping, line breaking, baseline alignment
- `table.go` runs table layout algorithm (auto or fixed)
- `page.go` handles pagination: split boxes across pages, inject headers/footers
- `box.go` defines the box model: margin, border, padding, content rect

### render/
- `pdf.go` is the main renderer: walks page tree, calls drawing functions
- `text.go` handles text drawing and font metrics
- `fonts.go` handles TTF loading, embedding, glyph measurement
- `image.go` handles image loading and embedding
- `draw.go` handles rectangles, lines, backgrounds, borders

### document/
- `document.go` is the public API: `doc := pdfml.NewDocument(reader, options); doc.Generate(writer)`
- `options.go` defines configuration: DPI, default fonts, debug mode, etc.

## Testing Strategy

Each package has unit tests. Integration tests live in `testdata/`:

```
testdata/
├── basic/
│   ├── input.xml
│   └── expected.pdf  (or expected_pages.txt for layout verification)
├── invoice/
│   ├── input.xml
│   ├── data.json
│   └── expected.pdf
└── ...
```

For layout verification, prefer text-based golden files that describe the box tree rather than pixel-comparing PDFs. Example:

```
Page 1 (595.28pt x 841.89pt)
  Block [0, 0, 555.28, 841.89] margin=(20mm)
    Block h1 [0, 0, 555.28, 30pt]
      Line "Invoice #1042" font=Liberation Sans 22pt color=#e60000
    Block p [0, 38pt, 555.28, 14pt]
      Line "Date: 2026-03-21" font=Liberation Sans 10pt
```

## Implementation Order

Follow the milestones in SPEC.md. Within each milestone, build bottom-up:

1. Write the types/interfaces first
2. Write tests for expected behavior
3. Implement the logic
4. Wire it into the pipeline
5. Write an integration test with a real XML document

For M0 specifically:
1. `parser/nodes.go` — DOM types
2. `parser/xml.go` — XML parsing
3. `parser/css.go` — CSS parsing (start with just element selectors and 5 properties)
4. `style/properties.go` — Property definitions
5. `style/selectors.go` — Selector matching
6. `style/resolver.go` — Basic cascade
7. `style/computed.go` — ComputedStyle
8. `layout/box.go` — Box model
9. `layout/block.go` — Block layout (single page)
10. `render/pdf.go` — PDF emission with gopdf
11. `render/text.go` — Text drawing
12. `document/document.go` — Wire it all together
13. `cmd/pdfml/main.go` — CLI

## Common Pitfalls to Avoid

- **Do not** try to support all of CSS. The subset is the feature, not the limitation.
- **Do not** use `float64` for pixel coordinates and then compare with `==`. Use an epsilon or fixed-point.
- **Do not** hardcode A4 dimensions anywhere except as a default. Page size must be configurable.
- **Do not** embed font metrics as constants. Read them from the TTF file at runtime.
- **Do not** load the entire document into memory if streaming is possible (though for v0.1, full DOM in memory is acceptable).
- **Do not** mix layout units. Everything in the layout engine works in points (1pt = 1/72 inch). Convert units at the style resolution boundary.
- **Do not** expose HTML alias tags to downstream packages. Alias normalisation (`html` → `document`, `header` → `page-header`, etc.) must happen inside `parser/xml.go` at parse time. All other packages only ever see canonical tag names.
- **Do not** use CSS selectors that match alias tag names. A developer writing `header { ... }` in CSS should target `page-header` content — the alias normalisation means `header` elements are stored as `page-header` in the DOM, so `header` selectors will never match. Document this clearly in error messages if needed.

## CSS Parser Notes

Do NOT use a regex-based CSS parser. Write a proper tokenizer that handles:
- Comments (`/* ... */`)
- String literals (both `'` and `"`)
- Numbers with units (`12pt`, `1.5em`, `50%`)
- Hash colors (`#fff`, `#e60000`)
- Function notation (`rgb(255, 0, 0)`) — only `rgb()` and `rgba()` for v0.1
- Selector combinators (space for descendant, `>` for child)

The CSS parser should produce structured output, not string soup.

## PDF Backend Choice

Use `github.com/signintech/gopdf` as the PDF backend. Reasons:
- Actively maintained
- Simpler API than gofpdf (which is archived)
- Good TTF font support
- Supports basic drawing primitives we need

If gopdf proves insufficient, `github.com/pdfcpu/pdfcpu` can be used for post-processing (merging, metadata).

## Font Strategy

Bundle Liberation Sans (Regular, Bold, Italic, Bold-Italic) as the default font family via `go:embed`. This provides a baseline that works without any external font files.

Users can specify additional fonts via:
```xml
<head>
  <font name="CustomFont" src="path/to/font.ttf" />
</head>
```

Or via Go API:
```go
opts := pdfml.Options{
    Fonts: map[string]string{
        "Custom Font": "/path/to/custom.ttf",
    },
}
```

## Debug Mode

When debug mode is enabled:
- Draw red outlines around every box
- Draw blue baselines for text
- Print the box tree to stderr
- Add comments in the PDF structure (if possible)

This is essential for development. Implement it early and keep it working.

## Performance Targets

- Parse + layout + render a 5-page invoice: < 100ms
- Parse + layout + render a 50-page report: < 1s
- Memory: < 50MB for a 100-page document

Benchmark from M0 onward. Use `testing.B` benchmarks in each package.

## CLI Interface

```
pdfml [flags] <input.xml> [output.pdf]

Flags:
  -data <file.json>     JSON data for template interpolation
  -font <name=path>     Register additional font (repeatable)
  -watch                Watch input file and regenerate on change
  -debug                Enable debug mode (box outlines, verbose output)
  -page-size <size>     Override page size (A4, letter, legal, WxH)
  -dpi <n>              Set DPI for px unit conversion (default: 96)
  -o <output.pdf>       Output file (default: input name with .pdf extension)
```

## Library API

```go
package pdfml

// Generate reads an XML document and writes PDF to the writer.
func Generate(r io.Reader, w io.Writer, opts ...Option) error

// GenerateFromFile reads an XML file and writes a PDF file.
func GenerateFromFile(inputPath, outputPath string, opts ...Option) error

// Document represents a parsed document ready for rendering.
type Document struct { ... }

// Parse reads and validates an XML document.
func Parse(r io.Reader) (*Document, error)

// Render generates PDF output from a parsed document.
func (d *Document) Render(w io.Writer, opts ...Option) error
```

## Commit Message Convention

Use conventional commits:
- `feat(parser): implement CSS class selector matching`
- `fix(layout): correct margin collapsing between siblings`
- `test(render): add integration test for multi-page invoice`
- `refactor(style): extract unit conversion to separate file`
- `docs: update SPEC.md with list element definitions`
