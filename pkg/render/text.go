package render

import (
	"strings"
	"unicode/utf8"

	"github.com/grahms/pdfml/pkg/layout"
	"github.com/grahms/pdfml/pkg/style"
	"github.com/signintech/gopdf"
)

// MeasureText returns the width of a text string in the given style using gopdf.
// This is used as the layout.TextMeasurer.
func MeasureText(pdf *gopdf.GoPdf, fm *FontManager) layout.TextMeasurer {
	return func(text string, cs style.ComputedStyle) float64 {
		bold := cs.IsBold()
		italic := cs.IsItalic()

		fontName, err := fm.EnsureFont(cs.FontFamily, bold, italic)
		if err != nil {
			fontName, _ = fm.EnsureFont("Liberation Sans", false, false)
		}

		// Set font to measure
		if err := pdf.SetFont(fontName, "", cs.FontSize); err != nil {
			return float64(len([]rune(text))) * cs.FontSize * 0.5 // rough fallback
		}

		w, err := pdf.MeasureTextWidth(text)
		if err != nil {
			return float64(len([]rune(text))) * cs.FontSize * 0.5
		}
		if cs.LetterSpacing != 0 {
			w += float64(utf8.RuneCountInString(text)) * cs.LetterSpacing
		}
		return w
	}
}

// drawTextRuns renders inline text runs at the given position.
// It handles line-by-line rendering based on the inline run data.
func drawTextRuns(pdf *gopdf.GoPdf, fm *FontManager, runs []layout.InlineRun, x, y, maxWidth float64, cs style.ComputedStyle) float64 {
	if len(runs) == 0 {
		return y
	}

	measurer := MeasureText(pdf, fm)

	// Re-break runs into lines for rendering
	lines := layout.BreakIntoLines(runs, maxWidth, measurer)

	curY := y
	for i, line := range lines {
		if len(line.Runs) == 0 {
			// Empty line (from br)
			lh := cs.LineHeight
			if lh <= 0 {
				lh = cs.FontSize * 1.2
			}
			curY += lh
			continue
		}

		// Compute line height
		lineH := line.Height
		if lineH <= 0 {
			lineH = cs.LineHeight
			if lineH <= 0 {
				lineH = cs.FontSize * 1.2
			}
		}

		// Compute line width for alignment
		lineWidth := measureLine(pdf, fm, line.Runs)
		lineX := computeAlignX(x, maxWidth, lineWidth, cs.TextAlign)

		curX := lineX

		// Justify: compute extra space per gap for non-last lines
		isLastLine := i == len(lines)-1
		justify := cs.TextAlign == "justify" && !isLastLine
		var extraSpacePerGap float64
		if justify {
			spaceCount := 0
			for _, r := range line.Runs {
				if r.Text == " " {
					spaceCount++
				}
			}
			if spaceCount > 0 {
				extraSpacePerGap = (maxWidth - line.Width) / float64(spaceCount)
			}
		}

		for _, run := range line.Runs {
			if run.Text == "" || run.Text == "\n" {
				continue
			}

			runCS := run.Style
			bold := runCS.IsBold()
			italic := runCS.IsItalic()

			// Apply font
			if err := fm.SetFont(runCS.FontFamily, bold, italic, runCS.FontSize); err != nil {
				continue
			}

			// Apply text color
			setTextColor(pdf, runCS.Color)

			// Render text
			runText := applyTextTransform(run.Text, runCS.TextTransform)

			// Handle justify space advancement without drawing
			if run.Text == " " && justify {
				spW, _ := pdf.MeasureTextWidth(" ")
				curX += spW + extraSpacePerGap
				continue
			}

			// Draw text at current position.
			// line.MaxFontSize*0.9 establishes a single reference baseline for the entire
			// line regardless of per-run font sizes; BaselineShift offsets sup/sub from it.
			textY := curY + lineH - line.MaxFontSize*0.9 - runCS.BaselineShift

			var w float64
			if runCS.LetterSpacing != 0 {
				// Render each rune individually with letter-spacing applied between chars.
				runX := curX
				for _, ch := range runText {
					chStr := string(ch)
					pdf.SetXY(runX, textY)
					_ = pdf.Text(chStr)
					chW, _ := pdf.MeasureTextWidth(chStr)
					runX += chW + runCS.LetterSpacing
				}
				w = runX - curX
			} else {
				pdf.SetXY(curX, textY)
				_ = pdf.Text(runText)
				w, _ = pdf.MeasureTextWidth(runText)
			}
			curX += w

			// Underline
			if runCS.TextDecoration == "underline" {
				underlineY := curY + lineH - runCS.FontSize*0.1
				setStrokeColor(pdf, runCS.Color)
				pdf.SetLineWidth(runCS.FontSize * 0.05)
				pdf.Line(curX-w, underlineY, curX, underlineY)
			}

			// Line-through
			if runCS.TextDecoration == "line-through" {
				strikeY := curY + lineH*0.5
				setStrokeColor(pdf, runCS.Color)
				pdf.SetLineWidth(runCS.FontSize * 0.05)
				pdf.Line(curX-w, strikeY, curX, strikeY)
			}

			// Link annotation for <a href="..."> runs.
			// AddExternalLink(url, x, y, w, h) uses top-left coordinates in pt.
			if run.HREF != "" {
				pdf.AddExternalLink(run.HREF, curX-w, textY, w, runCS.FontSize)
			}
		}

		curY += lineH
	}

	return curY
}

func measureLine(pdf *gopdf.GoPdf, fm *FontManager, runs []layout.InlineRun) float64 {
	total := 0.0
	for _, run := range runs {
		if run.Text == "" || run.Text == "\n" {
			continue
		}
		if err := fm.SetFont(run.Style.FontFamily, run.Style.IsBold(), run.Style.IsItalic(), run.Style.FontSize); err != nil {
			continue
		}
		w, err := pdf.MeasureTextWidth(run.Text)
		if err == nil {
			if run.Style.LetterSpacing != 0 {
				w += float64(utf8.RuneCountInString(run.Text)) * run.Style.LetterSpacing
			}
			total += w
		}
	}
	return total
}

func computeAlignX(x, maxWidth, lineWidth float64, align string) float64 {
	switch strings.ToLower(align) {
	case "center":
		return x + (maxWidth-lineWidth)/2
	case "right":
		return x + maxWidth - lineWidth
	default:
		return x
	}
}

func applyTextTransform(text, transform string) string {
	switch strings.ToLower(transform) {
	case "uppercase":
		return strings.ToUpper(text)
	case "lowercase":
		return strings.ToLower(text)
	case "capitalize":
		words := strings.Fields(text)
		for i, w := range words {
			if len(w) > 0 {
				words[i] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		return strings.Join(words, " ")
	}
	return text
}
