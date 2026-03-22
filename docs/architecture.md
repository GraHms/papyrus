# Library Architecture

Papyrus is a document-processing library, not a browser wrapper.

That distinction is the core architectural decision in the project.

Instead of delegating rendering to Chromium, WebKit, or a system PDF tool,
Papyrus owns the entire document pipeline in-process:

```text
template/data
  -> XML + CSS parse
  -> style resolution
  -> page-aware layout
  -> PDF rendering
```

This has a direct consequence: Papyrus can optimize for fixed-page document
generation instead of inheriting the complexity of a general-purpose browser
engine.

## Why This Architecture Exists

Browser engines solve a much larger problem than PDF generation:

- dynamic DOM mutation
- JavaScript execution
- event handling
- accessibility trees
- compositing
- scrolling and viewport behavior
- media queries and interactive layout

Papyrus does not need those capabilities to generate invoices, reports,
certificates, statements, or contracts. Carrying them into the system would
increase runtime cost, deployment cost, and failure surface without improving
the core product.

Papyrus therefore takes the opposite approach:

- narrow the document model
- make layout page-first
- keep the runtime deterministic
- expose the system as a composable Go library

That is why the architecture is split into explicit phases instead of hidden
behind a single monolithic renderer.

## Design Goals

The package structure is shaped by a few hard goals:

1. Deterministic output for the same inputs
2. No external binary dependencies
3. Clear failure boundaries between parse, style, layout, and render
4. Reusable intermediate representations
5. A public API that is simple even though the engine is layered internally

These are library goals first, not just implementation preferences.

## Top-Level Package Roles

### `pkg/document`

This is the public façade.

It exposes the main entry points:

- parse a document
- render a document
- generate from file, bytes, or strings
- work with templates and options

From a user perspective, `pkg/document` is "the library." Internally, it is an
orchestrator that wires together the deeper engine packages.

The important architectural choice here is that the public API does not leak
the internals of parsing, styling, layout, or rendering. That keeps the API
stable while allowing the engine to evolve behind it.

### `pkg/parser`

The parser layer converts external author input into internal, validated
structures.

Responsibilities:

- parse XML into the document tree
- parse CSS into structured rules
- normalize HTML aliases into canonical internal tags
- validate supported elements and attributes

This layer exists to establish a clean contract for everything downstream. By
the time layout begins, the system should not still be debating what a tag
means or whether a property is valid.

That early normalization is important. A robust engine wants semantic cleanup
as close to the input boundary as possible.

### `pkg/style`

The style layer resolves author intent into computed values.

Responsibilities:

- selector matching
- cascade resolution
- inheritance
- property defaults
- unit conversion
- computed style generation

This layer is separate from parsing because syntax is not the same thing as
meaning. CSS text can be parsed without yet knowing what a node's final margin,
font size, or display behavior should be.

Keeping style resolution separate makes three things easier:

1. testing cascade behavior independently of layout
2. changing selector/property behavior without destabilizing parsing
3. caching or reusing resolved structures when documents are rendered many
   times

### `pkg/layout`

The layout layer is the center of the engine.

Responsibilities:

- convert styled nodes into boxes
- compute block and inline dimensions
- break text into lines
- handle tables
- paginate across fixed-size pages
- repeat headers and footers where required

This phase is where Papyrus diverges most strongly from browser mental models.

A browser is fundamentally viewport-first. Papyrus is page-first.

That difference matters:

- every page has known dimensions before layout starts
- pagination is a primary concern, not a post-process
- headers, footers, page breaks, and page counters are first-class layout
  concepts
- layout can optimize for static output rather than incremental repaint

In architectural terms, `pkg/layout` is where a styled document becomes a
physical document.

### `pkg/render`

The render layer turns the laid-out page model into PDF bytes.

Responsibilities:

- text drawing
- font registration and measurement support
- images
- borders, fills, lines, and backgrounds
- metadata propagation
- PDF emission through the backend

This package is intentionally downstream of layout. Rendering should not decide
where things go; it should draw what layout already decided.

That separation prevents a common class of architectural drift where rendering
logic starts smuggling in layout rules. Once that happens, correctness becomes
harder to test and regressions become harder to localize.

## Pipeline Walkthrough

The flow through the library looks like this:

```text
author XML / template
  -> parser.Document
  -> []CSS rules
  -> computed styles per node
  -> box tree
  -> page layout
  -> renderer
  -> PDF bytes
```

Each transition narrows ambiguity:

- parse removes syntactic ambiguity
- style removes cascade ambiguity
- layout removes geometric ambiguity
- render removes output ambiguity

That staged reduction is one of the main reasons the library remains tractable.

## Why The Boundaries Matter

Good architecture is not just about splitting code into packages. It is about
controlling where decisions are allowed to happen.

Papyrus uses package boundaries to keep decisions local:

- parsing decides structure
- style decides computed visual values
- layout decides geometry and pagination
- render decides drawing commands only

This matters because PDF generation is otherwise prone to boundary collapse.
Without discipline, property parsing leaks into rendering, rendering leaks into
layout, and public APIs start depending on accidental engine details.

The current design prevents that collapse.

## Why This Is Better Than Browser Automation For The Target Problem

For the Papyrus problem domain, the library architecture has real advantages.

### Lower operational cost

- no headless browser lifecycle
- no IPC with an external renderer
- no heavy container images
- no system package dependency chain

### Stronger determinism

- fixed-page layout model
- no JavaScript or asynchronous document mutation
- fewer environment-specific rendering variables

### Better composability

- usable directly from Go services
- fits naturally into in-memory pipelines
- can parse once and render many times

### Better debuggability

- failures can be localized by phase
- layout tree snapshots provide stable regression signals
- internal representations can be inspected without reverse-engineering browser
  behavior

## Tradeoffs

This architecture is intentionally narrow, and that has consequences.

- Papyrus does not aim to support arbitrary web content.
- The supported HTML/CSS surface is intentionally constrained.
- Features that browsers get "for free" must be designed explicitly here.
- The layout engine is specialized work; correctness is owned by the library,
  not outsourced to a browser team.

Those are acceptable tradeoffs because the project is optimizing for
high-throughput, server-side generation of paged documents, not universal web
compatibility.

## Distinguished Engineer View

At a systems level, Papyrus is designed around a simple principle:

> Specialize the engine around the real problem instead of carrying a general
> platform into a narrow use case.

That principle shows up everywhere:

- XML + CSS subset instead of full browser HTML
- computed styles instead of ad hoc per-node rendering logic
- page-first layout instead of viewport-first layout
- render-after-layout instead of intertwined placement and drawing
- a façade package for stability, with internal packages for rigor

The result is not "smaller than a browser." It is structurally different from a
browser. That difference is the reason Papyrus can be fast, deterministic, and
operationally lightweight while still remaining understandable as a library.
