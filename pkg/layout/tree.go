package layout

import (
	"strings"

	"github.com/grahms/pdfml/pkg/parser"
	"github.com/grahms/pdfml/pkg/style"
)

// BuildBoxTree converts the styled DOM tree into a layout box tree.
// It returns the root box corresponding to the <body> element.
func BuildBoxTree(doc *parser.Document, styles map[*parser.Node]style.ComputedStyle) *Box {
	if doc == nil || doc.Root == nil {
		return nil
	}

	body := parser.FindElement(doc.Root, "body")
	if body == nil {
		// Fallback: use root
		body = doc.Root
	}

	bodyStyle, ok := styles[body]
	if !ok {
		bodyStyle = style.DefaultStyle(96)
	}

	root := &Box{
		Type:  BlockBox,
		Node:  body,
		Style: bodyStyle,
	}

	for _, child := range body.Children {
		buildNode(child, root, styles)
	}

	return root
}

// buildNode recursively builds box tree nodes from DOM nodes.
func buildNode(node *parser.Node, parent *Box, styles map[*parser.Node]style.ComputedStyle) {
	if node == nil {
		return
	}

	cs, ok := styles[node]
	if !ok {
		cs = style.DefaultStyle(96)
	}

	// Skip display:none elements
	if cs.Display == "none" {
		return
	}

	switch node.Type {
	case parser.TextNode:
		var text string
		if parent.Style.WhiteSpace == "pre" {
			// Preserve all whitespace inside <pre>
			text = node.Text
		} else {
			text = collapseTextWhitespace(node.Text)
		}
		if text == "" {
			return
		}
		box := &Box{
			Type:  TextBox,
			Node:  node,
			Style: cs,
			Text:  text,
		}
		parent.Children = append(parent.Children, box)

	case parser.ElementNode:
		box := buildElementBox(node, cs, styles)
		if box != nil {
			parent.Children = append(parent.Children, box)
		}
	}
}

// buildElementBox creates a Box for a DOM element.
func buildElementBox(node *parser.Node, cs style.ComputedStyle, styles map[*parser.Node]style.ComputedStyle) *Box {
	var boxType BoxType

	switch cs.Display {
	case "none":
		return nil
	case "inline":
		boxType = InlineBox
	case "table":
		boxType = TableBox
	case "table-row":
		boxType = TableRowBox
	case "table-cell":
		boxType = TableCellBox
	default:
		boxType = BlockBox
	}

	// Special handling for certain elements
	switch node.Tag {
	case "hr":
		return &Box{Type: HRBox, Node: node, Style: cs}
	case "img":
		src, _ := node.GetAttribute("src")
		box := &Box{Type: ImageBox, Node: node, Style: cs, ImageSrc: src}
		if w, ok := node.GetAttribute("width"); ok {
			box.ImgWidth = parsePtAttr(w)
		}
		if h, ok := node.GetAttribute("height"); ok {
			box.ImgHeight = parsePtAttr(h)
		}
		return box
	case "page-break":
		return &Box{Type: PageBreakBox, Node: node, Style: cs}
	case "head", "meta", "style", "var", "font":
		return nil
	case "page-header", "page-footer", "first-header", "first-footer":
		return nil
	case "a":
		href, _ := node.GetAttribute("href")
		box := &Box{Type: InlineBox, Node: node, Style: cs, HREF: href}
		for _, child := range node.Children {
			buildNode(child, box, styles)
		}
		return box
	case "br":
		return &Box{Type: InlineBox, Node: node, Style: cs, Text: "\n"}
	case "page-number":
		return &Box{Type: InlineBox, Node: node, Style: cs, Text: "{{PAGE}}"}
	case "page-count":
		return &Box{Type: InlineBox, Node: node, Style: cs, Text: "{{PAGES}}"}

	// <q> wraps its content with quotation marks
	case "q":
		box := &Box{Type: InlineBox, Node: node, Style: cs}
		open := &Box{Type: TextBox, Node: node, Style: cs, Text: "\u201c"}
		close := &Box{Type: TextBox, Node: node, Style: cs, Text: "\u201d"}
		box.Children = append(box.Children, open)
		for _, child := range node.Children {
			buildNode(child, box, styles)
		}
		box.Children = append(box.Children, close)
		return box

		// <dl>/<dt>/<dd> are rendered as block elements with UA-stylesheet indentation
		// applied via applyElementDefaults; no special box type needed.
		// <figure>, <figcaption>, <caption>, <pre> and the semantic containers
		// (<main>, <article>, <aside>, <nav>) are all plain block boxes handled
		// by the default path below.
	}

	box := &Box{
		Type:  boxType,
		Node:  node,
		Style: cs,
	}

	// Build list markers for <li> children
	if node.Tag == "ul" || node.Tag == "ol" {
		isOrdered := node.Tag == "ol"
		counter := 0
		for _, child := range node.Children {
			if child.Type == parser.ElementNode && child.Tag == "li" {
				counter++
				childCS := styles[child]
				liBox := &Box{Type: BlockBox, Node: child, Style: childCS}
				if isOrdered {
					liBox.ListMarker = formatOrderedMarker(counter)
				} else {
					liBox.ListMarker = "•"
				}
				for _, liChild := range child.Children {
					buildNode(liChild, liBox, styles)
				}
				box.Children = append(box.Children, liBox)
			}
		}
		return box
	}

	for _, child := range node.Children {
		buildNode(child, box, styles)
	}

	// For table boxes, identify thead/tfoot children for pagination support
	if box.Type == TableBox {
		for _, child := range box.Children {
			if child.TableSectionType == TableSectionHead && box.TheadBox == nil {
				box.TheadBox = child
			}
			if child.TableSectionType == TableSectionFoot && box.TfootBox == nil {
				box.TfootBox = child
			}
		}
	}

	return box
}

// formatOrderedMarker returns the marker text for an ordered list item.
func formatOrderedMarker(n int) string {
	return formatInt(n) + "."
}

func formatInt(n int) string {
	if n <= 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// isInlineContext returns true if the parent box is in an inline formatting context.
func isInlineContext(parent *Box) bool {
	return parent.Type == InlineBox || parent.Type == TextBox
}

// collapseTextWhitespace normalises a text node's whitespace using CSS "normal" rules:
//   - collapse runs of internal whitespace to a single space
//   - preserve a single leading space if the raw text started with whitespace
//   - preserve a single trailing space if the raw text ended with whitespace
//   - return "" for nodes that are pure newlines (structural whitespace)
func collapseTextWhitespace(raw string) string {
	if raw == "" {
		return ""
	}
	// Pure-newline whitespace is structural — discard it.
	onlyNewlines := true
	for _, r := range raw {
		if r != '\n' && r != '\r' {
			onlyNewlines = false
			break
		}
	}
	if onlyNewlines {
		return ""
	}

	words := strings.Fields(raw)
	if len(words) == 0 {
		// Only spaces/tabs — keep as a single space (word-separator between siblings)
		return " "
	}

	result := strings.Join(words, " ")

	// Preserve a single leading space when the raw text started with a space/tab.
	first := raw[0]
	if first == ' ' || first == '\t' {
		result = " " + result
	}

	// Preserve a single trailing space when the raw text ended with a space/tab.
	last := raw[len(raw)-1]
	if last == ' ' || last == '\t' {
		result = result + " "
	}

	return result
}

// parsePtAttr tries to parse an attribute string as a pt value.
func parsePtAttr(s string) float64 {
	l := style.MustParseLength(s)
	return l.ToPoints(0, 10, 10, 96)
}
