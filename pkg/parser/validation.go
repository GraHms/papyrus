package parser

import "fmt"

// ValidateElement checks that the given element tag is in the allowed vocabulary.
// Returns an error with line and column if the element is not allowed.
func ValidateElement(tag string, line, col int) error {
	if !AllowedElements[tag] {
		return fmt.Errorf("parser: unknown element <%s> at line %d col %d", tag, line, col)
	}
	return nil
}

// ValidateDocument walks the DOM and validates all elements.
func ValidateDocument(doc *Document) error {
	if doc == nil || doc.Root == nil {
		return fmt.Errorf("parser: document is nil")
	}
	return validateNode(doc.Root)
}

func validateNode(n *Node) error {
	if n.Type == ElementNode {
		if !AllowedElements[n.Tag] {
			return fmt.Errorf("parser: unknown element <%s> at line %d col %d", n.Tag, n.Line, n.Col)
		}
	}
	for _, child := range n.Children {
		if err := validateNode(child); err != nil {
			return err
		}
	}
	return nil
}
