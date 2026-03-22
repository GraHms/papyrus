# pdfml 📄⚡

A pure Go, lightning-fast PDF generation library powered by a custom HTML/CSS subset designed specifically for paged media. 

**No Chromium. No wkhtmltopdf. No external binaries. Zero CGO dependencies.**

---

## Why `pdfml`?

Historically, generating PDFs in web services has meant installing heavy dependencies like Puppeteer, Headless Chrome, or `wkhtmltopdf`. These tools:
- Consume massive amounts of RAM and CPU.
- Make Docker images enormous.
- Add significant latency to every request by spinning up headless browser contexts.
- Treat pagination (headers, footers, page breaks) as an afterthought since they were built for infinite-scroll web pages.

**`pdfml` solves this.** It parses a strictly scoped subset of HTML/XML and CSS that makes sense for printable, paged interfaces (invoices, certificates, reports) and directly emits raw PDF bytes using an embedded layout engine. 

### Performance Gains 🚀
1. **Insanely Little Overhead:** Running purely in Go means you don't spin up external processes. Memory usage drops from hundreds of megabytes to mere kilobytes.
2. **Pre-Parsed Templates:** The engine allows parsing a template and its CSS *once* at server startup, resolving the layout algorithms, and repeatedly emitting PDFs dynamically through `Render()`.
3. **Instantaneous Execution:** Typical generations of intricate 5-page invoices resolve in single-digit milliseconds. 

---

## Concept & HTML Compatibility

Define documents using a web-familiar vocabulary. `pdfml` natively accepts HTML aliasing (`<html>`, `<main>`, `<header>`, `<footer>`), meaning developers comfortable with standard HTML and CSS can immediately start designing layouts.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<html>
  <head>
    <meta title="Invoice #1042" author="Acme Corp" />
    <style>
      page { size: A4; margin: 15mm; }
      body { font-family: "Liberation Sans"; font-size: 10pt; color: #333; }
      h1 { font-size: 22pt; color: #e60000; }
      header { text-align: right; border-bottom: 2pt solid #e60000; }
      table { width: 100%; border-collapse: collapse; margin-top: 16pt; }
      th { background-color: #333; color: white; padding: 6pt; }
      td { border-bottom: 0.5pt solid #ccc; padding: 6pt; }
      footer { text-align: center; font-size: 8pt; color: #999; }
    </style>
  </head>
  <body>
    <!-- <header> automatically repeats on every page -->
    <header>Acme Corp Invoices</header>

    <main>
      <h1>Invoice #1042</h1>
      <table>
        <thead>
          <tr><th>Description</th><th>Total</th></tr>
        </thead>
        <tbody>
          <tr><td>API Calls (March)</td><td>$375.00</td></tr>
        </tbody>
      </table>
    </main>

    <!-- <footer> automatically repeats on every page -->
    <footer>Page <page-number/> of <page-count/></footer>
  </body>
</html>
```

---

## Installation

```bash
# Add the library to your Go project
go get github.com/grahms/pdfml/pkg/document

# Install the CLI tool
go install github.com/grahms/pdfml/cmd/pdfml@latest
```

## Examples

The library's API relies heavily on standard Go `io` interfaces and highly extensible Functional Options. 

### Quick Generation
```go
import "github.com/grahms/pdfml/pkg/document"

// Simplest method (generates straight from a file to a file)
err := document.GenerateFromFile("invoice.xml", "invoice.pdf")

// Generate entirely in-memory using byte slices
pdfBytes, err := document.GenerateFromBytes(xmlBytes)
```

### Pre-Parsed Templates (High-Performance)
If you generate the same invoice style frequently, you can parse the XML and resolve the CSS rules *once* at startup, saving significant CPU cycles.

```go
// 1. On server startup: parse the document once
templateFile, _ := os.Open("invoice_template.xml")
doc, _ := document.Parse(templateFile)

// 2. In your HTTP Handlers: render repeatedly using the cached parsed state 
err := doc.Render(w, document.WithData(myDataMap))
```

### In-Memory Font Registry (`go:embed` friendly)
Instead of shipping `.ttf` files on your disk alongside your binary, you can use `go:embed` and register fonts dynamically:

```go
import _ "embed"

//go:embed my-custom-font.ttf
var myFont []byte

err := document.Generate(in, out,
    document.WithFontFromBytes("My Custom Font", myFont),
    document.WithPageSize("LETTER"),
    document.WithDPI(300),
    document.WithDebug(),
)
```

## Documentation

- [SPEC.md](SPEC.md) — Full language specification (supported tags, supported CSS, box models, etc)
- [examples/](examples/) — Example `.xml` layout documents showcasing tables, typography, and certificates

## License
MIT
