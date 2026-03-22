package parser

import (
	"strings"
	"testing"
)

func TestParseXML_Basic(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<document>
  <head>
    <meta title="Test Doc" author="Tester" />
    <style>body { color: red; }</style>
  </head>
  <body>
    <h1>Hello</h1>
    <p>World</p>
  </body>
</document>`

	doc, err := ParseXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseXML error: %v", err)
	}
	if doc.Root == nil {
		t.Fatal("expected non-nil root")
	}
	if doc.Root.Tag != "document" {
		t.Errorf("expected root tag 'document', got %q", doc.Root.Tag)
	}
	if doc.Meta["title"] != "Test Doc" {
		t.Errorf("expected meta title 'Test Doc', got %q", doc.Meta["title"])
	}
	if !strings.Contains(doc.Styles, "color: red") {
		t.Errorf("expected style to contain 'color: red', got %q", doc.Styles)
	}

	body := FindElement(doc.Root, "body")
	if body == nil {
		t.Fatal("expected <body> element")
	}
	h1 := FindElement(body, "h1")
	if h1 == nil {
		t.Fatal("expected <h1> element")
	}
}

func TestParseXML_UnknownElement(t *testing.T) {
	xml := `<document><body><foobar/></body></document>`
	_, err := ParseXML(strings.NewReader(xml))
	if err == nil {
		t.Fatal("expected error for unknown element <foobar>")
	}
}

func TestParseXML_TextNodes(t *testing.T) {
	xml := `<document><body><p>Hello world</p></body></document>`
	doc, err := ParseXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseXML error: %v", err)
	}
	body := FindElement(doc.Root, "body")
	p := FindElement(body, "p")
	if p == nil {
		t.Fatal("expected <p>")
	}
	if len(p.Children) == 0 {
		t.Fatal("expected text child in <p>")
	}
	if p.Children[0].Type != TextNode {
		t.Errorf("expected TextNode child, got type %v", p.Children[0].Type)
	}
	if !strings.Contains(p.Children[0].Text, "Hello world") {
		t.Errorf("expected text 'Hello world', got %q", p.Children[0].Text)
	}
}

func TestParseXML_Attributes(t *testing.T) {
	xml := `<document><body><p id="main" class="highlight big">text</p></body></document>`
	doc, err := ParseXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseXML error: %v", err)
	}
	body := FindElement(doc.Root, "body")
	p := FindElement(body, "p")
	if p.ID != "main" {
		t.Errorf("expected ID 'main', got %q", p.ID)
	}
	if !p.HasClass("highlight") {
		t.Error("expected class 'highlight'")
	}
	if !p.HasClass("big") {
		t.Error("expected class 'big'")
	}
}
