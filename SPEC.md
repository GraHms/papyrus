# GoXML2PDF — Project Specification

## Vision

A pure Go library and CLI tool that converts a custom XML+CSS document language into PDF.
No Chromium. No wkhtmltopdf. No external binaries. Zero CGO dependencies.

The XML vocabulary is purpose-built for paged documents (invoices, reports, contracts, certificates, statements).
CSS is a strict subset scoped to properties that make sense for fixed-page layout.

## Name Candidates

- `goxml2pdf`
- `pagego`
- `docxml` 
- `pdfml`

(Decision deferred — use `goxml2pdf` as working name)

## Design Principles

1. **Paged-first**: Every element exists in the context of a physical page with known dimensions.
2. **Strict subset**: We define exactly which XML elements and CSS properties are supported. Anything outside the subset is a parse error, not a silent degradation.
3. **Declarative only**: No scripting, no JavaScript, no dynamic behavior. The document is a static description of content and style.
4. **Composable**: Templates, partials, and variables are first-class so documents can be generated from data.
5. **Fast**: Must be suitable for server-side PDF generation at scale (target: <100ms for a 5-page invoice on commodity hardware).

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    INPUT LAYER                       │
│                                                      │
│  XML Document String/File ──► XML Parser             │
│  CSS String/Embedded     ──► CSS Parser              │
│  Data (JSON/Go struct)   ──► Template Engine          │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│                  RESOLUTION LAYER                     │
│                                                      │
│  Style Resolver: Match CSS selectors to XML nodes    │
│  Cascade + Specificity + Inheritance                 │
│  Computed Styles per node                            │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│                  LAYOUT ENGINE                        │
│                                                      │
│  Box Model Calculator                                │
│  Block Layout (vertical stacking)                    │
│  Inline Layout (text lines, wrapping)                │
│  Table Layout                                        │
│  Page Breaking / Pagination                          │
│  Running Headers & Footers                           │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│                  RENDER LAYER                         │
│                                                      │
│  PDF Emitter (via gopdf or gofpdf fork)              │
│  Text rendering + font metrics                       │
│  Image embedding                                     │
│  Vector drawing (lines, borders, backgrounds)        │
│  Metadata (title, author, keywords)                  │
└─────────────────────────────────────────────────────┘
```

## Pipeline Phases

### Phase 1: Parse
- Parse XML using `encoding/xml` into a DOM-like tree (`Node` structs)
- Parse CSS into a list of `Rule` structs (selector + declarations)
- Validate both against the supported subset schema

### Phase 2: Style Resolution
- Walk the DOM tree
- For each node, collect all matching CSS rules
- Apply cascade: specificity, source order, inheritance
- Produce a `ComputedStyle` struct per node with all resolved values

### Phase 3: Layout
- Convert styled DOM into a Box Tree (block boxes, inline boxes, table boxes)
- Run block layout: compute widths, then heights, stack vertically
- Run inline layout: break text into lines (greedy or Knuth-Plass)
- Run table layout: auto or fixed column widths
- Run pagination: split box tree across pages, handle orphans/widows, repeat table headers
- Inject running headers/footers per page

### Phase 4: Render
- Walk the laid-out page tree
- For each box: draw background, borders, text, images
- Emit PDF primitives via the chosen PDF backend

## XML Element Vocabulary (v0.1)

### Document Structure
| Element | Description |
|---|---|
| `<document>` | Root element |
| `<head>` | Contains `<meta>`, `<style>`, `<var>` |
| `<body>` | Contains all visible content |
| `<style>` | Embedded CSS |
| `<meta>` | Document metadata (title, author, subject, keywords) |
| `<var>` | Template variable definition |

### Page Control
| Element | Description |
|---|---|
| `<page-header>` | Repeated at top of each page |
| `<page-footer>` | Repeated at bottom of each page |
| `<page-break/>` | Force a new page |
| `<page-number/>` | Insert current page number |
| `<page-count/>` | Insert total page count |

### Block Content
| Element | Description |
|---|---|
| `<section>` | Generic block container |
| `<div>` | Generic block container (alias) |
| `<h1>` ... `<h6>` | Headings |
| `<p>` | Paragraph |
| `<blockquote>` | Indented block |
| `<hr/>` | Horizontal rule |

### Inline Content
| Element | Description |
|---|---|
| `<span>` | Generic inline container |
| `<strong>` / `<b>` | Bold text |
| `<em>` / `<i>` | Italic text |
| `<u>` | Underlined text |
| `<code>` | Monospace text |
| `<br/>` | Line break |
| `<a>` | Hyperlink (href becomes PDF link annotation) |

### Tables
| Element | Description |
|---|---|
| `<table>` | Table container |
| `<thead>` | Header group (repeats on page breaks) |
| `<tbody>` | Body group |
| `<tfoot>` | Footer group |
| `<tr>` | Table row |
| `<td>` | Table cell |
| `<th>` | Header cell (bold, centered by default) |

### Media
| Element | Description |
|---|---|
| `<img>` | Image (src: file path or base64 data URI) |
| `<svg>` | Inline SVG (future: v0.2+) |

### Lists
| Element | Description |
|---|---|
| `<ul>` | Unordered list |
| `<ol>` | Ordered list |
| `<li>` | List item |

### Template / Data Binding (v0.2+)
| Element | Description |
|---|---|
| `<for-each>` | Loop over data array |
| `<if>` | Conditional rendering |
| `<slot>` | Named insertion point |
| `<include>` | Include external XML partial |
| `{{var}}` | Interpolate variable value |

## CSS Subset (v0.1)

### Selectors Supported
- Element: `p`, `h1`, `table`
- Class: `.invoice-header`
- ID: `#total`
- Descendant: `table td`
- Child: `table > tr`
- Pseudo-class: `:first-child`, `:last-child`, `:nth-child(n)`
- NOT supported: pseudo-elements, attribute selectors, sibling combinators

### Properties Supported

**Page**
- `size` (A4, letter, legal, or explicit WxH)
- `margin` (page margins)
- `orientation` (portrait, landscape)

**Box Model**
- `margin`, `margin-top/right/bottom/left`
- `padding`, `padding-top/right/bottom/left`
- `border`, `border-width`, `border-style`, `border-color`, `border-top/right/bottom/left`
- `width`, `min-width`, `max-width`
- `height`, `min-height`, `max-height`

**Typography**
- `font-family` (mapped to embedded TTF/OTF fonts)
- `font-size` (pt, px, mm, cm, em, rem)
- `font-weight` (normal, bold, 100-900)
- `font-style` (normal, italic)
- `text-align` (left, right, center, justify)
- `text-decoration` (none, underline, line-through)
- `line-height`
- `letter-spacing`
- `text-transform` (uppercase, lowercase, capitalize)
- `white-space` (normal, nowrap, pre)
- `text-indent`

**Colors & Backgrounds**
- `color`
- `background-color`
- `opacity`

**Layout**
- `display` (block, inline, table, table-row, table-cell, none)
- `vertical-align` (for table cells: top, middle, bottom)
- `overflow` (hidden, visible — for clipping)

**Table**
- `border-collapse` (collapse, separate)
- `border-spacing`
- `table-layout` (auto, fixed)

**Page Break**
- `page-break-before` (auto, always, avoid)
- `page-break-after` (auto, always, avoid)
- `page-break-inside` (auto, avoid)
- `orphans`
- `widows`

**NOT Supported (explicit exclusion)**
- `float`, `position`, `flexbox`, `grid`
- `transform`, `transition`, `animation`
- `z-index`, `overflow-x/y`
- Any pseudo-element styling (`::before`, `::after`)

## Units

| Unit | Description |
|---|---|
| `pt` | Points (1/72 inch) — default for fonts |
| `px` | Pixels (mapped to 1pt for simplicity, or configurable DPI) |
| `mm` | Millimeters |
| `cm` | Centimeters |
| `in` | Inches |
| `em` | Relative to current font-size |
| `rem` | Relative to root font-size |
| `%` | Percentage of parent dimension |

## Go Package Structure

```
goxml2pdf/
├── cmd/
│   └── goxml2pdf/
│       └── main.go                 # CLI entry point
├── pkg/
│   ├── parser/
│   │   ├── xml.go                  # XML DOM parser
│   │   ├── css.go                  # CSS tokenizer + parser
│   │   ├── nodes.go                # DOM node types
│   │   └── validation.go           # Schema validation
│   ├── style/
│   │   ├── resolver.go             # CSS cascade + specificity
│   │   ├── computed.go             # ComputedStyle struct
│   │   ├── properties.go           # Property definitions + defaults
│   │   ├── selectors.go            # Selector matching engine
│   │   ├── inheritance.go          # Property inheritance rules
│   │   └── units.go                # Unit parsing + conversion
│   ├── layout/
│   │   ├── box.go                  # Box model (margin, border, padding, content)
│   │   ├── block.go                # Block layout algorithm
│   │   ├── inline.go               # Inline/text layout + line breaking
│   │   ├── table.go                # Table layout algorithm
│   │   ├── page.go                 # Pagination + page management
│   │   └── tree.go                 # Box tree construction from styled DOM
│   ├── render/
│   │   ├── pdf.go                  # PDF rendering backend
│   │   ├── text.go                 # Text drawing + font metrics
│   │   ├── image.go                # Image embedding
│   │   ├── draw.go                 # Lines, rects, backgrounds
│   │   └── fonts.go                # Font loading + management
│   ├── template/
│   │   ├── engine.go               # Variable interpolation
│   │   ├── loops.go                # for-each processing
│   │   └── conditionals.go         # if/else processing
│   └── document/
│       ├── document.go             # Top-level Document type + Generate()
│       └── options.go              # Configuration options
├── fonts/                          # Bundled default fonts (embedded via go:embed)
│   ├── liberation-sans/
│   └── liberation-serif/
├── testdata/                       # Test XML documents + expected outputs
├── examples/                       # Example documents
│   ├── invoice.xml
│   ├── report.xml
│   └── certificate.xml
├── go.mod
├── go.sum
├── CLAUDE.md                       # AI development prompt
├── SPEC.md                         # This file
└── README.md
```

## Milestone Plan

### M0: Foundation (Week 1-2)
- [ ] XML parser: `<document>`, `<head>`, `<body>`, `<style>`, block elements (`<h1>`..`<h6>`, `<p>`, `<div>`, `<section>`)
- [ ] CSS parser: element/class/ID selectors, core properties (font-size, color, margin, padding, text-align)
- [ ] Style resolver: basic cascade (no specificity tiebreaking yet)
- [ ] Block layout: vertical stacking, width/height, margin/padding
- [ ] PDF render: single page, text + backgrounds
- [ ] **Deliverable**: Render a simple one-page document with headings, paragraphs, and basic styling

### M1: Text & Inline (Week 3-4)
- [ ] Inline layout: text wrapping, line breaking, mixed bold/italic spans
- [ ] Font management: TTF loading, metrics, embedded fonts
- [ ] Units: full unit support (pt, mm, em, %, etc.)
- [ ] CSS specificity: proper cascade ordering
- [ ] **Deliverable**: Render a document with mixed inline formatting and proper text reflow

### M2: Tables (Week 5-6)
- [ ] Table layout: auto column widths, fixed layout, border-collapse
- [ ] `<thead>` repeat on page breaks
- [ ] Cell spanning (colspan, rowspan)
- [ ] **Deliverable**: Render an invoice with header, line items table, and totals

### M3: Pagination (Week 7-8)
- [ ] Multi-page documents
- [ ] Page breaks (auto + forced)
- [ ] Running headers/footers
- [ ] Page numbers / page count
- [ ] Orphans/widows control
- [ ] **Deliverable**: Render a multi-page report with page numbers and repeated headers

### M4: Media & Links (Week 9)
- [ ] Image embedding (JPEG, PNG)
- [ ] Image sizing (width/height constraints, aspect ratio)
- [ ] Hyperlinks (PDF link annotations)
- [ ] Horizontal rules
- [ ] **Deliverable**: Render a branded document with logo, images, and clickable links

### M5: Lists & Polish (Week 10)
- [ ] Ordered/unordered lists with proper indentation
- [ ] Nested lists
- [ ] CLI tool with watch mode
- [ ] Error messages with line/column numbers
- [ ] **Deliverable**: Feature-complete v0.1

### M6: Templates & Data Binding (Week 11-12)
- [ ] Variable interpolation `{{var}}`
- [ ] `<for-each>` loops
- [ ] `<if>` conditionals
- [ ] `<include>` partials
- [ ] JSON data input
- [ ] **Deliverable**: Generate invoices from JSON data + XML template

## Non-Goals (v0.1)

- Full CSS compliance
- JavaScript execution
- SVG rendering
- PDF forms
- PDF encryption
- Accessibility (tagged PDF)
- Right-to-left text
- CJK text shaping
- Flexbox or Grid layout

## Dependencies (Go modules)

- `github.com/signintech/gopdf` — PDF generation backend (or `github.com/jung-kurt/gofpdf`)
- `golang.org/x/image` — image decoding
- Standard library: `encoding/xml`, `image`, `strings`, `strconv`, `fmt`, `os`, `io`, `path/filepath`

No CGO. No external binaries. Pure Go.

## Example Document

```xml
<?xml version="1.0" encoding="UTF-8"?>
<document>
  <head>
    <meta title="Invoice #1042" author="Vodacom Mozambique" />
    <style>
      page {
        size: A4;
        margin: 15mm 20mm;
      }
      body {
        font-family: "Liberation Sans";
        font-size: 10pt;
        color: #333333;
      }
      h1 {
        font-size: 22pt;
        color: #e60000;
        margin-bottom: 8pt;
      }
      .invoice-meta {
        margin-bottom: 12pt;
      }
      .invoice-meta p {
        margin: 2pt 0;
        font-size: 9pt;
        color: #666666;
      }
      table {
        width: 100%;
        border-collapse: collapse;
        margin-top: 16pt;
      }
      th {
        background-color: #e60000;
        color: #ffffff;
        padding: 6pt 8pt;
        text-align: left;
        font-size: 9pt;
      }
      td {
        padding: 6pt 8pt;
        border-bottom: 0.5pt solid #cccccc;
        font-size: 9pt;
      }
      .total-row td {
        font-weight: bold;
        border-top: 2pt solid #333333;
        border-bottom: none;
      }
      .right {
        text-align: right;
      }
      page-footer {
        font-size: 8pt;
        color: #999999;
        text-align: center;
      }
    </style>
  </head>
  <body>
    <page-header>
      <div class="header-bar">
        <img src="vodacom-logo.png" width="120pt" />
        <span class="right">Invoice</span>
      </div>
    </page-header>

    <h1>Invoice #1042</h1>
    <div class="invoice-meta">
      <p><strong>Date:</strong> 2026-03-21</p>
      <p><strong>Customer:</strong> Empresa XYZ, Lda.</p>
      <p><strong>NIF:</strong> 400123456</p>
    </div>

    <table>
      <thead>
        <tr>
          <th>Description</th>
          <th class="right">Qty</th>
          <th class="right">Unit Price</th>
          <th class="right">Total</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>API Calls (March 2026)</td>
          <td class="right">15,000</td>
          <td class="right">2.50 MZN</td>
          <td class="right">37,500.00 MZN</td>
        </tr>
        <tr>
          <td>SMS Notifications</td>
          <td class="right">3,200</td>
          <td class="right">1.00 MZN</td>
          <td class="right">3,200.00 MZN</td>
        </tr>
        <tr>
          <td>Storage (50GB)</td>
          <td class="right">1</td>
          <td class="right">5,000.00 MZN</td>
          <td class="right">5,000.00 MZN</td>
        </tr>
        <tr class="total-row">
          <td colspan="3" class="right">Total</td>
          <td class="right">45,700.00 MZN</td>
        </tr>
      </tbody>
    </table>

    <page-footer>
      <p>Page <page-number/> of <page-count/></p>
      <p>Vodacom Moçambique, S.A. — NIF 400000001</p>
    </page-footer>
  </body>
</document>
```
