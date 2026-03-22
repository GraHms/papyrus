// Package parser handles XML and CSS parsing for pdfml documents.
package parser

// NodeType represents the type of a DOM node.
type NodeType int

const (
	// DocumentNode is the root of the document tree.
	DocumentNode NodeType = iota
	// ElementNode represents an XML element like <p>, <h1>, <table>.
	ElementNode
	// TextNode represents raw text content within an element.
	TextNode
)

// Node represents a single node in the document tree.
// Every node carries its source position for error reporting.
type Node struct {
	Type       NodeType
	Tag        string            // Element tag name (empty for text nodes)
	Attributes map[string]string // Element attributes
	Children   []*Node
	Parent     *Node
	Text       string // Text content (only for TextNode)
	Line       int    // Source line number
	Col        int    // Source column number

	// Classes and ID are extracted from attributes for fast access.
	ID      string
	Classes []string
}

// NewElement creates a new element node.
func NewElement(tag string, line, col int) *Node {
	return &Node{
		Type:       ElementNode,
		Tag:        tag,
		Attributes: make(map[string]string),
		Line:       line,
		Col:        col,
	}
}

// NewText creates a new text node.
func NewText(text string, line, col int) *Node {
	return &Node{
		Type: TextNode,
		Text: text,
		Line: line,
		Col:  col,
	}
}

// AppendChild adds a child node and sets the parent reference.
func (n *Node) AppendChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// GetAttribute returns an attribute value and whether it exists.
func (n *Node) GetAttribute(name string) (string, bool) {
	v, ok := n.Attributes[name]
	return v, ok
}

// SetAttribute sets an attribute and updates ID/Classes if relevant.
func (n *Node) SetAttribute(name, value string) {
	n.Attributes[name] = value
	switch name {
	case "id":
		n.ID = value
	case "class":
		n.Classes = splitClasses(value)
	}
}

// HasClass returns true if the element has the given class.
func (n *Node) HasClass(class string) bool {
	for _, c := range n.Classes {
		if c == class {
			return true
		}
	}
	return false
}

// splitClasses splits a class attribute value into individual class names.
func splitClasses(value string) []string {
	var classes []string
	current := ""
	for _, r := range value {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current != "" {
				classes = append(classes, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		classes = append(classes, current)
	}
	return classes
}

// HTMLAliases maps HTML tag names to their canonical pdfml equivalents.
// Alias normalisation happens at parse time in xml.go; all other packages
// only ever see canonical tag names.
var HTMLAliases = map[string]string{
	"html":   "document",
	"header": "page-header",
	"footer": "page-footer",
}

// AllowedElements defines the valid element names in the pdfml vocabulary.
// This set uses canonical tag names only (after alias normalisation).
var AllowedElements = map[string]bool{
	// Document structure
	"document": true, "head": true, "body": true,
	"style": true, "meta": true, "var": true, "font": true,

	// Page control
	"page-header": true, "page-footer": true,
	"page-break": true, "page-number": true, "page-count": true,

	// Block content
	"section": true, "div": true,
	"main": true, "article": true, "aside": true, "nav": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"p": true, "blockquote": true, "hr": true,
	"pre": true,
	"figure": true, "figcaption": true,

	// Inline content
	"span": true, "strong": true, "b": true, "em": true, "i": true,
	"u": true, "s": true, "code": true, "br": true, "a": true,
	"mark": true, "small": true, "sub": true, "sup": true,
	"cite": true, "q": true,

	// Tables
	"table": true, "caption": true,
	"thead": true, "tbody": true, "tfoot": true,
	"tr": true, "td": true, "th": true,

	// Media
	"img": true,

	// Lists
	"ul": true, "ol": true, "li": true,
	"dl": true, "dt": true, "dd": true,
}

// InlineElements are elements that participate in inline formatting.
var InlineElements = map[string]bool{
	"span": true, "strong": true, "b": true, "em": true, "i": true,
	"u": true, "s": true, "code": true, "br": true, "a": true,
	"mark": true, "small": true, "sub": true, "sup": true,
	"cite": true, "q": true,
	"page-number": true, "page-count": true,
}

// IsInline returns true if this element is an inline-level element.
func (n *Node) IsInline() bool {
	if n.Type == TextNode {
		return true
	}
	return InlineElements[n.Tag]
}
