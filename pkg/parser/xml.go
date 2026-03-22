// Package parser handles XML and CSS parsing for goxml2pdf documents.
package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Document is the root of a parsed goxml2pdf document.
type Document struct {
	Root    *Node
	Meta    map[string]string
	Styles  string // raw CSS text from <style> blocks
	VarDefs map[string]string
}

// ParseXML parses an XML document from r into a Document.
func ParseXML(r io.Reader) (*Document, error) {
	doc := &Document{
		Meta:    make(map[string]string),
		VarDefs: make(map[string]string),
	}

	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose

	var stack []*Node
	var cssBuilder strings.Builder
	inStyle := false
	inHead := false

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Get position info
			if syntaxErr, ok := err.(*xml.SyntaxError); ok {
				return nil, fmt.Errorf("parser: XML syntax error at line %d: %w", syntaxErr.Line, err)
			}
			return nil, fmt.Errorf("parser: XML parse error: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			line, col := getPos(decoder)

			tag := strings.ToLower(t.Name.Local)

			// Validate element
			if err := ValidateElement(tag, line, col); err != nil {
				return nil, err
			}

			if tag == "head" {
				inHead = true
			}
			if tag == "style" {
				inStyle = true
				cssBuilder.Reset()
				continue
			}

			node := NewElement(tag, line, col)
			for _, attr := range t.Attr {
				attrName := strings.ToLower(attr.Name.Local)
				node.SetAttribute(attrName, attr.Value)
			}

			if tag == "meta" && inHead {
				// Capture metadata attributes
				for k, v := range node.Attributes {
					doc.Meta[k] = v
				}
				stack = append(stack, node)
				continue
			}

			if tag == "var" && inHead {
				name, _ := node.GetAttribute("name")
				value, _ := node.GetAttribute("value")
				if name != "" {
					doc.VarDefs[name] = value
				}
				stack = append(stack, node)
				continue
			}

			if tag == "document" {
				doc.Root = node
				stack = append(stack, node)
				continue
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.AppendChild(node)
			}
			stack = append(stack, node)

		case xml.EndElement:
			tag := strings.ToLower(t.Name.Local)

			if tag == "style" {
				inStyle = false
				doc.Styles += cssBuilder.String()
				continue
			}
			if tag == "head" {
				inHead = false
			}

			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			text := string(t)

			if inStyle {
				cssBuilder.WriteString(text)
				continue
			}

			// Only add non-whitespace-only text nodes to non-structure elements
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				continue
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				// Don't add text to structural elements
				if parent.Tag == "head" || parent.Tag == "document" || parent.Tag == "body" {
					// Allow text in body children, not body itself
				} else {
					line, col := getPos(decoder)
					textNode := NewText(text, line, col)
					parent.AppendChild(textNode)
				}
			}
		}
	}

	if doc.Root == nil {
		return nil, fmt.Errorf("parser: no <document> root element found")
	}

	return doc, nil
}

// getPos returns the current line and column from the decoder.
// xml.Decoder doesn't expose position directly after token reads,
// so we use a best-effort approach.
func getPos(d *xml.Decoder) (int, int) {
	// xml.Decoder doesn't expose a public position method for line/col after Token()
	// We return 0,0 as fallback; real position is captured in SyntaxErrors.
	return 0, 0
}

// FindElement finds the first child element with the given tag.
func FindElement(n *Node, tag string) *Node {
	if n == nil {
		return nil
	}
	for _, child := range n.Children {
		if child.Type == ElementNode && child.Tag == tag {
			return child
		}
	}
	return nil
}

// FindElements returns all descendant elements with the given tag.
func FindElements(n *Node, tag string) []*Node {
	var results []*Node
	walkNodes(n, func(node *Node) {
		if node.Type == ElementNode && node.Tag == tag {
			results = append(results, node)
		}
	})
	return results
}

// walkNodes recursively walks the node tree calling fn on each node.
func walkNodes(n *Node, fn func(*Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, child := range n.Children {
		walkNodes(child, fn)
	}
}
