package layout

import (
	"testing"

	"github.com/grahms/papyrus/pkg/style"
)

// simpleMeasurer approximates character width as fontSize * 0.6 per character.
func simpleMeasurer(text string, cs style.ComputedStyle) float64 {
	return float64(len([]rune(text))) * cs.FontSize * 0.6
}

func makeStyle(fontSize, lineHeight float64) style.ComputedStyle {
	return style.ComputedStyle{
		FontSize:   fontSize,
		LineHeight: lineHeight,
	}
}

func TestBreakIntoLines_SingleRun(t *testing.T) {
	cs := makeStyle(10, 12)
	runs := []InlineRun{{Text: "Hello", Style: cs}}
	lines := BreakIntoLines(runs, 200, simpleMeasurer)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if len(lines[0].Runs) != 1 {
		t.Fatalf("expected 1 run in line, got %d", len(lines[0].Runs))
	}
	if lines[0].Runs[0].Text != "Hello" {
		t.Errorf("expected run text 'Hello', got %q", lines[0].Runs[0].Text)
	}
}

func TestBreakIntoLines_WordWrap(t *testing.T) {
	// Each character is 10pt * 0.6 = 6pt wide. "Hello" = 30pt, "World" = 30pt.
	// maxWidth = 50pt, so "Hello World" won't fit on one line.
	cs := makeStyle(10, 12)
	runs := []InlineRun{{Text: "Hello World", Style: cs}}
	lines := BreakIntoLines(runs, 50, simpleMeasurer)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	// First line should contain "Hello", second "World"
	firstText := ""
	for _, r := range lines[0].Runs {
		firstText += r.Text
	}
	secondText := ""
	for _, r := range lines[1].Runs {
		secondText += r.Text
	}
	if firstText != "Hello" {
		t.Errorf("first line: expected 'Hello', got %q", firstText)
	}
	if secondText != "World" {
		t.Errorf("second line: expected 'World', got %q", secondText)
	}
}

func TestBreakIntoLines_Nowrap(t *testing.T) {
	cs := makeStyle(10, 12)
	cs.WhiteSpace = "nowrap"
	// Even with a tiny maxWidth, nowrap should keep everything on one line.
	runs := []InlineRun{{Text: "Hello World Foo Bar", Style: cs}}
	lines := BreakIntoLines(runs, 10, simpleMeasurer)
	if len(lines) != 1 {
		t.Fatalf("nowrap: expected 1 line, got %d", len(lines))
	}
}

func TestBreakIntoLines_HardBreak(t *testing.T) {
	cs := makeStyle(10, 12)
	runs := []InlineRun{
		{Text: "Line one", Style: cs},
		{Text: "\n", Style: cs},
		{Text: "Line two", Style: cs},
	}
	lines := BreakIntoLines(runs, 500, simpleMeasurer)
	if len(lines) != 2 {
		t.Fatalf("hard break: expected 2 lines, got %d", len(lines))
	}
}

func TestLineHeight_SingleFont(t *testing.T) {
	cs := makeStyle(10, 14)
	line := Line{
		Runs: []InlineRun{{Text: "Hello", Style: cs}},
	}
	h := lineHeight(&line, cs)
	if h != 14 {
		t.Errorf("expected line height 14, got %g", h)
	}
	if line.MaxFontSize != 10 {
		t.Errorf("expected MaxFontSize 10, got %g", line.MaxFontSize)
	}
}

func TestLineHeight_MixedFonts_BaselineAligned(t *testing.T) {
	// A 10pt run and a 7.5pt <small>/<sup> run.
	bigCS := makeStyle(10, 12)
	smallCS := makeStyle(7.5, 9)
	line := Line{
		Runs: []InlineRun{
			{Text: "Normal", Style: bigCS},
			{Text: "small", Style: smallCS},
		},
	}
	defaultCS := bigCS
	lineHeight(&line, defaultCS)
	// MaxFontSize must be the largest font (10pt), not the smallest.
	if line.MaxFontSize != 10 {
		t.Errorf("expected MaxFontSize 10, got %g", line.MaxFontSize)
	}
}

func TestLineHeight_Superscript_ExtraHeight(t *testing.T) {
	// A 10pt base run and a superscript with positive BaselineShift.
	baseCS := makeStyle(10, 12)
	supCS := makeStyle(7.5, 9)
	supCS.BaselineShift = 5 // positive: shifts text up

	line := Line{
		Runs: []InlineRun{
			{Text: "Base", Style: baseCS},
			{Text: "sup", Style: supCS},
		},
	}
	h := lineHeight(&line, baseCS)
	// With maxPosBs = 5 and maxFs = 10:
	// needed = 10*0.9 + 5 + 10*0.3 = 9 + 5 + 3 = 17
	// baseLH = 12; 17 > 12, so h should be 17.
	const needed = 10*0.9 + 5 + 10*0.3
	if h < needed-1e-9 {
		t.Errorf("superscript: expected line height >= %g, got %g", needed, h)
	}
}

func TestBreakIntoLines_MaxFontSizeSet(t *testing.T) {
	// Verify that MaxFontSize is correctly propagated through BreakIntoLines.
	bigCS := makeStyle(10, 12)
	smallCS := makeStyle(7.5, 9)
	runs := []InlineRun{
		{Text: "Normal ", Style: bigCS},
		{Text: "small", Style: smallCS},
	}
	lines := BreakIntoLines(runs, 500, simpleMeasurer)
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
	if lines[0].MaxFontSize != 10 {
		t.Errorf("expected MaxFontSize 10, got %g", lines[0].MaxFontSize)
	}
}
