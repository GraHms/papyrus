package document_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/grahms/pdfml/pkg/document"
)

const benchXML = `<?xml version="1.0" encoding="UTF-8"?>
<html>
<head>
	<meta title="Benchmark Report" author="Test" />
	<style>
		page { size: A4; margin: 20mm; }
		body { font-family: "Liberation Sans"; font-size: 10pt; color: #333; line-height: 1.5; }
		h1 { font-size: 24pt; font-weight: bold; color: #e60000; margin-bottom: 12pt; }
		h2 { font-size: 18pt; font-weight: bold; margin-top: 16pt; margin-bottom: 8pt; }
		p { margin-bottom: 8pt; text-indent: 20pt; }
		table { width: 100%; border-collapse: collapse; margin-top: 10pt; margin-bottom: 10pt; }
		th { font-weight: bold; background-color: #f0f0f0; border: 1pt solid #ccc; padding: 6pt; }
		td { border: 1pt solid #ccc; padding: 6pt; }
	</style>
</head>
<body>
	<header>
		<p>CONFIDENTIAL | Performance Report</p>
	</header>
	<main>
		<h1>PDFML Performance Benchmark</h1>
		<p>This document is used to measure the execution speed of the layout pipeline.</p>
		
		<h2>Data Simulation</h2>
		<table>
			<thead>
				<tr><th>ID</th><th>Metric</th><th>Value</th></tr>
			</thead>
			<tbody>
				<tr><td>1</td><td>Parse Speed</td><td>Fast</td></tr>
				<tr><td>2</td><td>Layout Speed</td><td>Faster</td></tr>
				<tr><td>3</td><td>Rendering Speed</td><td>Lightning</td></tr>
				<tr><td>4</td><td>Overall</td><td>Excellent</td></tr>
			</tbody>
		</table>

		<h2>Analysis</h2>
		<p>By creating a medium-sized DOM tree with mixed inline runs, block layouts, tables, and cascading CSS styles, we can observe the raw algorithmic overhead associated with the boxing model.</p>
		<p>Subsequent calls using a pre-parsed template should bypass the parser phase entirely.</p>
	</main>
	<footer>
		<p>Page <page-number/> of <page-count/></p>
	</footer>
</body>
</html>`

// BenchmarkGenerate profiles the full pipeline (Parse -> Style -> Layout -> Render)
func BenchmarkGenerate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := document.Generate(strings.NewReader(benchXML), &buf)
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

// BenchmarkParse profiles only the XML and CSS parsing phase
func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := document.Parse(strings.NewReader(benchXML))
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkRender profiles the execution of a pre-parsed cached template
func BenchmarkRender(b *testing.B) {
	doc, err := document.Parse(strings.NewReader(benchXML))
	if err != nil {
		b.Fatalf("Initial parse failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use io.Discard because io writes (like writing to an in-memory byte slice)
		// can allocate dynamically, and we only want to measure algorithmic overhead.
		err := doc.Render(io.Discard)
		if err != nil {
			b.Fatalf("Render failed: %v", err)
		}
	}
}
