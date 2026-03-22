// Package render handles PDF generation for goxml2pdf.
package render

import (
	"embed"
	"fmt"
	"io"
	"strings"

	"github.com/signintech/gopdf"
)

//go:embed fonts/liberation-sans/*.ttf
var embeddedFonts embed.FS

// FontKey identifies a font variant.
type FontKey struct {
	Family string
	Bold   bool
	Italic bool
}

// FontManager manages font loading and registration with gopdf.
type FontManager struct {
	pdf      *gopdf.GoPdf
	loaded   map[FontKey]string // key → gopdf font family name
	paths    map[string]string  // family name → path (user-provided)
}

// NewFontManager creates a FontManager.
func NewFontManager(pdf *gopdf.GoPdf) *FontManager {
	return &FontManager{
		pdf:    pdf,
		loaded: make(map[FontKey]string),
		paths:  make(map[string]string),
	}
}

// RegisterFont registers a user-provided font family path.
func (fm *FontManager) RegisterFont(family, path string) {
	fm.paths[strings.ToLower(family)] = path
}

// EnsureFont loads and registers the font for a given family/bold/italic combination.
// Returns the gopdf font family name to use in SetFont.
func (fm *FontManager) EnsureFont(family string, bold, italic bool) (string, error) {
	key := FontKey{Family: strings.ToLower(family), Bold: bold, Italic: italic}

	if name, ok := fm.loaded[key]; ok {
		return name, nil
	}

	// Map family to internal name
	gopdfName := fontFamilyName(family, bold, italic)

	// Try user-provided path
	if userPath, ok := fm.paths[strings.ToLower(family)]; ok {
		return fm.loadFromPath(key, gopdfName, userPath, bold, italic)
	}

	// Try Liberation Sans (embedded)
	if isLiberationSans(family) {
		return fm.loadEmbedded(key, gopdfName, bold, italic)
	}

	// Try system fallback: Liberation Sans
	return fm.loadEmbedded(key, fontFamilyName("liberation sans", bold, italic), bold, italic)
}

// SetFont applies the font to the PDF for the given family/size/style.
func (fm *FontManager) SetFont(family string, bold, italic bool, size float64) error {
	name, err := fm.EnsureFont(family, bold, italic)
	if err != nil {
		// Fallback to Liberation Sans Regular
		name, err = fm.EnsureFont("Liberation Sans", false, false)
		if err != nil {
			return fmt.Errorf("render: failed to load fallback font: %w", err)
		}
	}
	return fm.pdf.SetFont(name, "", size)
}

func (fm *FontManager) loadFromPath(key FontKey, gopdfName, path string, bold, italic bool) (string, error) {
	style := fontStyle(bold, italic)
	err := fm.pdf.AddTTFFont(gopdfName, path)
	if err != nil {
		return "", fmt.Errorf("render: failed to load font %q from %q: %w", gopdfName, path, err)
	}
	_ = style
	fm.loaded[key] = gopdfName
	return gopdfName, nil
}

func (fm *FontManager) loadEmbedded(key FontKey, gopdfName string, bold, italic bool) (string, error) {
	filename := liberationSansFilename(bold, italic)
	data, err := embeddedFonts.ReadFile("fonts/liberation-sans/" + filename)
	if err != nil {
		return "", fmt.Errorf("render: embedded font %q not found: %w", filename, err)
	}

	err = fm.pdf.AddTTFFontByReader(gopdfName, bytesReader(data))
	if err != nil {
		return "", fmt.Errorf("render: failed to register embedded font %q: %w", gopdfName, err)
	}

	fm.loaded[key] = gopdfName
	return gopdfName, nil
}

func isLiberationSans(family string) bool {
	lower := strings.ToLower(family)
	return lower == "liberation sans" || lower == "sans-serif" || lower == "sans" ||
		lower == "helvetica" || lower == "arial" || lower == "" ||
		lower == "liberation serif" || lower == "serif" || lower == "times"
}

func liberationSansFilename(bold, italic bool) string {
	switch {
	case bold && italic:
		return "LiberationSans-BoldItalic.ttf"
	case bold:
		return "LiberationSans-Bold.ttf"
	case italic:
		return "LiberationSans-Italic.ttf"
	default:
		return "LiberationSans-Regular.ttf"
	}
}

func fontFamilyName(family string, bold, italic bool) string {
	base := strings.ReplaceAll(strings.ToLower(family), " ", "-")
	if bold && italic {
		return base + "-bolditalic"
	}
	if bold {
		return base + "-bold"
	}
	if italic {
		return base + "-italic"
	}
	return base + "-regular"
}

func fontStyle(bold, italic bool) string {
	switch {
	case bold && italic:
		return "BI"
	case bold:
		return "B"
	case italic:
		return "I"
	default:
		return ""
	}
}

// bytesReader wraps a byte slice in an io.Reader.
func bytesReader(data []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: data}
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func (b *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}
