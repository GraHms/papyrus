package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ismaelvodacom/goxml2pdf/pkg/document"
)

var (
	dataFile = flag.String("data", "", "JSON data file for template interpolation")
	watch    = flag.Bool("watch", false, "Watch input file and regenerate on change")
	debug    = flag.Bool("debug", false, "Enable debug mode (box outlines, verbose output)")
	pageSize = flag.String("page-size", "", "Override page size (A4, letter, legal, WxH)")
	dpi      = flag.Int("dpi", 96, "DPI for px unit conversion")
	output   = flag.String("o", "", "Output PDF file path")
	fontFlag = flag.String("font", "", "Register additional font (name=path), repeatable via comma")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: goxml2pdf [flags] <input.xml> [output.pdf]\n\n")
		fmt.Fprintf(os.Stderr, "Convert goxml2pdf XML documents to PDF.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := args[0]
	outputPath := *output

	// Derive output path from input if not specified
	if outputPath == "" {
		if len(args) > 1 {
			outputPath = args[1]
		} else {
			ext := filepath.Ext(inputPath)
			outputPath = strings.TrimSuffix(inputPath, ext) + ".pdf"
		}
	}

	fmt.Fprintf(os.Stderr, "goxml2pdf: %s → %s\n", inputPath, outputPath)

	if *watch {
		fmt.Fprintf(os.Stderr, "goxml2pdf: watch mode is not yet implemented\n")
		os.Exit(1)
	}

	// Build options
	var opts []document.Option

	if *debug {
		opts = append(opts, document.WithDebug())
	}
	if *dpi != 96 {
		opts = append(opts, document.WithDPI(float64(*dpi)))
	}
	if *pageSize != "" {
		opts = append(opts, document.WithPageSize(*pageSize))
	}
	if *dataFile != "" {
		opts = append(opts, document.WithDataFile(*dataFile))
	}

	// Parse font flags: "FamilyName=/path/to/font.ttf"
	if *fontFlag != "" {
		for _, entry := range strings.Split(*fontFlag, ",") {
			entry = strings.TrimSpace(entry)
			idx := strings.Index(entry, "=")
			if idx < 0 {
				fmt.Fprintf(os.Stderr, "goxml2pdf: invalid font flag %q (expected name=path)\n", entry)
				continue
			}
			family := strings.TrimSpace(entry[:idx])
			path := strings.TrimSpace(entry[idx+1:])
			opts = append(opts, document.WithFont(family, path))
		}
	}

	// Generate PDF
	if err := document.GenerateFromFile(inputPath, outputPath, opts...); err != nil {
		fmt.Fprintf(os.Stderr, "goxml2pdf: error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "goxml2pdf: wrote %s\n", outputPath)
}
