# goxml2pdf — Roadmap

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
| CLI tool | ✅ | `goxml2pdf -o out.pdf input.xml` with `-debug`, `-font`, `-data`, `-page-size`, `-dpi` flags |
| Example documents | ✅ | `invoice.xml`, `report.xml`, `certificate.xml` |
| Unit tests (parser, style, document) | ✅ | Table-driven tests in `*_test.go` |

---

## M1 — HTML-Compatible Vocabulary + Text Polish ⬜

Expand the element set to match HTML authoring habits and improve text rendering fidelity.

### HTML vocabulary expansion

| Item | Status | Notes |
|---|---|---|
| `<html>` root alias for `<document>` | ⬜ | Normalised to `document` at parse time |
| `<header>` alias for `<page-header>` (body-level) | ⬜ | Normalised to `page-header` at parse time |
| `<footer>` alias for `<page-footer>` (body-level) | ⬜ | Normalised to `page-footer` at parse time |
| `<main>`, `<article>`, `<aside>`, `<nav>` semantic blocks | ⬜ | Render as generic block containers |
| `<pre>` preformatted block | ⬜ | Preserves whitespace; monospace font default |
| `<figure>` + `<figcaption>` | ⬜ | Block image container with optional caption |
| `<s>` strikethrough inline | ⬜ | UA stylesheet default: `text-decoration: line-through` |
| `<mark>` highlight inline | ⬜ | UA stylesheet default: `background-color: #ffff00` |
| `<small>` inline | ⬜ | UA stylesheet default: `font-size: 0.85em` |
| `<sub>` / `<sup>` inline | ⬜ | Render at reduced size, shifted baseline |
| `<cite>` / `<q>` inline | ⬜ | UA stylesheet defaults: `font-style: italic` / quotes |
| `<caption>` in tables | ⬜ | Rendered above the table |
| `<dl>`, `<dt>`, `<dd>` definition lists | ⬜ | `dt` bold, `dd` indented by default |
| CSS `header` / `footer` selectors map to canonical names | ⬜ | Warn developer if they write `header {}` — normalisation means they should use `page-header {}` |

### Typography polish

| Item | Status | Notes |
|---|---|---|
| `text-align: justify` | ⬜ | Currently falls through to left-align |
| `text-decoration: underline` | ⬜ | Render underline lines below text runs |
| `text-decoration: line-through` | ⬜ | Render strikethrough lines |
| `text-transform` (uppercase / lowercase / capitalize) | ⬜ | Applied at render time |
| `letter-spacing` | ⬜ | Inter-character spacing |
| `white-space: pre` / `nowrap` | ⬜ | Preserve whitespace / disable wrapping |
| `line-height` ratio inheritance fix | ⬜ | Inherited value should be the ratio, not the resolved pt value |
| Knuth-Plass line breaking (optional) | ⬜ | Better paragraph quality than greedy; can be a build flag |
| Baseline alignment for mixed inline styles | ⬜ | Align text baselines when font sizes differ within a line |
| `vertical-align` in table cells | ⬜ | top / middle / bottom cell content alignment |
| `<a>` PDF link annotations | ⬜ | Emit `/Annot` with `/URI` for `href` attributes |
| **Layout/render unit tests** | ⬜ | `pkg/layout` and `pkg/render` currently have no `*_test.go` |

---

## M2 — Tables (Full) ⬜

Complete the table layout algorithm.

| Item | Status | Notes |
|---|---|---|
| `table-layout: fixed` | ⬜ | Column widths from first row; faster than auto |
| `table-layout: auto` (column min/max widths) | ⬜ | True CSS auto table sizing |
| `rowspan` | ⬜ | Currently only `colspan` is handled |
| `border-collapse: collapse` | ⬜ | Merge adjacent cell borders; currently drawn separately |
| `border-spacing` | ⬜ | Gap between cells in `separate` mode |
| `<thead>` repetition on page breaks | ⬜ | Re-emit header rows when a table spans pages |
| `<tfoot>` at page bottom | ⬜ | Emit footer rows before the page break |
| `<th>` default bold + center styling | ⬜ | Apply via UA stylesheet defaults |
| Cell padding inheritance | 🔧 | Works but not fully tested with collapsed borders |

---

## M3 — Pagination Polish ⬜

More control over how content flows across pages.

| Item | Status | Notes |
|---|---|---|
| `page-break-before: always` / `page-break-after: always` | ⬜ | CSS-triggered forced breaks |
| `page-break-inside: avoid` | ⬜ | Try to keep a box on one page |
| `orphans` / `widows` control | ⬜ | Minimum lines at top/bottom of a page |
| Different first-page header/footer | ⬜ | `<page-header first-only>` or CSS `:first` selector |
| Per-page size / orientation changes | ⬜ | `@page :left / :right` analog |

---

## M4 — CSS Completeness ⬜

Round out the CSS subset defined in SPEC.md.

| Item | Status | Notes |
|---|---|---|
| `:first-child` / `:last-child` pseudo-classes | ⬜ | Selector matching |
| `:nth-child(n)` | ⬜ | Formula-based matching |
| `border` shorthand (full) | 🔧 | Width+style+color shorthand partially handled |
| `background-image` (solid only for now) | ⬜ | Background images with `url()` |
| `opacity` | ⬜ | Applied to box and children |
| `overflow: hidden` | ⬜ | Clip content to box bounds |
| `display: inline-block` | ⬜ | Inline container with block sizing |
| `max-height` / `min-height` | ⬜ | Height constraint resolution |
| CSS `@page` rule | ⬜ | `size`, `margin` from stylesheet instead of XML attributes |
| Unknown property warnings with location | ⬜ | `css: unknown property "float" at line 12` |

---

## M5 — Templates & Data Binding ⬜

Generate documents from structured data without writing XML by hand.

| Item | Status | Notes |
|---|---|---|
| `{{var}}` interpolation | ⬜ | Replace variables from JSON or Go map |
| `<var name="x" value="y"/>` definitions | ⬜ | Inline variable declaration in `<head>` |
| `<for-each>` loops | ⬜ | Repeat a block for each item in a JSON array |
| `<if>` conditionals | ⬜ | Render block only when expression is truthy |
| `<include src="partial.xml"/>` | ⬜ | Compose documents from reusable fragments |
| JSON data file via `-data` CLI flag | ⬜ | Load data before template expansion |
| Go struct data binding via API | ⬜ | Pass `any` to `Generate()` options |

---

## M6 — Quality & Performance ⬜

Harden the library for production use.

| Item | Status | Notes |
|---|---|---|
| Golden-file integration tests | ⬜ | Box-tree text snapshots for each example document |
| Benchmark suite | ⬜ | `testing.B` for parse, layout, and render on 5-page invoice |
| Fuzzing (parser) | ⬜ | `go test -fuzz` on XML and CSS parsers |
| Error messages with line/column for all parse errors | 🔧 | XML has line tracking; CSS warnings need location |
| Strict unknown-element errors | 🔧 | Validation exists but unknown elements currently warn only |
| `go vet` + `staticcheck` CI gate | ⬜ | Add GitHub Actions workflow |
| Godoc for all exported symbols | 🔧 | Partial coverage |
| README with usage examples and screenshots | 🔧 | Basic README exists; needs screenshots |

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
