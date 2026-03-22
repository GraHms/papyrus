package layout

import (
	"strings"

	"github.com/ismaelvodacom/goxml2pdf/pkg/parser"
	"github.com/ismaelvodacom/goxml2pdf/pkg/style"
)

// Line represents a single rendered line of text.
type Line struct {
	Runs   []InlineRun
	Width  float64
	Height float64 // line height (ascent + descent of tallest run)
}

// TextMeasurer is a function that measures the width of a string in a given style.
type TextMeasurer func(text string, cs style.ComputedStyle) float64

// BreakIntoLines takes a list of inline runs and breaks them into lines
// that fit within maxWidth using a greedy algorithm.
func BreakIntoLines(runs []InlineRun, maxWidth float64, measurer TextMeasurer) []Line {
	if len(runs) == 0 {
		return nil
	}

	// Flatten all runs into a sequence of atomic tokens (words / spaces / newlines),
	// each carrying its style.
	type token struct {
		text  string
		style style.ComputedStyle
		node  *parser.Node
	}
	var tokens []token
	for _, run := range runs {
		if run.Text == "\n" {
			tokens = append(tokens, token{"\n", run.Style, run.Node})
			continue
		}
		// Split text into words; preserve inter-word spaces as explicit space tokens.
		parts := strings.Fields(run.Text)
		hasLeading := len(run.Text) > 0 && (run.Text[0] == ' ' || run.Text[0] == '\t')
		hasTrailing := len(run.Text) > 0 && (run.Text[len(run.Text)-1] == ' ' || run.Text[len(run.Text)-1] == '\t')

		if len(parts) == 0 {
			// Pure whitespace run — emit a single space token.
			tokens = append(tokens, token{" ", run.Style, run.Node})
			continue
		}
		for i, word := range parts {
			if i == 0 && hasLeading {
				tokens = append(tokens, token{" ", run.Style, run.Node})
			}
			tokens = append(tokens, token{word, run.Style, run.Node})
			if i < len(parts)-1 {
				tokens = append(tokens, token{" ", run.Style, run.Node})
			}
			if i == len(parts)-1 && hasTrailing {
				tokens = append(tokens, token{" ", run.Style, run.Node})
			}
		}
	}

	var lines []Line
	var currentLine Line
	currentWidth := 0.0
	lastStyle := runs[0].Style

	flushLine := func(s style.ComputedStyle) {
		currentLine.Height = lineHeight(currentLine, s)
		lines = append(lines, currentLine)
		currentLine = Line{}
		currentWidth = 0
	}

	for _, tok := range tokens {
		lastStyle = tok.style

		if tok.text == "\n" {
			flushLine(tok.style)
			continue
		}

		tokW := measurer(tok.text, tok.style)

		// Leading spaces at start of a line are swallowed.
		if tok.text == " " && currentWidth == 0 {
			continue
		}

		if currentWidth+tokW <= maxWidth || currentWidth == 0 {
			currentLine.Runs = append(currentLine.Runs, InlineRun{
				Text:  tok.text,
				Style: tok.style,
				Node:  tok.node,
			})
			currentWidth += tokW
			currentLine.Width = currentWidth
		} else {
			// Token doesn't fit — wrap.
			// Trim trailing space from current line before wrapping.
			if len(currentLine.Runs) > 0 {
				last := &currentLine.Runs[len(currentLine.Runs)-1]
				if last.Text == " " {
					currentLine.Runs = currentLine.Runs[:len(currentLine.Runs)-1]
				}
			}
			flushLine(tok.style)

			if tok.text == " " {
				continue // don't start new line with a space
			}
			currentLine.Runs = append(currentLine.Runs, InlineRun{
				Text:  tok.text,
				Style: tok.style,
				Node:  tok.node,
			})
			currentWidth = tokW
			currentLine.Width = currentWidth
		}
	}

	// Flush final line.
	if len(currentLine.Runs) > 0 {
		currentLine.Height = lineHeight(currentLine, lastStyle)
		lines = append(lines, currentLine)
	}

	return lines
}

// lineHeight computes the line height for a line based on the tallest run.
func lineHeight(line Line, defaultStyle style.ComputedStyle) float64 {
	maxH := defaultStyle.LineHeight
	for _, run := range line.Runs {
		if run.Style.LineHeight > maxH {
			maxH = run.Style.LineHeight
		}
	}
	if maxH <= 0 {
		maxH = defaultStyle.FontSize * 1.2
	}
	return maxH
}

// CollectInlineRuns gathers all inline content from a block box into a flat list of runs.
func CollectInlineRuns(box *Box, baseStyle style.ComputedStyle) []InlineRun {
	var runs []InlineRun
	collectRuns(box, baseStyle, &runs)
	return runs
}

func collectRuns(box *Box, parentStyle style.ComputedStyle, runs *[]InlineRun) {
	if box == nil {
		return
	}

	switch box.Type {
	case TextBox:
		if box.Text != "" {
			*runs = append(*runs, InlineRun{
				Text:  box.Text,
				Style: box.Style,
				Node:  box.Node,
			})
		}
	case InlineBox:
		// Boxes with direct text (br, page-number, page-count, etc.)
		if box.Text != "" {
			*runs = append(*runs, InlineRun{Text: box.Text, Style: box.Style, Node: box.Node})
			return
		}
		for _, child := range box.Children {
			collectRuns(child, box.Style, runs)
		}
	default:
		for _, child := range box.Children {
			if child.Type == TextBox || child.Type == InlineBox {
				collectRuns(child, box.Style, runs)
			}
		}
	}
}
