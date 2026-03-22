package style

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ismaelvodacom/goxml2pdf/pkg/parser"
)

// PageStyle holds resolved page-level CSS from @page rules.
type PageStyle struct {
	Width   float64 // pt
	Height  float64 // pt
	MarginTop, MarginRight, MarginBottom, MarginLeft float64
}

// ResolverContext holds shared state for style resolution.
type ResolverContext struct {
	Rules         []parser.Rule
	DPI           float64
	RootFontSize  float64 // pt
	PageStyle     PageStyle
}

// NewResolver creates a new resolver context.
func NewResolver(rules []parser.Rule, dpi float64) *ResolverContext {
	r := &ResolverContext{
		Rules:        rules,
		DPI:          dpi,
		RootFontSize: 10, // default root font size
	}
	r.resolvePageStyle()
	return r
}

// resolvePageStyle processes @page rules to compute page dimensions.
func (r *ResolverContext) resolvePageStyle() {
	// Set defaults: A4
	r.PageStyle = PageStyle{
		Width:        595.28, // A4 in pt
		Height:       841.89,
		MarginTop:    56.69, // 20mm
		MarginRight:  56.69,
		MarginBottom: 56.69,
		MarginLeft:   56.69,
	}

	for _, rule := range r.Rules {
		if len(rule.Selectors) == 1 && rule.Selectors[0].Raw == "@page" {
			r.applyPageDeclarations(rule.Declarations)
		}
		// Also check for "page" selector (SPEC uses bare "page" not "@page")
		for _, sel := range rule.Selectors {
			if sel.Raw == "page" {
				r.applyPageDeclarations(rule.Declarations)
			}
		}
	}
}

func (r *ResolverContext) applyPageDeclarations(decls []parser.Declaration) {
	for _, d := range decls {
		switch d.Property {
		case "size":
			w, h := parsePageSize(d.Value)
			if w > 0 {
				r.PageStyle.Width = w
				r.PageStyle.Height = h
			}
		case "margin":
			sides := parseMarginShorthand(d.Value, 72, r.DPI)
			r.PageStyle.MarginTop = sides[0]
			r.PageStyle.MarginRight = sides[1]
			r.PageStyle.MarginBottom = sides[2]
			r.PageStyle.MarginLeft = sides[3]
		case "margin-top":
			r.PageStyle.MarginTop = parseLengthToPt(d.Value, 0, 10, 10, r.DPI)
		case "margin-right":
			r.PageStyle.MarginRight = parseLengthToPt(d.Value, 0, 10, 10, r.DPI)
		case "margin-bottom":
			r.PageStyle.MarginBottom = parseLengthToPt(d.Value, 0, 10, 10, r.DPI)
		case "margin-left":
			r.PageStyle.MarginLeft = parseLengthToPt(d.Value, 0, 10, 10, r.DPI)
		}
	}
}

// parsePageSize parses a CSS `size` value and returns width, height in points.
func parsePageSize(value string) (float64, float64) {
	value = strings.TrimSpace(value)
	switch strings.ToUpper(value) {
	case "A4", "A4 PORTRAIT":
		return 595.28, 841.89
	case "A4 LANDSCAPE":
		return 841.89, 595.28
	case "LETTER", "LETTER PORTRAIT":
		return 612, 792
	case "LETTER LANDSCAPE":
		return 792, 612
	case "LEGAL":
		return 612, 1008
	case "A3":
		return 841.89, 1190.55
	case "A5":
		return 419.53, 595.28
	}
	// Try parsing "WxH" or "W H" format
	parts := strings.Fields(value)
	if len(parts) == 2 {
		w := MustParseLength(parts[0]).ToPoints(0, 10, 10, 96)
		h := MustParseLength(parts[1]).ToPoints(0, 10, 10, 96)
		if w > 0 && h > 0 {
			return w, h
		}
	}
	return 0, 0
}

// Resolve computes the ComputedStyle for a node given its parent's style.
// This implements the CSS cascade: matched rules + inheritance.
func (r *ResolverContext) Resolve(node *parser.Node, parent *ComputedStyle) ComputedStyle {
	if node.Type == parser.TextNode {
		// Text nodes inherit everything from parent
		if parent != nil {
			return *parent
		}
		return DefaultStyle(r.DPI)
	}

	// Start with parent for inheritance, otherwise defaults
	cs := DefaultStyle(r.DPI)
	if parent != nil {
		cs = inheritFrom(cs, parent)
	}

	// Apply element-specific defaults
	cs = applyElementDefaults(node.Tag, cs)

	// Collect all matching rules, sorted by specificity then source order
	type matchedRule struct {
		rule        parser.Rule
		selector    parser.Selector
		sourceOrder int
	}
	var matched []matchedRule

	for _, rule := range r.Rules {
		for _, sel := range rule.Selectors {
			if sel.Raw == "@page" || sel.Raw == "page" {
				continue
			}
			if MatchSelector(sel, node) {
				matched = append(matched, matchedRule{rule, sel, rule.SourceOrder})
				break
			}
		}
	}

	// Sort: lower specificity first, then by source order (later wins)
	sort.SliceStable(matched, func(i, j int) bool {
		si := matched[i].selector.Specificity
		sj := matched[j].selector.Specificity
		if si.A != sj.A {
			return si.A < sj.A
		}
		if si.B != sj.B {
			return si.B < sj.B
		}
		if si.C != sj.C {
			return si.C < sj.C
		}
		return matched[i].sourceOrder < matched[j].sourceOrder
	})

	// Apply matched declarations in order (last wins, unless !important)
	for _, m := range matched {
		for _, decl := range m.rule.Declarations {
			cs = r.applyDeclaration(cs, decl, node)
		}
	}

	// Apply inline style (highest specificity)
	if inlineStyle, ok := node.GetAttribute("style"); ok && inlineStyle != "" {
		inlineDecls, err := parseInlineStyle(inlineStyle)
		if err == nil {
			for _, decl := range inlineDecls {
				cs = r.applyDeclaration(cs, decl, node)
			}
		}
	}

	// Resolve line-height relative to font size
	cs.LineHeight = resolveLineHeight(cs, node)

	return cs
}

// inheritFrom copies inherited properties from parent into child defaults.
func inheritFrom(child ComputedStyle, parent *ComputedStyle) ComputedStyle {
	if parent == nil {
		return child
	}
	child.FontFamily = parent.FontFamily
	child.FontSize = parent.FontSize
	child.FontWeight = parent.FontWeight
	child.FontStyle = parent.FontStyle
	child.Color = parent.Color
	child.TextAlign = parent.TextAlign
	child.LetterSpacing = parent.LetterSpacing
	child.TextTransform = parent.TextTransform
	child.WhiteSpace = parent.WhiteSpace
	child.TextIndent = parent.TextIndent
	child.BorderCollapse = parent.BorderCollapse
	child.BorderSpacing = parent.BorderSpacing
	child.Orphans = parent.Orphans
	child.Widows = parent.Widows
	// LineHeightRatio is inherited so children can re-resolve against their own font-size.
	// When a ratio is set, we clear the absolute LineHeight so it gets re-resolved.
	child.LineHeightRatio = parent.LineHeightRatio
	if parent.LineHeightRatio > 0 {
		child.LineHeight = 0
	} else {
		child.LineHeight = parent.LineHeight
	}
	return child
}

// applyElementDefaults applies browser-like default styles for specific HTML elements.
func applyElementDefaults(tag string, cs ComputedStyle) ComputedStyle {
	switch tag {
	case "h1":
		cs.FontSize = 24
		cs.FontWeight = "bold"
		cs.MarginTop = 14
		cs.MarginBottom = 14
		cs.LineHeight = 28.8
	case "h2":
		cs.FontSize = 18
		cs.FontWeight = "bold"
		cs.MarginTop = 12
		cs.MarginBottom = 12
		cs.LineHeight = 21.6
	case "h3":
		cs.FontSize = 14
		cs.FontWeight = "bold"
		cs.MarginTop = 10
		cs.MarginBottom = 10
		cs.LineHeight = 16.8
	case "h4":
		cs.FontSize = 12
		cs.FontWeight = "bold"
		cs.MarginTop = 8
		cs.MarginBottom = 8
	case "h5":
		cs.FontSize = 10
		cs.FontWeight = "bold"
		cs.MarginTop = 6
		cs.MarginBottom = 6
	case "h6":
		cs.FontSize = 9
		cs.FontWeight = "bold"
		cs.MarginTop = 5
		cs.MarginBottom = 5
	case "p":
		cs.MarginTop = 0
		cs.MarginBottom = 8
	case "strong", "b":
		cs.FontWeight = "bold"
		cs.Display = "inline"
	case "em", "i":
		cs.FontStyle = "italic"
		cs.Display = "inline"
	case "u":
		cs.TextDecoration = "underline"
		cs.Display = "inline"
	case "code":
		cs.FontFamily = "Courier"
		cs.Display = "inline"
	case "span":
		cs.Display = "inline"
	case "a":
		cs.Display = "inline"
		cs.Color = Color{R: 0, G: 0, B: 238, A: 255}
		cs.TextDecoration = "underline"
	case "br":
		cs.Display = "inline"
	case "blockquote":
		cs.MarginLeft = 28
		cs.MarginRight = 28
		cs.MarginTop = 8
		cs.MarginBottom = 8
	case "th":
		cs.FontWeight = "bold"
		cs.TextAlign = "left"
		cs.Display = "table-cell"
	case "td":
		cs.Display = "table-cell"
		cs.VerticalAlign = "top"
	case "tr":
		cs.Display = "table-row"
	case "thead", "tbody", "tfoot":
		cs.Display = "table-row-group"
	case "table":
		cs.Display = "table"
	case "hr":
		cs.MarginTop = 8
		cs.MarginBottom = 8
		cs.BorderTopWidth = 1
		cs.BorderTopStyle = "solid"
		cs.BorderTopColor = Color{R: 204, G: 204, B: 204, A: 255}
	case "ul", "ol":
		cs.MarginTop = 8
		cs.MarginBottom = 8
		cs.PaddingLeft = 20
	case "li":
		cs.MarginBottom = 4

	// Semantic block containers — same as div but with small vertical margins
	case "main", "article", "section":
		cs.MarginTop = 0
		cs.MarginBottom = 8
	case "aside", "nav":
		// no extra margins by default

	// Preformatted block
	case "pre":
		cs.FontFamily = "Courier"
		cs.WhiteSpace = "pre"
		cs.MarginTop = 8
		cs.MarginBottom = 8

	// Figure / caption
	case "figure":
		cs.MarginTop = 8
		cs.MarginBottom = 8
		cs.MarginLeft = 28
		cs.MarginRight = 28
	case "figcaption", "caption":
		cs.TextAlign = "center"
		cs.FontSize = cs.FontSize * 0.9
		cs.MarginTop = 4
		cs.MarginBottom = 4

	// New inline elements
	case "s":
		cs.TextDecoration = "line-through"
		cs.Display = "inline"
	case "mark":
		cs.BackgroundColor = Color{R: 255, G: 255, B: 0, A: 255}
		cs.Display = "inline"
	case "small":
		cs.FontSize = cs.FontSize * 0.85
		cs.Display = "inline"
	case "sub":
		cs.FontSize = cs.FontSize * 0.75
		cs.BaselineShift = -cs.FontSize * 0.35 // shift down
		cs.Display = "inline"
	case "sup":
		cs.FontSize = cs.FontSize * 0.75
		cs.BaselineShift = cs.FontSize * 0.5 // shift up
		cs.Display = "inline"
	case "cite":
		cs.FontStyle = "italic"
		cs.Display = "inline"
	case "q":
		cs.Display = "inline"

	// Definition lists
	case "dl":
		cs.MarginTop = 8
		cs.MarginBottom = 8
	case "dt":
		cs.FontWeight = "bold"
		cs.MarginTop = 4
	case "dd":
		cs.MarginLeft = 28
		cs.MarginBottom = 4

	// Page number/count inline markers
	case "page-number", "page-count":
		cs.Display = "inline"
	}
	return cs
}

// applyDeclaration applies a single CSS declaration to a ComputedStyle.
func (r *ResolverContext) applyDeclaration(cs ComputedStyle, d parser.Declaration, node *parser.Node) ComputedStyle {
	parentPt := 0.0 // will be updated by caller if needed
	fontPt := cs.FontSize
	rootPt := r.RootFontSize
	dpi := r.DPI

	lp := func(s string) float64 {
		return parseLengthToPt(s, parentPt, fontPt, rootPt, dpi)
	}

	switch d.Property {
	case "font-family":
		cs.FontFamily = cleanFontFamily(d.Value)
	case "font-size":
		size := parseFontSize(d.Value, cs.FontSize, r.RootFontSize)
		if size > 0 {
			cs.FontSize = size
		}
	case "font-weight":
		cs.FontWeight = d.Value
	case "font-style":
		cs.FontStyle = d.Value
	case "color":
		if c, err := ParseColor(d.Value); err == nil {
			cs.Color = c
		}
	case "background-color":
		if c, err := ParseColor(d.Value); err == nil {
			cs.BackgroundColor = c
		}
	case "text-align":
		cs.TextAlign = d.Value
	case "text-decoration":
		cs.TextDecoration = d.Value
	case "text-transform":
		cs.TextTransform = d.Value
	case "white-space":
		cs.WhiteSpace = d.Value
	case "text-indent":
		cs.TextIndent = lp(d.Value)
	case "letter-spacing":
		if d.Value != "normal" {
			cs.LetterSpacing = lp(d.Value)
		}
	case "line-height":
		if d.Value == "normal" {
			// Store as ratio so children inherit correctly
			cs.LineHeightRatio = 1.2
			cs.LineHeight = 0
		} else if v, err := strconv.ParseFloat(d.Value, 64); err == nil {
			// Unitless number (e.g. 1.5) — store ratio; resolve after inheritance
			cs.LineHeightRatio = v
			cs.LineHeight = 0
		} else {
			// Absolute length — store as absolute pt, clear ratio
			l, err := ParseLength(d.Value)
			if err == nil {
				cs.LineHeightRatio = 0
				cs.LineHeight = l.ToPoints(0, cs.FontSize, r.RootFontSize, r.DPI)
			}
		}
	case "display":
		cs.Display = d.Value
	case "vertical-align":
		cs.VerticalAlign = d.Value
	case "overflow":
		cs.Overflow = d.Value

	// Margin shorthand / longhand
	case "margin":
		sides := parseMarginShorthand(d.Value, parentPt, dpi)
		cs.MarginTop = sides[0]
		cs.MarginRight = sides[1]
		cs.MarginBottom = sides[2]
		cs.MarginLeft = sides[3]
	case "margin-top":
		cs.MarginTop = lp(d.Value)
	case "margin-right":
		cs.MarginRight = lp(d.Value)
	case "margin-bottom":
		cs.MarginBottom = lp(d.Value)
	case "margin-left":
		cs.MarginLeft = lp(d.Value)

	// Padding shorthand / longhand
	case "padding":
		sides := parseMarginShorthand(d.Value, parentPt, dpi)
		cs.PaddingTop = sides[0]
		cs.PaddingRight = sides[1]
		cs.PaddingBottom = sides[2]
		cs.PaddingLeft = sides[3]
	case "padding-top":
		cs.PaddingTop = lp(d.Value)
	case "padding-right":
		cs.PaddingRight = lp(d.Value)
	case "padding-bottom":
		cs.PaddingBottom = lp(d.Value)
	case "padding-left":
		cs.PaddingLeft = lp(d.Value)

	// Width / height
	case "width":
		if l, err := ParseLength(d.Value); err == nil {
			cs.Width = l
		}
	case "height":
		if l, err := ParseLength(d.Value); err == nil {
			cs.Height = l
		}
	case "min-width":
		cs.MinWidth = lp(d.Value)
	case "max-width":
		if strings.ToLower(d.Value) == "none" {
			cs.MaxWidth = -1
		} else {
			cs.MaxWidth = lp(d.Value)
		}
	case "min-height":
		cs.MinHeight = lp(d.Value)
	case "max-height":
		if strings.ToLower(d.Value) == "none" {
			cs.MaxHeight = -1
		} else {
			cs.MaxHeight = lp(d.Value)
		}

	// Border shorthand
	case "border":
		bw, bs, bc := parseBorderShorthand(d.Value, dpi)
		cs.BorderTopWidth = bw
		cs.BorderRightWidth = bw
		cs.BorderBottomWidth = bw
		cs.BorderLeftWidth = bw
		cs.BorderTopStyle = bs
		cs.BorderRightStyle = bs
		cs.BorderBottomStyle = bs
		cs.BorderLeftStyle = bs
		if !bc.IsZero() || bs != "none" {
			cs.BorderTopColor = bc
			cs.BorderRightColor = bc
			cs.BorderBottomColor = bc
			cs.BorderLeftColor = bc
		}
	case "border-top":
		bw, bs, bc := parseBorderShorthand(d.Value, dpi)
		cs.BorderTopWidth = bw
		cs.BorderTopStyle = bs
		if !bc.IsZero() || bs != "none" {
			cs.BorderTopColor = bc
		}
	case "border-right":
		bw, bs, bc := parseBorderShorthand(d.Value, dpi)
		cs.BorderRightWidth = bw
		cs.BorderRightStyle = bs
		if !bc.IsZero() || bs != "none" {
			cs.BorderRightColor = bc
		}
	case "border-bottom":
		bw, bs, bc := parseBorderShorthand(d.Value, dpi)
		cs.BorderBottomWidth = bw
		cs.BorderBottomStyle = bs
		if !bc.IsZero() || bs != "none" {
			cs.BorderBottomColor = bc
		}
	case "border-left":
		bw, bs, bc := parseBorderShorthand(d.Value, dpi)
		cs.BorderLeftWidth = bw
		cs.BorderLeftStyle = bs
		if !bc.IsZero() || bs != "none" {
			cs.BorderLeftColor = bc
		}
	case "border-width":
		sides := parseMarginShorthand(d.Value, parentPt, dpi)
		cs.BorderTopWidth = sides[0]
		cs.BorderRightWidth = sides[1]
		cs.BorderBottomWidth = sides[2]
		cs.BorderLeftWidth = sides[3]
	case "border-style":
		parts := strings.Fields(d.Value)
		styles := expandSides(parts, "none")
		cs.BorderTopStyle = styles[0]
		cs.BorderRightStyle = styles[1]
		cs.BorderBottomStyle = styles[2]
		cs.BorderLeftStyle = styles[3]
	case "border-color":
		parts := strings.Fields(d.Value)
		colors := expandSides(parts, "#000000")
		if c, err := ParseColor(colors[0]); err == nil {
			cs.BorderTopColor = c
		}
		if c, err := ParseColor(colors[1]); err == nil {
			cs.BorderRightColor = c
		}
		if c, err := ParseColor(colors[2]); err == nil {
			cs.BorderBottomColor = c
		}
		if c, err := ParseColor(colors[3]); err == nil {
			cs.BorderLeftColor = c
		}
	case "border-top-width":
		cs.BorderTopWidth = lp(d.Value)
	case "border-right-width":
		cs.BorderRightWidth = lp(d.Value)
	case "border-bottom-width":
		cs.BorderBottomWidth = lp(d.Value)
	case "border-left-width":
		cs.BorderLeftWidth = lp(d.Value)
	case "border-top-style":
		cs.BorderTopStyle = d.Value
	case "border-right-style":
		cs.BorderRightStyle = d.Value
	case "border-bottom-style":
		cs.BorderBottomStyle = d.Value
	case "border-left-style":
		cs.BorderLeftStyle = d.Value
	case "border-top-color":
		if c, err := ParseColor(d.Value); err == nil {
			cs.BorderTopColor = c
		}
	case "border-right-color":
		if c, err := ParseColor(d.Value); err == nil {
			cs.BorderRightColor = c
		}
	case "border-bottom-color":
		if c, err := ParseColor(d.Value); err == nil {
			cs.BorderBottomColor = c
		}
	case "border-left-color":
		if c, err := ParseColor(d.Value); err == nil {
			cs.BorderLeftColor = c
		}

	// Table
	case "border-collapse":
		cs.BorderCollapse = d.Value
	case "border-spacing":
		cs.BorderSpacing = lp(d.Value)
	case "table-layout":
		cs.TableLayout = d.Value

	// Page break
	case "page-break-before":
		cs.PageBreakBefore = d.Value
	case "page-break-after":
		cs.PageBreakAfter = d.Value
	case "page-break-inside":
		cs.PageBreakInside = d.Value
	case "orphans":
		if v, err := strconv.Atoi(d.Value); err == nil {
			cs.Orphans = v
		}
	case "widows":
		if v, err := strconv.Atoi(d.Value); err == nil {
			cs.Widows = v
		}
	}

	return cs
}

// resolveLineHeight ensures line-height is absolute (in pt).
// If LineHeightRatio > 0, it takes precedence and is multiplied by FontSize.
func resolveLineHeight(cs ComputedStyle, node *parser.Node) float64 {
	if cs.LineHeightRatio > 0 {
		return cs.FontSize * cs.LineHeightRatio
	}
	if cs.LineHeight <= 0 {
		return cs.FontSize * 1.2
	}
	return cs.LineHeight
}

// parseFontSize parses font-size values including relative keywords.
func parseFontSize(value string, parentSize, rootSize float64) float64 {
	switch strings.ToLower(value) {
	case "xx-small":
		return 6
	case "x-small":
		return 7.5
	case "small":
		return 8.5
	case "medium":
		return 10
	case "large":
		return 12
	case "x-large":
		return 14
	case "xx-large":
		return 18
	case "smaller":
		return parentSize * 0.85
	case "larger":
		return parentSize * 1.15
	}
	l, err := ParseLength(value)
	if err != nil {
		return 0
	}
	switch l.Unit {
	case UnitEm:
		return parentSize * l.Value
	case UnitRem:
		return rootSize * l.Value
	case UnitPct:
		return parentSize * l.Value / 100
	default:
		return l.ToPoints(0, parentSize, rootSize, 96)
	}
}

// parseLengthToPt is a helper that converts a CSS value string to points.
func parseLengthToPt(s string, parentPt, fontPt, rootPt, dpi float64) float64 {
	l, err := ParseLength(s)
	if err != nil {
		return 0
	}
	return l.ToPoints(parentPt, fontPt, rootPt, dpi)
}

// parseMarginShorthand parses CSS shorthand margin/padding values.
// Returns [top, right, bottom, left] in points.
func parseMarginShorthand(value string, parentPt, dpi float64) [4]float64 {
	parts := strings.Fields(value)
	var lengths [4]float64

	parse := func(s string) float64 {
		return parseLengthToPt(s, parentPt, 10, 10, dpi)
	}

	switch len(parts) {
	case 1:
		v := parse(parts[0])
		lengths = [4]float64{v, v, v, v}
	case 2:
		v0 := parse(parts[0])
		v1 := parse(parts[1])
		lengths = [4]float64{v0, v1, v0, v1}
	case 3:
		v0 := parse(parts[0])
		v1 := parse(parts[1])
		v2 := parse(parts[2])
		lengths = [4]float64{v0, v1, v2, v1}
	case 4:
		lengths = [4]float64{parse(parts[0]), parse(parts[1]), parse(parts[2]), parse(parts[3])}
	}

	return lengths
}

// parseBorderShorthand parses "width style color" border shorthand.
// Returns (width in pt, style string, color).
func parseBorderShorthand(value string, dpi float64) (float64, string, Color) {
	if strings.ToLower(value) == "none" {
		return 0, "none", Color{}
	}

	parts := strings.Fields(value)
	var width float64
	style := "solid"
	color := Color{R: 0, G: 0, B: 0, A: 255}

	for _, p := range parts {
		// Try style keyword
		switch strings.ToLower(p) {
		case "none":
			return 0, "none", Color{}
		case "solid", "dashed", "dotted", "double":
			style = strings.ToLower(p)
			continue
		}
		// Try color
		if c, err := ParseColor(p); err == nil {
			color = c
			continue
		}
		// Try length
		if l, err := ParseLength(p); err == nil {
			width = l.ToPoints(0, 10, 10, dpi)
		}
	}

	return width, style, color
}

// expandSides expands a 1-4 element slice to [top, right, bottom, left].
func expandSides(parts []string, def string) [4]string {
	switch len(parts) {
	case 0:
		return [4]string{def, def, def, def}
	case 1:
		return [4]string{parts[0], parts[0], parts[0], parts[0]}
	case 2:
		return [4]string{parts[0], parts[1], parts[0], parts[1]}
	case 3:
		return [4]string{parts[0], parts[1], parts[2], parts[1]}
	default:
		return [4]string{parts[0], parts[1], parts[2], parts[3]}
	}
}

// cleanFontFamily strips quotes from font family names.
func cleanFontFamily(value string) string {
	value = strings.TrimSpace(value)
	// Take first family in comma-separated list
	if idx := strings.Index(value, ","); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	value = strings.Trim(value, `"'`)
	return value
}

// parseInlineStyle parses a CSS inline style attribute value.
func parseInlineStyle(s string) ([]parser.Declaration, error) {
	// Wrap in a fake rule to reuse the CSS parser
	fakeCSS := "x{" + s + "}"
	rules, err := parser.ParseCSS(fakeCSS)
	if err != nil || len(rules) == 0 {
		return nil, err
	}
	return rules[0].Declarations, nil
}

// ResolveTree walks the entire DOM tree and computes styles for every node.
// Returns a map from node pointer to ComputedStyle.
func (r *ResolverContext) ResolveTree(root *parser.Node) map[*parser.Node]ComputedStyle {
	styles := make(map[*parser.Node]ComputedStyle)
	r.resolveNode(root, nil, styles)
	return styles
}

func (r *ResolverContext) resolveNode(node *parser.Node, parent *ComputedStyle, styles map[*parser.Node]ComputedStyle) {
	if node == nil {
		return
	}

	cs := r.Resolve(node, parent)
	styles[node] = cs

	for _, child := range node.Children {
		r.resolveNode(child, &cs, styles)
	}
}
