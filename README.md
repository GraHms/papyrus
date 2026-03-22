# goxml2pdf

A pure Go library for converting XML+CSS documents to PDF. No Chromium. No external binaries. No CGO.

## Status

**Early development** — Not yet functional. See [SPEC.md](SPEC.md) for the full design specification.

## Concept

Define documents using a purpose-built XML vocabulary styled with a CSS subset, optimized for paged output (invoices, reports, contracts, certificates).

```xml
<document>
  <head>
    <style>
      page { size: A4; margin: 20mm; }
      h1 { font-size: 22pt; color: #e60000; }
      table { width: 100%; border-collapse: collapse; }
      td { padding: 6pt; border-bottom: 0.5pt solid #ccc; }
    </style>
  </head>
  <body>
    <h1>Invoice #1042</h1>
    <table>
      <tr><td>API Calls</td><td>37,500.00 MZN</td></tr>
    </table>
  </body>
</document>
```

## Design Principles

- **Paged-first**: Fixed page dimensions, not infinite canvas
- **Strict subset**: Only supported elements/properties are valid
- **Declarative**: No JavaScript, no dynamic behavior
- **Fast**: Target <100ms for a 5-page invoice
- **Zero dependencies**: Pure Go, no external binaries

## Installation

```bash
go install github.com/ismaelvodacom/goxml2pdf/cmd/goxml2pdf@latest
```

## Usage

### CLI

```bash
goxml2pdf invoice.xml                     # → invoice.pdf
goxml2pdf -data data.json template.xml    # with data binding
goxml2pdf -debug -o output.pdf input.xml  # debug mode
```

### Library

```go
import "github.com/ismaelvodacom/goxml2pdf/pkg/document"

err := document.GenerateFromFile("invoice.xml", "invoice.pdf")

// Or with options
err := document.Generate(reader, writer,
    document.WithDebug(),
    document.WithFont("Custom", "/path/to/font.ttf"),
)
```

## Documentation

- [SPEC.md](SPEC.md) — Full language specification (XML elements, CSS properties, architecture)
- [CLAUDE.md](CLAUDE.md) — AI development guide
- [examples/](examples/) — Example documents

## License

MIT
