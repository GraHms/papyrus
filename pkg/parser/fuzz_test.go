package parser

import (
	"bytes"
	"testing"
)

// FuzzParseXML ensures the XML parser doesn't panic on malformed input.
func FuzzParseXML(f *testing.F) {
	// Seed with valid XML instances to guide the fuzzer
	f.Add([]byte(`<html><body><h1>Hello</h1></body></html>`))
	f.Add([]byte(`<xml><head><meta title="doc"/></head><body><p>Test</p></body></xml>`))
	f.Add([]byte(`<html>
					<head><style>body { color: red; }</style></head>
					<body>
						<table><tr><td>1</td></tr></table>
					</body>
				  </html>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Just ensure it doesn't panic. Errors are expected.
		doc, err := ParseXML(bytes.NewReader(data))
		if err == nil {
			// If it somehow parsed successfully, make sure validation doesn't panic either.
			_ = ValidateDocument(doc)
		}
	})
}

// FuzzParseCSS ensures the CSS parser doesn't panic on malformed input.
func FuzzParseCSS(f *testing.F) {
	f.Add(`body { font-size: 12pt; }`)
	f.Add(`h1, h2 { color: #f00; margin-top: 10px; }`)
	f.Add(`@page { margin: 20mm; size: A4; }`)
	f.Add(`tr:nth-child(even) { background-color: #eee; }`)

	f.Fuzz(func(t *testing.T, data string) {
		_, _ = ParseCSS(data)
	})
}
