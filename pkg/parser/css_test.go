package parser

import (
	"testing"
)

func TestParseCSS_ElementRule(t *testing.T) {
	css := `p { color: red; font-size: 12pt; }`
	rules, err := ParseCSS(css)
	if err != nil {
		t.Fatalf("ParseCSS error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	r := rules[0]
	if len(r.Selectors) != 1 || r.Selectors[0].Raw != "p" {
		t.Errorf("expected selector 'p', got %v", r.Selectors)
	}
	if len(r.Declarations) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(r.Declarations))
	}
	if r.Declarations[0].Property != "color" || r.Declarations[0].Value != "red" {
		t.Errorf("unexpected declaration: %+v", r.Declarations[0])
	}
}

func TestParseCSS_ClassSelector(t *testing.T) {
	css := `.highlight { background-color: #ffff00; }`
	rules, err := ParseCSS(css)
	if err != nil {
		t.Fatalf("ParseCSS error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	sel := rules[0].Selectors[0]
	if len(sel.Parts) == 0 || len(sel.Parts[0].Classes) == 0 || sel.Parts[0].Classes[0] != "highlight" {
		t.Errorf("expected class selector '.highlight', got parts: %+v", sel.Parts)
	}
}

func TestParseCSS_IDSelector(t *testing.T) {
	css := `#total { font-weight: bold; }`
	rules, err := ParseCSS(css)
	if err != nil || len(rules) != 1 {
		t.Fatalf("ParseCSS failed: err=%v rules=%d", err, len(rules))
	}
	sel := rules[0].Selectors[0]
	if len(sel.Parts) == 0 || sel.Parts[0].ID != "total" {
		t.Errorf("expected ID 'total', got %+v", sel.Parts)
	}
}

func TestParseCSS_DescendantSelector(t *testing.T) {
	css := `table td { padding: 4pt; }`
	rules, err := ParseCSS(css)
	if err != nil || len(rules) != 1 {
		t.Fatalf("ParseCSS failed: %v", err)
	}
	sel := rules[0].Selectors[0]
	if len(sel.Parts) != 2 {
		t.Fatalf("expected 2 selector parts, got %d: %+v", len(sel.Parts), sel.Parts)
	}
	if sel.Parts[0].Tag != "table" {
		t.Errorf("expected first part tag 'table', got %q", sel.Parts[0].Tag)
	}
	if sel.Parts[1].Tag != "td" {
		t.Errorf("expected second part tag 'td', got %q", sel.Parts[1].Tag)
	}
}

func TestParseCSS_CompoundSelector(t *testing.T) {
	css := `td.right { text-align: right; }`
	rules, err := ParseCSS(css)
	if err != nil || len(rules) != 1 {
		t.Fatalf("ParseCSS failed: %v", err)
	}
	sel := rules[0].Selectors[0]
	if len(sel.Parts) == 0 {
		t.Fatal("expected at least one selector part")
	}
	part := sel.Parts[len(sel.Parts)-1]
	if part.Tag != "td" {
		t.Errorf("expected tag 'td', got %q", part.Tag)
	}
	if len(part.Classes) == 0 || part.Classes[0] != "right" {
		t.Errorf("expected class 'right', got %v", part.Classes)
	}
}

func TestParseCSS_MultipleSelectors(t *testing.T) {
	css := `h1, h2, h3 { font-weight: bold; }`
	rules, err := ParseCSS(css)
	if err != nil || len(rules) != 1 {
		t.Fatalf("ParseCSS failed: %v", err)
	}
	if len(rules[0].Selectors) != 3 {
		t.Errorf("expected 3 selectors, got %d", len(rules[0].Selectors))
	}
}

func TestParseCSS_PageRule(t *testing.T) {
	css := `page { size: A4; margin: 20mm; }`
	rules, err := ParseCSS(css)
	if err != nil {
		t.Fatalf("ParseCSS error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	found := false
	for _, d := range rules[0].Declarations {
		if d.Property == "size" && d.Value == "A4" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'size: A4' declaration in page rule")
	}
}

func TestParseCSS_Comments(t *testing.T) {
	css := `/* header styles */
p { /* inline comment */ color: blue; }`
	rules, err := ParseCSS(css)
	if err != nil {
		t.Fatalf("ParseCSS error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Declarations[0].Value != "blue" {
		t.Errorf("expected value 'blue', got %q", rules[0].Declarations[0].Value)
	}
}

func TestParseCSS_Specificity(t *testing.T) {
	tests := []struct {
		selector string
		wantA    int // ID
		wantB    int // class
		wantC    int // element
	}{
		{"p", 0, 0, 1},
		{".foo", 0, 1, 0},
		{"#bar", 1, 0, 0},
		{"table td", 0, 0, 2},
		{"td.right", 0, 1, 1},
		{"#id .class p", 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			sels, err := parseSelectors(tt.selector)
			if err != nil || len(sels) == 0 {
				t.Fatalf("parseSelectors(%q) failed: %v", tt.selector, err)
			}
			sp := sels[0].Specificity
			if sp.A != tt.wantA || sp.B != tt.wantB || sp.C != tt.wantC {
				t.Errorf("specificity(%q) = {%d,%d,%d}, want {%d,%d,%d}",
					tt.selector, sp.A, sp.B, sp.C, tt.wantA, tt.wantB, tt.wantC)
			}
		})
	}
}
