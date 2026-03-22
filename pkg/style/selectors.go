package style

import (
	"strconv"
	"strings"

	"github.com/grahms/papyrus/pkg/parser"
)

// MatchSelector returns true if the given node matches the selector.
// This implements the full compound selector matching including combinators.
func MatchSelector(sel parser.Selector, node *parser.Node) bool {
	if len(sel.Parts) == 0 {
		return false
	}
	return matchParts(sel.Parts, node)
}

// matchParts recursively matches selector parts against a node.
// Parts are applied right-to-left; the last part must match the current node.
func matchParts(parts []parser.SelectorPart, node *parser.Node) bool {
	if len(parts) == 0 {
		return true
	}
	if node == nil || node.Type != parser.ElementNode {
		return false
	}

	last := parts[len(parts)-1]
	remaining := parts[:len(parts)-1]

	// Match the last part against the current node
	if !matchPart(last, node) {
		return false
	}

	if len(remaining) == 0 {
		return true
	}

	// Process combinator for the preceding part
	combinator := last.Combinator
	switch combinator {
	case ">":
		// Child combinator: parent must match remaining
		if node.Parent == nil || node.Parent.Type != parser.ElementNode {
			return false
		}
		return matchParts(remaining, node.Parent)
	case " ":
		// Descendant combinator: any ancestor must match remaining
		ancestor := node.Parent
		for ancestor != nil {
			if ancestor.Type == parser.ElementNode {
				if matchParts(remaining, ancestor) {
					return true
				}
			}
			ancestor = ancestor.Parent
		}
		return false
	default:
		// No combinator on first part, or unknown — descendant
		if len(remaining) > 0 {
			ancestor := node.Parent
			for ancestor != nil {
				if ancestor.Type == parser.ElementNode {
					if matchParts(remaining, ancestor) {
						return true
					}
				}
				ancestor = ancestor.Parent
			}
			return false
		}
		return true
	}
}

// matchPart returns true if the node matches a single selector part (no combinator).
func matchPart(part parser.SelectorPart, node *parser.Node) bool {
	if node.Type != parser.ElementNode {
		return false
	}

	// Special pseudo-tag for @page
	if part.Tag == "@page" {
		return false
	}

	// Tag check
	if part.Tag != "" && part.Tag != "*" {
		if !strings.EqualFold(node.Tag, part.Tag) {
			return false
		}
	}

	// ID check
	if part.ID != "" {
		if node.ID != part.ID {
			return false
		}
	}

	// Class checks
	for _, cls := range part.Classes {
		if !node.HasClass(cls) {
			return false
		}
	}

	// Pseudo-class check
	if part.PseudoClass != "" {
		if !matchPseudo(part, node) {
			return false
		}
	}

	return true
}

// matchPseudo checks pseudo-class conditions.
func matchPseudo(part parser.SelectorPart, node *parser.Node) bool {
	switch part.PseudoClass {
	case "first-child":
		return isNthChild(node, 1)
	case "last-child":
		return isLastChild(node)
	case "nth-child":
		return matchNthChild(part.NthArg, node)
	}
	return true
}

// isNthChild returns true if node is the nth child (1-based) of its parent.
func isNthChild(node *parser.Node, n int) bool {
	if node.Parent == nil {
		return n == 1
	}
	count := 0
	for _, child := range node.Parent.Children {
		if child.Type == parser.ElementNode {
			count++
			if child == node {
				return count == n
			}
		}
	}
	return false
}

// isLastChild returns true if node is the last element child of its parent.
func isLastChild(node *parser.Node) bool {
	if node.Parent == nil {
		return true
	}
	var lastElem *parser.Node
	for _, child := range node.Parent.Children {
		if child.Type == parser.ElementNode {
			lastElem = child
		}
	}
	return lastElem == node
}

// matchNthChild parses the nth-child argument and checks the node's position.
// Supports: "odd", "even", a number, or An+B form.
func matchNthChild(arg string, node *parser.Node) bool {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return false
	}

	// Get 1-based index
	idx := childIndex(node)
	if idx < 0 {
		return false
	}

	switch strings.ToLower(arg) {
	case "odd":
		return idx%2 == 1
	case "even":
		return idx%2 == 0
	}

	// Pure number
	if n, err := strconv.Atoi(arg); err == nil {
		return idx == n
	}

	// An+B form
	a, b := parseNthAB(arg)
	if a == 0 {
		return idx == b
	}
	// idx = a*n + b => (idx - b) must be divisible by a and >= 0
	diff := idx - b
	if diff < 0 {
		return false
	}
	return diff%a == 0
}

func childIndex(node *parser.Node) int {
	if node.Parent == nil {
		return 1
	}
	count := 0
	for _, child := range node.Parent.Children {
		if child.Type == parser.ElementNode {
			count++
			if child == node {
				return count
			}
		}
	}
	return -1
}

// parseNthAB parses "An+B" form, returns a and b.
func parseNthAB(s string) (a, b int) {
	s = strings.ReplaceAll(s, " ", "")
	plusIdx := strings.LastIndex(s, "+")
	if plusIdx < 0 {
		// Only A*n
		s = strings.TrimSuffix(s, "n")
		a, _ = strconv.Atoi(s)
		return a, 0
	}
	aPart := strings.TrimSuffix(s[:plusIdx], "n")
	bPart := s[plusIdx+1:]
	a, _ = strconv.Atoi(aPart)
	b, _ = strconv.Atoi(bPart)
	return a, b
}
