package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// Rule represents a single CSS rule: selector(s) + declarations.
type Rule struct {
	Selectors    []Selector
	Declarations []Declaration
	SourceOrder  int
}

// Selector represents a parsed CSS selector.
type Selector struct {
	// Raw is the original selector string.
	Raw string
	// Parts are the parsed selector components in order.
	Parts []SelectorPart
	// Specificity of this selector.
	Specificity Specificity
}

// SelectorPart is a single part of a compound selector.
type SelectorPart struct {
	Combinator string // "", " " (descendant), ">" (child)
	Tag        string // element tag or "*"
	ID         string
	Classes    []string
	PseudoClass string // first-child, last-child, nth-child(n)
	NthArg      string // argument for nth-child
}

// Specificity holds CSS specificity values (a=IDs, b=classes, c=elements).
type Specificity struct {
	A int // ID selectors
	B int // class selectors
	C int // element selectors
}

// Less returns true if s is less specific than other.
func (s Specificity) Less(other Specificity) bool {
	if s.A != other.A {
		return s.A < other.A
	}
	if s.B != other.B {
		return s.B < other.B
	}
	return s.C < other.C
}

// Declaration is a CSS property-value pair.
type Declaration struct {
	Property  string
	Value     string
	Important bool
}

// cssTokenType identifies token types in the CSS tokenizer.
type cssTokenType int

const (
	cssIdent      cssTokenType = iota // identifier or keyword
	cssString                         // "..." or '...'
	cssNumber                         // 123 or 1.5
	cssDimension                      // 12pt, 1.5em
	cssPercentage                     // 50%
	cssHash                           // #color
	cssAt                             // @rule
	cssDelim                          // single char delimiter
	cssColon                          // :
	cssSemicolon                      // ;
	cssComma                          // ,
	cssLBrace                         // {
	cssRBrace                         // }
	cssLParen                         // (
	cssRParen                         // )
	cssWhitespace                     // whitespace
	cssComment                        // comment (consumed)
	cssEOF
)

type cssToken struct {
	typ   cssTokenType
	value string
	line  int
}

// cssTokenizer tokenizes CSS text.
type cssTokenizer struct {
	input []rune
	pos   int
	line  int
}

func newCSSTokenizer(input string) *cssTokenizer {
	return &cssTokenizer{
		input: []rune(input),
		pos:   0,
		line:  1,
	}
}

func (t *cssTokenizer) peek() (rune, bool) {
	if t.pos >= len(t.input) {
		return 0, false
	}
	return t.input[t.pos], true
}

func (t *cssTokenizer) peekAt(offset int) (rune, bool) {
	idx := t.pos + offset
	if idx >= len(t.input) {
		return 0, false
	}
	return t.input[idx], true
}

func (t *cssTokenizer) consume() rune {
	r := t.input[t.pos]
	t.pos++
	if r == '\n' {
		t.line++
	}
	return r
}

func (t *cssTokenizer) next() cssToken {
	for {
		r, ok := t.peek()
		if !ok {
			return cssToken{typ: cssEOF}
		}

		line := t.line

		// Skip comments
		if r == '/' {
			next, ok2 := t.peekAt(1)
			if ok2 && next == '*' {
				t.consume() // /
				t.consume() // *
				for {
					c, ok3 := t.peek()
					if !ok3 {
						break
					}
					t.consume()
					if c == '*' {
						if end, ok4 := t.peek(); ok4 && end == '/' {
							t.consume()
							break
						}
					}
				}
				continue
			}
		}

		// Whitespace
		if unicode.IsSpace(r) {
			for {
				c, ok2 := t.peek()
				if !ok2 || !unicode.IsSpace(c) {
					break
				}
				t.consume()
			}
			return cssToken{typ: cssWhitespace, value: " ", line: line}
		}

		// String literals
		if r == '"' || r == '\'' {
			quote := t.consume()
			var sb strings.Builder
			for {
				c, ok2 := t.peek()
				if !ok2 || c == rune(quote) {
					if ok2 {
						t.consume()
					}
					break
				}
				if c == '\\' {
					t.consume()
					escaped, ok3 := t.peek()
					if ok3 {
						sb.WriteRune(t.consume())
						_ = escaped
					}
					continue
				}
				sb.WriteRune(t.consume())
			}
			return cssToken{typ: cssString, value: sb.String(), line: line}
		}

		// Hash color
		if r == '#' {
			t.consume()
			var sb strings.Builder
			sb.WriteRune('#')
			for {
				c, ok2 := t.peek()
				if !ok2 {
					break
				}
				if isHexOrIdent(c) {
					sb.WriteRune(t.consume())
				} else {
					break
				}
			}
			return cssToken{typ: cssHash, value: sb.String(), line: line}
		}

		// Numbers and dimensions
		if unicode.IsDigit(r) || (r == '.' && isDigitAt(t, 1)) || (r == '-' && (isDigitAt(t, 1) || (peekCharAt(t, 1) == '.' && isDigitAt(t, 2)))) {
			return t.readNumberOrDimension(line)
		}

		// Identifiers and keywords
		if isIdentStart(r) {
			return t.readIdent(line)
		}

		// Single character tokens
		t.consume()
		switch r {
		case '{':
			return cssToken{typ: cssLBrace, value: "{", line: line}
		case '}':
			return cssToken{typ: cssRBrace, value: "}", line: line}
		case '(':
			return cssToken{typ: cssLParen, value: "(", line: line}
		case ')':
			return cssToken{typ: cssRParen, value: ")", line: line}
		case ':':
			return cssToken{typ: cssColon, value: ":", line: line}
		case ';':
			return cssToken{typ: cssSemicolon, value: ";", line: line}
		case ',':
			return cssToken{typ: cssComma, value: ",", line: line}
		case '@':
			return cssToken{typ: cssAt, value: "@", line: line}
		default:
			return cssToken{typ: cssDelim, value: string(r), line: line}
		}
	}
}

func (t *cssTokenizer) readNumberOrDimension(line int) cssToken {
	var sb strings.Builder
	// optional minus
	if r, ok := t.peek(); ok && r == '-' {
		sb.WriteRune(t.consume())
	}
	// digits
	for {
		r, ok := t.peek()
		if !ok || !unicode.IsDigit(r) {
			break
		}
		sb.WriteRune(t.consume())
	}
	// decimal
	if r, ok := t.peek(); ok && r == '.' {
		sb.WriteRune(t.consume())
		for {
			r2, ok2 := t.peek()
			if !ok2 || !unicode.IsDigit(r2) {
				break
			}
			sb.WriteRune(t.consume())
		}
	}

	numStr := sb.String()

	// Check for percentage or unit suffix
	if r, ok := t.peek(); ok && r == '%' {
		t.consume()
		return cssToken{typ: cssPercentage, value: numStr + "%", line: line}
	}

	// Check for unit identifier
	if r, ok := t.peek(); ok && isIdentStart(r) {
		var unit strings.Builder
		for {
			c, ok2 := t.peek()
			if !ok2 || (!unicode.IsLetter(c) && c != '-' && !unicode.IsDigit(c)) {
				break
			}
			unit.WriteRune(t.consume())
		}
		return cssToken{typ: cssDimension, value: numStr + unit.String(), line: line}
	}

	return cssToken{typ: cssNumber, value: numStr, line: line}
}

func (t *cssTokenizer) readIdent(line int) cssToken {
	var sb strings.Builder
	for {
		r, ok := t.peek()
		if !ok || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_') {
			break
		}
		sb.WriteRune(t.consume())
	}
	return cssToken{typ: cssIdent, value: sb.String(), line: line}
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_' || r == '-'
}

func isHexOrIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
}

func isDigitAt(t *cssTokenizer, offset int) bool {
	r, ok := t.peekAt(offset)
	return ok && unicode.IsDigit(r)
}

func peekCharAt(t *cssTokenizer, offset int) rune {
	r, _ := t.peekAt(offset)
	return r
}

// ParseCSS parses CSS text into a list of Rules.
func ParseCSS(cssText string) ([]Rule, error) {
	tokenizer := newCSSTokenizer(cssText)
	var tokens []cssToken

	// Collect all non-whitespace tokens (keep whitespace for selector parsing)
	for {
		tok := tokenizer.next()
		if tok.typ == cssEOF {
			break
		}
		tokens = append(tokens, tok)
	}

	p := &cssParser{tokens: tokens, pos: 0}
	return p.parseStylesheet()
}

type cssParser struct {
	tokens []cssToken
	pos    int
	order  int
}

func (p *cssParser) peek() cssToken {
	// Skip whitespace for most contexts
	for p.pos < len(p.tokens) && p.tokens[p.pos].typ == cssWhitespace {
		p.pos++
	}
	if p.pos >= len(p.tokens) {
		return cssToken{typ: cssEOF}
	}
	return p.tokens[p.pos]
}

func (p *cssParser) peekRaw() cssToken {
	if p.pos >= len(p.tokens) {
		return cssToken{typ: cssEOF}
	}
	return p.tokens[p.pos]
}

func (p *cssParser) consume() cssToken {
	tok := p.peek() // skip whitespace
	p.pos++
	return tok
}

func (p *cssParser) parseStylesheet() ([]Rule, error) {
	var rules []Rule

	for {
		tok := p.peek()
		if tok.typ == cssEOF {
			break
		}

		// Skip @rules (page, etc.) for now — just collect declarations
		if tok.typ == cssAt {
			p.consume() // @
			atName := p.peek()
			if atName.typ == cssIdent {
				p.consume()
			}
			// read until { or ;
			for {
				t := p.peek()
				if t.typ == cssEOF || t.typ == cssLBrace || t.typ == cssSemicolon {
					break
				}
				p.consume()
			}
			if p.peek().typ == cssLBrace {
				// Parse @page rules as special rules
				if atName.typ == cssIdent && atName.value == "page" {
					p.consume() // {
					decls := p.parseDeclarations()
					// Create a special page rule
					rule := Rule{
						Selectors:    []Selector{{Raw: "@page", Parts: []SelectorPart{{Tag: "@page"}}}},
						Declarations: decls,
						SourceOrder:  p.order,
					}
					p.order++
					rules = append(rules, rule)
				} else {
					// Skip other @rules
					p.skipBlock()
				}
			} else if p.peek().typ == cssSemicolon {
				p.consume()
			}
			continue
		}

		// Parse selector(s) + rule block
		rule, err := p.parseRule()
		if err != nil {
			// Skip to next rule
			for {
				t := p.peek()
				if t.typ == cssEOF || t.typ == cssRBrace {
					if t.typ == cssRBrace {
						p.consume()
					}
					break
				}
				if t.typ == cssLBrace {
					p.skipBlock()
					break
				}
				p.consume()
			}
			continue
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (p *cssParser) parseRule() (Rule, error) {
	// Collect selector text until {
	var selectorTokens []cssToken
	for {
		tok := p.peekRaw()
		if tok.typ == cssEOF {
			return Rule{}, fmt.Errorf("css: unexpected EOF in selector")
		}
		if tok.typ == cssLBrace {
			p.pos++
			break
		}
		selectorTokens = append(selectorTokens, tok)
		p.pos++
	}

	// Parse selectors (comma-separated)
	selectorText := tokensToString(selectorTokens)
	selectors, err := parseSelectors(selectorText)
	if err != nil {
		return Rule{}, err
	}

	decls := p.parseDeclarations()

	rule := Rule{
		Selectors:    selectors,
		Declarations: decls,
		SourceOrder:  p.order,
	}
	p.order++
	return rule, nil
}

func (p *cssParser) parseDeclarations() []Declaration {
	var decls []Declaration

	for {
		tok := p.peek()
		if tok.typ == cssEOF || tok.typ == cssRBrace {
			if tok.typ == cssRBrace {
				p.consume()
			}
			break
		}

		// Parse property name
		if tok.typ != cssIdent {
			p.consume() // skip unknown
			continue
		}

		propTok := p.consume()
		propName := propTok.value

		// Expect colon
		if p.peek().typ != cssColon {
			// skip to ; or }
			for {
				t := p.peek()
				if t.typ == cssSemicolon || t.typ == cssRBrace || t.typ == cssEOF {
					if t.typ == cssSemicolon {
						p.consume()
					}
					break
				}
				p.consume()
			}
			continue
		}
		p.consume() // :

		// Collect value tokens until ; or }
		var valueParts []string
		important := false
		for {
			t := p.peek()
			if t.typ == cssSemicolon || t.typ == cssRBrace || t.typ == cssEOF {
				if t.typ == cssSemicolon {
					p.consume()
				}
				break
			}
			if t.typ == cssDelim && t.value == "!" {
				p.consume()
				next := p.peek()
				if next.typ == cssIdent && strings.ToLower(next.value) == "important" {
					p.consume()
					important = true
				}
				continue
			}
			valueParts = append(valueParts, tokenValue(p.consume()))
		}

		value := strings.TrimSpace(strings.Join(valueParts, ""))
		if value != "" {
			decls = append(decls, Declaration{
				Property:  strings.ToLower(propName),
				Value:     value,
				Important: important,
			})
		}
	}

	return decls
}

func (p *cssParser) skipBlock() {
	depth := 1
	for {
		tok := p.peek()
		if tok.typ == cssEOF {
			break
		}
		p.consume()
		if tok.typ == cssLBrace {
			depth++
		} else if tok.typ == cssRBrace {
			depth--
			if depth == 0 {
				break
			}
		}
	}
}

func tokenValue(tok cssToken) string {
	switch tok.typ {
	case cssWhitespace:
		return " "
	default:
		return tok.value
	}
}

func tokensToString(tokens []cssToken) string {
	var sb strings.Builder
	for _, t := range tokens {
		sb.WriteString(t.value)
	}
	return strings.TrimSpace(sb.String())
}

// parseSelectors parses a comma-separated list of selectors.
func parseSelectors(text string) ([]Selector, error) {
	parts := strings.Split(text, ",")
	var selectors []Selector
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		sel, err := parseSingleSelector(part)
		if err != nil {
			continue // skip invalid selectors
		}
		selectors = append(selectors, sel)
	}
	if len(selectors) == 0 {
		return nil, fmt.Errorf("css: no valid selectors in %q", text)
	}
	return selectors, nil
}

// parseSingleSelector parses a single compound CSS selector.
func parseSingleSelector(text string) (Selector, error) {
	sel := Selector{Raw: text}

	// Tokenize the selector
	tokens := tokenizeSelector(text)
	var parts []SelectorPart
	current := SelectorPart{}
	combinator := ""

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		switch tok.typ {
		case selWhitespace:
			// descendant combinator if followed by non-combinator
			if i+1 < len(tokens) && tokens[i+1].typ != selCombinator {
				if current.Tag != "" || current.ID != "" || len(current.Classes) > 0 {
					current.Combinator = combinator
					parts = append(parts, current)
					current = SelectorPart{}
					combinator = " "
				}
			}

		case selCombinator:
			if current.Tag != "" || current.ID != "" || len(current.Classes) > 0 {
				current.Combinator = combinator
				parts = append(parts, current)
				current = SelectorPart{}
			}
			combinator = tok.value

		case selElement:
			current.Tag = tok.value

		case selID:
			current.ID = strings.TrimPrefix(tok.value, "#")

		case selClass:
			current.Classes = append(current.Classes, strings.TrimPrefix(tok.value, "."))

		case selPseudo:
			pseudo := strings.TrimPrefix(tok.value, ":")
			// Handle nth-child(n)
			if strings.HasPrefix(pseudo, "nth-child(") {
				arg := strings.TrimSuffix(strings.TrimPrefix(pseudo, "nth-child("), ")")
				current.PseudoClass = "nth-child"
				current.NthArg = arg
			} else {
				current.PseudoClass = pseudo
			}
		}
	}

	// Add last part
	if current.Tag != "" || current.ID != "" || len(current.Classes) > 0 || current.PseudoClass != "" {
		current.Combinator = combinator
		parts = append(parts, current)
	}

	sel.Parts = parts
	sel.Specificity = calcSpecificity(parts)
	return sel, nil
}

// calcSpecificity calculates the CSS specificity for a list of selector parts.
func calcSpecificity(parts []SelectorPart) Specificity {
	var sp Specificity
	for _, p := range parts {
		if p.ID != "" {
			sp.A++
		}
		sp.B += len(p.Classes)
		if p.PseudoClass != "" {
			sp.B++
		}
		if p.Tag != "" && p.Tag != "*" {
			sp.C++
		}
	}
	return sp
}

// Selector tokenizer types
type selTokType int

const (
	selElement    selTokType = iota
	selClass
	selID
	selPseudo
	selCombinator
	selWhitespace
)

type selTok struct {
	typ   selTokType
	value string
}

func tokenizeSelector(text string) []selTok {
	var tokens []selTok
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		r := runes[i]

		// Whitespace
		if r == ' ' || r == '\t' || r == '\n' {
			for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\n') {
				i++
			}
			tokens = append(tokens, selTok{typ: selWhitespace, value: " "})
			continue
		}

		// Child combinator
		if r == '>' {
			i++
			// skip surrounding whitespace
			for i < len(runes) && runes[i] == ' ' {
				i++
			}
			tokens = append(tokens, selTok{typ: selCombinator, value: ">"})
			continue
		}

		// ID selector
		if r == '#' {
			i++
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '-' || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, selTok{typ: selID, value: "#" + string(runes[start:i])})
			continue
		}

		// Class selector
		if r == '.' {
			i++
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '-' || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, selTok{typ: selClass, value: "." + string(runes[start:i])})
			continue
		}

		// Pseudo-class
		if r == ':' {
			i++
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '-' || runes[i] == '(' || runes[i] == ')') {
				i++
			}
			tokens = append(tokens, selTok{typ: selPseudo, value: ":" + string(runes[start:i])})
			continue
		}

		// Universal selector
		if r == '*' {
			i++
			tokens = append(tokens, selTok{typ: selElement, value: "*"})
			continue
		}

		// Element name
		if unicode.IsLetter(r) || r == '_' || r == '-' {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '-' || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, selTok{typ: selElement, value: string(runes[start:i])})
			continue
		}

		// Unknown character - skip
		i++
	}

	return tokens
}
