package layout

import (
	"fmt"
	"strings"
)

// DumpTreeToString recursively dumps a textual representation of the layout box tree,
// including absolute coordinates, dimensions, types, and raw text content.
// This is used for golden-file snapshot testing to catch layout regressions.
func DumpTreeToString(pageLayout *PageLayout) string {
	var sb strings.Builder

	for pIdx, page := range pageLayout.Pages {
		sb.WriteString(fmt.Sprintf("\n--- Page %d [%.2fx%.2f] ---\n", pIdx+1, page.Width, page.Height))
		if page.Header != nil {
			sb.WriteString(fmt.Sprintf("[HEADER] AbsY:%.2f\n", page.Header.AbsY))
			dumpBox(&sb, page.Header, 1)
		}
		if page.Footer != nil {
			sb.WriteString(fmt.Sprintf("[FOOTER] AbsY:%.2f\n", page.Footer.AbsY))
			dumpBox(&sb, page.Footer, 1)
		}

		sb.WriteString("[BODY]\n")
		for _, box := range page.Boxes {
			dumpBox(&sb, box, 1)
		}
	}

	return sb.String()
}

func dumpBox(sb *strings.Builder, box *Box, depth int) {
	if box == nil {
		return
	}
	indent := strings.Repeat("  ", depth)

	// Format: {Type} [Tag] <X:Y WxH> M[Top Right Bottom Left] P[Top Right Bottom Left]
	tag := "text"
	if box.Node != nil && box.Node.Tag != "" {
		tag = box.Node.Tag
	} else if box.Type == TextBox {
		tag = "text-node"
	}

	meta := ""
	if box.Type == InlineBox {
		meta = " INLINE"
	}

	cs := box.Style

	sb.WriteString(fmt.Sprintf("%s%s [%s]%s X:%.2f Y:%.2f W:%.2f H:%.2f ",
		indent, boxTypeString(box.Type), tag, meta,
		box.AbsX, box.AbsY, box.Width, box.Height))

	sb.WriteString(fmt.Sprintf("M[%.1f %.1f %.1f %.1f] P[%.1f %.1f %.1f %.1f] B[%.1f %.1f %.1f %.1f]",
		cs.MarginTop, cs.MarginRight, cs.MarginBottom, cs.MarginLeft,
		cs.PaddingTop, cs.PaddingRight, cs.PaddingBottom, cs.PaddingLeft,
		cs.BorderTopWidth, cs.BorderRightWidth, cs.BorderBottomWidth, cs.BorderLeftWidth))

	if box.Text != "" {
		snippet := box.Text
		if len(snippet) > 40 {
			snippet = snippet[:37] + "..."
		}
		snippet = strings.ReplaceAll(snippet, "\n", "\\n")
		sb.WriteString(fmt.Sprintf(" %q", snippet))
	}

	if box.ImageSrc != "" {
		sb.WriteString(fmt.Sprintf(" SRC:%q", box.ImageSrc))
	}

	sb.WriteString("\n")

	// Dump Inline Runs (no absolute geometry available, just text)
	if len(box.InlineRuns) > 0 {
		runIndent := indent + "  "
		for i, run := range box.InlineRuns {
			runText := strings.ReplaceAll(run.Text, "\n", "\\n")
			sb.WriteString(fmt.Sprintf("%s[RUN %d] %q\n", runIndent, i, runText))
		}
	}

	// Dump Children
	for _, child := range box.Children {
		dumpBox(sb, child, depth+1)
	}
}

func boxTypeString(t BoxType) string {
	switch t {
	case BlockBox:
		return "BLOCK"
	case InlineBox:
		return "INLINE"
	case TextBox:
		return "TEXT"
	case TableBox:
		return "TABLE"
	case TableRowBox:
		return "ROW"
	case TableCellBox:
		return "CELL"
	case ImageBox:
		return "IMAGE"
	case HRBox:
		return "HR"
	case PageBreakBox:
		return "BRD"
	default:
		return "UNKNOWN"
	}
}
