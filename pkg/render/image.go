package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/signintech/gopdf"
)

// drawImage renders an image onto the PDF at the specified position.
func drawImage(pdf *gopdf.GoPdf, src string, x, y, w, h float64, basePath string) error {
	if src == "" {
		return nil
	}

	// Resolve path
	imgPath := src
	if !filepath.IsAbs(imgPath) && basePath != "" {
		imgPath = filepath.Join(basePath, src)
	}

	// Check if file exists
	if _, err := os.Stat(imgPath); err != nil {
		// Try relative to current directory
		if _, err2 := os.Stat(src); err2 != nil {
			return fmt.Errorf("render: image not found: %q", src)
		}
		imgPath = src
	}

	rect := &gopdf.Rect{W: w, H: h}
	if err := pdf.Image(imgPath, x, y, rect); err != nil {
		return fmt.Errorf("render: failed to embed image %q: %w", src, err)
	}
	return nil
}
