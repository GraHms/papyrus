package style

import (
	"testing"

	"github.com/grahms/pdfml/pkg/parser"
)

func makeNode(tag, id, class string, parent *parser.Node) *parser.Node {
	n := parser.NewElement(tag, 0, 0)
	if id != "" {
		n.SetAttribute("id", id)
	}
	if class != "" {
		n.SetAttribute("class", class)
	}
	if parent != nil {
		parent.AppendChild(n)
	}
	return n
}

func parseFirstSelector(t *testing.T, css string) parser.Selector {
	t.Helper()
	rules, err := parser.ParseCSS(css + `{}`)
	if err != nil || len(rules) == 0 || len(rules[0].Selectors) == 0 {
		t.Fatalf("failed to parse selector from %q: %v", css, err)
	}
	return rules[0].Selectors[0]
}

func TestMatchSelector_Element(t *testing.T) {
	sel := parseFirstSelector(t, "p")
	p := makeNode("p", "", "", nil)
	div := makeNode("div", "", "", nil)

	if !MatchSelector(sel, p) {
		t.Error("expected p to match selector 'p'")
	}
	if MatchSelector(sel, div) {
		t.Error("expected div NOT to match selector 'p'")
	}
}

func TestMatchSelector_Class(t *testing.T) {
	sel := parseFirstSelector(t, ".highlight")
	n1 := makeNode("p", "", "highlight", nil)
	n2 := makeNode("p", "", "other", nil)
	n3 := makeNode("p", "", "highlight extra", nil)

	if !MatchSelector(sel, n1) {
		t.Error("expected 'highlight' class to match")
	}
	if MatchSelector(sel, n2) {
		t.Error("expected 'other' NOT to match '.highlight'")
	}
	if !MatchSelector(sel, n3) {
		t.Error("expected 'highlight extra' to match '.highlight'")
	}
}

func TestMatchSelector_ID(t *testing.T) {
	sel := parseFirstSelector(t, "#total")
	n1 := makeNode("p", "total", "", nil)
	n2 := makeNode("p", "other", "", nil)

	if !MatchSelector(sel, n1) {
		t.Error("expected id='total' to match '#total'")
	}
	if MatchSelector(sel, n2) {
		t.Error("expected id='other' NOT to match '#total'")
	}
}

func TestMatchSelector_Descendant(t *testing.T) {
	sel := parseFirstSelector(t, "table td")
	table := makeNode("table", "", "", nil)
	tbody := makeNode("tbody", "", "", table)
	td := makeNode("td", "", "", tbody)
	p := makeNode("p", "", "", nil)

	if !MatchSelector(sel, td) {
		t.Error("expected td inside table to match 'table td'")
	}
	if MatchSelector(sel, p) {
		t.Error("expected standalone p NOT to match 'table td'")
	}
}

func TestMatchSelector_ChildCombinator(t *testing.T) {
	sel := parseFirstSelector(t, "table > tr")
	table := makeNode("table", "", "", nil)
	tr := makeNode("tr", "", "", table)
	tbody := makeNode("tbody", "", "", table)
	tr2 := makeNode("tr", "", "", tbody) // tr inside tbody, not direct child of table

	if !MatchSelector(sel, tr) {
		t.Error("expected direct child tr to match 'table > tr'")
	}
	if MatchSelector(sel, tr2) {
		t.Error("expected nested tr NOT to match 'table > tr'")
	}
}

func TestMatchSelector_Compound(t *testing.T) {
	sel := parseFirstSelector(t, "td.right")
	td := makeNode("td", "", "right", nil)
	tdOther := makeNode("td", "", "left", nil)
	pRight := makeNode("p", "", "right", nil)

	if !MatchSelector(sel, td) {
		t.Error("expected td.right to match 'td.right'")
	}
	if MatchSelector(sel, tdOther) {
		t.Error("expected td.left NOT to match 'td.right'")
	}
	if MatchSelector(sel, pRight) {
		t.Error("expected p.right NOT to match 'td.right'")
	}
}

func TestMatchSelector_PseudoFirstChild(t *testing.T) {
	sel := parseFirstSelector(t, "li:first-child")
	ul := makeNode("ul", "", "", nil)
	li1 := makeNode("li", "", "", ul)
	li2 := makeNode("li", "", "", ul)

	if !MatchSelector(sel, li1) {
		t.Error("expected first li to match 'li:first-child'")
	}
	if MatchSelector(sel, li2) {
		t.Error("expected second li NOT to match 'li:first-child'")
	}
}

func TestMatchSelector_PseudoLastChild(t *testing.T) {
	sel := parseFirstSelector(t, "li:last-child")
	ul := makeNode("ul", "", "", nil)
	li1 := makeNode("li", "", "", ul)
	li2 := makeNode("li", "", "", ul)

	if MatchSelector(sel, li1) {
		t.Error("expected first li NOT to match 'li:last-child'")
	}
	if !MatchSelector(sel, li2) {
		t.Error("expected last li to match 'li:last-child'")
	}
}
