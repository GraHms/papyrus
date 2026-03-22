package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grahms/papyrus/pkg/document"
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
		fmt.Fprintf(os.Stderr, "Usage: papyrus [flags] <input.xml> [output.pdf]\n\n")
		fmt.Fprintf(os.Stderr, "Convert Papyrus XML documents to PDF.\n\n")
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

	fmt.Fprintf(os.Stderr, "papyrus: %s → %s\n", inputPath, outputPath)

	if *watch {
		fmt.Fprintf(os.Stderr, "papyrus: watch mode is not yet implemented\n")
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
				fmt.Fprintf(os.Stderr, "papyrus: invalid font flag %q (expected name=path)\n", entry)
				continue
			}
			family := strings.TrimSpace(entry[:idx])
			path := strings.TrimSpace(entry[idx+1:])
			opts = append(opts, document.WithFont(family, path))
		}
	}

	// Generate PDF
	if *dataFile != "" {
		// 1. Read JSON data
		dataBytes, err := os.ReadFile(*dataFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: error reading data file: %v\n", err)
			os.Exit(1)
		}
		var data map[string]interface{}
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: error parsing JSON: %v\n", err)
			os.Exit(1)
		}

		// 2. Open template
		f, err := os.Open(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: error opening template: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		// 3. Compile template
		tmpl, err := document.ParseTemplate(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: template error: %v\n", err)
			os.Exit(1)
		}

		// 4. Render template to document
		doc, err := tmpl.Execute("", data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: template execute error: %v\n", err)
			os.Exit(1)
		}

		// 5. Render document to PDF
		out, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()

		// Important: supply BasePath explicitly since we're bypassing GenerateFromFile
		opts = append(opts, func(o *document.Options) {
			o.BasePath = filepath.Dir(inputPath)
		})

		if err := doc.Render(out, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: error rendering PDF: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Standard static XML to PDF path
		if err := document.GenerateFromFile(inputPath, outputPath, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "papyrus: error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "papyrus: wrote %s\n", outputPath)
}
