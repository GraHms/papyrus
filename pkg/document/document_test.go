package document

import (
	"bytes"
	"strings"
	"testing"
)

const basicXML = `<?xml version="1.0" encoding="UTF-8"?>
<document>
  <head>
    <meta title="Test" author="Tester" />
    <style>
      page { size: A4; margin: 20mm; }
      body { font-family: "Liberation Sans"; font-size: 11pt; color: #333333; }
      h1 { font-size: 24pt; font-weight: bold; color: #000000; }
      p { margin-bottom: 8pt; }
      .highlight { background-color: #ffffcc; }
    </style>
  </head>
  <body>
    <h1>Hello World</h1>
    <p>This is a <strong>test</strong> paragraph with <em>italic</em> text.</p>
    <p class="highlight">Highlighted paragraph.</p>
  </body>
</document>`

func TestGenerate_Basic(t *testing.T) {
	var out bytes.Buffer
	err := Generate(strings.NewReader(basicXML), &out)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
	// PDF starts with %PDF-
	if !bytes.HasPrefix(out.Bytes(), []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF-, got: %q", out.Bytes()[:min(20, out.Len())])
	}
}

func TestGenerate_WithDebug(t *testing.T) {
	var out bytes.Buffer
	err := Generate(strings.NewReader(basicXML), &out, WithDebug())
	if err != nil {
		t.Fatalf("Generate with debug error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty PDF output in debug mode")
	}
}

func TestGenerate_InvalidXML(t *testing.T) {
	err := Generate(strings.NewReader(`<not-valid-xml`), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestGenerate_UnknownElement(t *testing.T) {
	bad := `<document><body><foobar/></body></document>`
	err := Generate(strings.NewReader(bad), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for unknown element")
	}
}

func TestGenerate_PageBreak(t *testing.T) {
	xml := `<document>
  <head><style>page{size:A4;margin:20mm;}</style></head>
  <body>
    <h1>Page One</h1>
    <page-break/>
    <h1>Page Two</h1>
  </body>
</document>`
	var out bytes.Buffer
	err := Generate(strings.NewReader(xml), &out)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty PDF")
	}
}

func TestGenerate_Lists(t *testing.T) {
	xml := `<document>
  <head><style>page{size:A4;margin:20mm;} body{font-size:11pt;}</style></head>
  <body>
    <ul>
      <li>Item one</li>
      <li>Item two</li>
      <li>Item three</li>
    </ul>
    <ol>
      <li>First</li>
      <li>Second</li>
    </ol>
  </body>
</document>`
	var out bytes.Buffer
	err := Generate(strings.NewReader(xml), &out)
	if err != nil {
		t.Fatalf("Generate with lists error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty PDF")
	}
}

func TestGenerate_Table(t *testing.T) {
	xml := `<document>
  <head><style>page{size:A4;margin:20mm;} table{width:100%;} td,th{padding:4pt;}</style></head>
  <body>
    <table>
      <thead><tr><th>Name</th><th>Value</th></tr></thead>
      <tbody>
        <tr><td>Alpha</td><td>1</td></tr>
        <tr><td>Beta</td><td>2</td></tr>
      </tbody>
    </table>
  </body>
</document>`
	var out bytes.Buffer
	err := Generate(strings.NewReader(xml), &out)
	if err != nil {
		t.Fatalf("Generate with table error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty PDF")
	}
}

func TestParse_Valid(t *testing.T) {
	doc, err := Parse(strings.NewReader(basicXML))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil Document")
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse(strings.NewReader(`<document><body><bad/></body></document>`))
	if err == nil {
		t.Fatal("expected error for invalid element")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
