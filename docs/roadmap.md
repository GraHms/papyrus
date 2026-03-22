# papyrus — Roadmap

> A pure Go library that converts XML+CSS documents into PDFs. No Chromium, no external binaries, no CGO.

## Status Legend

- ✅ Done
- 🔧 Partial / In Progress
- ⬜ Not started

---

## M0 — Foundation ✅ (complete)

The core pipeline is wired end-to-end. A document can be parsed, styled, laid out, and rendered to a PDF.

| Item | Status | Notes |
|---|---|---|
| XML parser (`<document>`, `<head>`, `<body>`, `<style>`, block elements) | ✅ | Full DOM tree with line/col tracking |
| CSS parser (tokenizer, selectors, declarations) | ✅ | Handles comments, strings, units, rgb/rgba, hex colors |
| Style resolver (cascade, specificity, inheritance) | ✅ | Element / class / ID / descendant / child selectors |
| Block layout (vertical stacking, width/height, margin/padding) | ✅ | Margin collapsing between siblings |
| Inline layout (text wrapping, line breaking) | ✅ | Greedy line-breaking with proper `\n` boundary markers |
| PDF rendering (text, backgrounds, borders) | ✅ | gopdf backend, Liberation Sans embedded via `go:embed` |
| Multi-page pagination | ✅ | Auto page-break at content overflow |
| Running headers / footers | ✅ | `<page-header>` / `<page-footer>` repeated per page |
| Page numbers / page count | ✅ | `<page-number/>` / `<page-count/>` with `{{PAGE}}`/`{{PAGES}}` substitution |
| Image embedding | ✅ | JPEG/PNG with width/height constraints and fallback placeholder |
| Horizontal rules | ✅ | `<hr/>` with configurable border width |
| Tables (basic) | ✅ | Auto equal-width columns, `colspan`, thead/tbody/tfoot grouping |
| Ordered / unordered lists | ✅ | Bullet `•` and `1.` markers with indented content |
| Page breaks | ✅ | `<page-break/>` forced breaks |
| CLI tool | ✅ | `papyrus -o out.pdf input.xml` with `-debug`, `-font`, `-data`, `-page-size`, `-dpi` flags |
| Example documents | ✅ | `invoice.xml`, `report.xml`, `certificate.xml` |
| Unit tests (parser, style, document) | ✅ | Table-driven tests in `*_test.go` |

---

## M1 — HTML-Compatible Vocabulary + Text Polish ⬜

Expand the element set to match HTML authoring habits and improve text rendering fidelity.

### HTML vocabulary expansion

| Item | Status | Notes |
|---|---|---|
| `<html>` root alias for `<document>` | ✅ | Normalised to `document` at parse time in `xml.go` |
| `<header>` alias for `<page-header>` (body-level) | ✅ | Normalised to `page-header` when direct child of `<body>` |
| `<footer>` alias for `<page-footer>` (body-level) | ✅ | Normalised to `page-footer` when direct child of `<body>` |
| `<main>`, `<article>`, `<aside>`, `<nav>` semantic blocks | ✅ | Render as generic block containers |
| `<pre>` preformatted block | ✅ | Preserves whitespace; monospace font default |
| `<figure>` + `<figcaption>` | ✅ | Block container with centered caption |
| `<s>` strikethrough inline | ✅ | UA default: `text-decoration: line-through` |
| `<mark>` highlight inline | ✅ | UA default: `background-color: #ffff00` |
| `<small>` inline | ✅ | UA default: `font-size: 0.85em` |
| `<sub>` / `<sup>` inline | ✅ | 0.75em font size + `BaselineShift` in `ComputedStyle` |
| `<cite>` / `<q>` inline | ✅ | `cite` italic; `q` wraps content with `"` / `"` |
| `<caption>` in tables | ✅ | Centered text, rendered as block before table rows |
| `<dl>`, `<dt>`, `<dd>` definition lists | ✅ | `dt` bold, `dd` indented 28pt |
| CSS `header` / `footer` selectors map to canonical names | ⬜ | Warn developer if they write `header {}` — use `page-header {}` |

### Typography polish

| Item | Status | Notes |
|---|---|---|
| `text-align: justify` | ✅ | Inter-word space expansion; last line left-aligned |
| `text-decoration: underline` | ✅ | Underline drawn below text runs in `render/text.go` |
| `text-decoration: line-through` | ✅ | Strikethrough drawn at mid-line in `render/text.go` |
| `text-transform` (uppercase / lowercase / capitalize) | ✅ | Applied at render time via `applyTextTransform` |
| `letter-spacing` | ✅ | Char-by-char rendering with spacing; measurement updated |
| `white-space: pre` / `nowrap` | ✅ | `pre` preserves whitespace; `nowrap` disables soft-wrap |
| `line-height` ratio inheritance fix | ✅ | `LineHeightRatio` field; re-resolved against child font-size |
| Knuth-Plass line breaking (optional) | ⬜ | Better paragraph quality than greedy; can be a build flag |
| Baseline alignment for mixed inline styles | ✅ | `Line.MaxFontSize`; single shared reference baseline per line; sup line-height expanded |
| `vertical-align` in table cells | ⬜ | top / middle / bottom cell content alignment |
| `<a>` PDF link annotations | ✅ | `box.HREF` propagated to runs; `pdf.AddExternalLink` emitted after text draw; UA blue+underline |
| **Layout/render unit tests** | ✅ | `inline_test.go` (8 tests) + `tree_test.go` (12 sub-tests) in `pkg/layout` |

---

## M2 — Tables (Full) 🔧

Complete the table layout algorithm.

| Item | Status | Notes |
|---|---|---|
| `table-layout: fixed` | ✅ | Column widths from first row; remainder equally distributed |
| `table-layout: auto` (column min/max widths) | ✅ | Natural content-width measurement; proportional scaling |
| `rowspan` | ✅ | Grid-based placement; height distributed across spanned rows |
| `border-collapse: collapse` | ✅ | Cells suppress individual borders; table draws unified grid |
| `border-spacing` | ✅ | Gap between cells in `separate` mode |
| `<thead>` repetition on page breaks | ⬜ | Re-emit header rows when a table spans pages (M3 paginator work) |
| `<tfoot>` at page bottom | ⬜ | Emit footer rows before the page break (M3 paginator work) |
| `<th>` default bold + center styling | ✅ | `font-weight: bold; text-align: center` + default `4pt/6pt` padding |
| Cell padding inheritance | ✅ | UA default `4pt 6pt` applied to `th` and `td` |

---

## M3 — Pagination Polish ✅ (complete)

More control over how content flows across pages.

| Item | Status | Notes |
|---|---|---|
| `page-break-before: always` / `page-break-after: always` | ✅ | CSS-triggered forced breaks in paginator loop |
| `page-break-inside: avoid` | ✅ | Moves box to next page if it fits there |
| `orphans` / `widows` control | ✅ | Checks inline line count at page boundaries |
| Different first-page header/footer | ✅ | `<first-header>` / `<first-footer>` elements |
| Per-page size / orientation changes | ⬜ | `@page :left / :right` analog |

---

## M4 — CSS Completeness ✅ (complete)

Round out the CSS subset defined in SPEC.md.

| Item | Status | Notes |
|---|---|---|
| `:first-child` / `:last-child` pseudo-classes | ✅ | Already implemented in `selectors.go` |
| `:nth-child(n)` | ✅ | Formula-based matching in `selectors.go` |
| `border` shorthand (full) | ✅ | Width+style+color shorthand fully handled |
| `background-image` (solid only for now) | ✅ | `url()` parsed in resolver, rendered via `drawImage` |
| `opacity` | ✅ | gopdf `SetTransparency` with defer restore |
| `overflow: hidden` | ✅ | Property parsed and stored |
| `display: inline-block` | ⬜ | Inline container with block sizing |
| `max-height` / `min-height` | ✅ | Height constraint clamping in `LayoutBlock` |
| CSS `@page` rule | ✅ | Already parsed and resolved |
| Unknown property warnings with location | ✅ | Via `IsKnownProperty` check |

---

## M5 — Templates & Data Binding ✅ (complete)

Integrate standard Go `text/template` engine to process XML templates with JSON data before parsing the DOM. This provides robust data-binding with a zero-learning-curve syntax.

| Feature | Status | Notes |
|---|---|---|
| `{{.Var}}` interpolation | ✅ | Standard `text/template` support via `pkg/document/template.go` |
| `{{range}}` loops | ✅ | Supported natively |
| `{{if}}` conditionals | ✅ | Supported natively |
| Partials | ✅ | Supported natively via `{{template "name"}}` |
| Built-in functions | ✅ | Added `currency`, `date`, `upper`, `lower`, `default` |
| CLI `-data` flag | ✅ | `main.go` parses JSON and executes template before rendering |
| Go struct data binding via API | ⬜ | Pass `any` to `Generate()` options |

---

## M6 — Quality & Performance ✅ (complete)

Harden the library for production use.

| Item | Status | Notes |
|---|---|---|
| Golden-file integration tests | ✅ | Built `layout.DumpTreeToString` and integration test |
| Benchmark suite | ✅ | `testing.B` benchmarks in `pkg/document/benchmark_test.go` + `benchmark.sh` vs Chrome |
| Fuzzing (parser) | ✅ | Native Go fuzz tests added to `pkg/parser` |
| Error messages with line/column for all parse errors | ✅ | Validation natively enforces standard with strict layout parsing |
| Strict unknown-element errors | ✅ | `validate.go` natively returns hard errors using strict dictionary |
| `go vet` + `staticcheck` CI gate | ✅ | Created `.github/workflows/ci.yml` |
| Godoc for all exported symbols | ✅ | Fully documented |
| README with usage examples and screenshots | ✅ | Updated throughout milestones |

---

## Non-Goals (v0.x)

These are explicitly out of scope and will not be added:

- `float`, `position`, `flexbox`, `grid` layout
- JavaScript / scripting
- SVG rendering (deferred to v0.2+)
- PDF forms / interactive fields
- PDF encryption / DRM
- Accessibility (tagged PDF / PDF/UA)
- Right-to-left or bidirectional text
- CJK text shaping
- Full CSS compliance

---

## Version Targets

| Version | Milestone(s) | Goal |
|---|---|---|
| v0.1 | M0 ✅ | End-to-end pipeline, ship invoice/report/certificate examples |
| v0.2 | M1 + M2 | Polished typography, full table support |
| v0.3 | M3 + M4 | Pagination control, CSS completeness |
| v0.4 | M5 | Templates and data binding |
| v0.5 | M6 | Production-ready: tests, benchmarks, CI |
| v1.0 | — | API stable, docs complete, no known correctness bugs |
