package document_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grahms/papyrus/pkg/document"
)

func TestIntegrationGoldenFiles(t *testing.T) {
	matches, err := filepath.Glob("../../examples/*.xml")
	if err != nil {
		t.Fatalf("failed to glob examples: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no example XML files found")
	}

	goldenDir := filepath.Join("testdata", "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("failed to create golden directory: %v", err)
	}

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, xmlPath := range matches {
		name := filepath.Base(xmlPath)
		t.Run(name, func(t *testing.T) {
			baseName := strings.TrimSuffix(name, ".xml")
			dataPath := filepath.Join(filepath.Dir(xmlPath), baseName+"_data.json")
			if baseName == "template_demo" {
				dataPath = filepath.Join(filepath.Dir(xmlPath), "template_data.json")
			}

			var doc *document.Document
			opts := []document.Option{
				document.WithDPI(96),
			}

			if _, err := os.Stat(dataPath); err == nil {
				dataBytes, err := os.ReadFile(dataPath)
				if err != nil {
					t.Fatalf("failed to read data file: %v", err)
				}
				var data map[string]interface{}
				if err := json.Unmarshal(dataBytes, &data); err != nil {
					t.Fatalf("failed to parse JSON data: %v", err)
				}

				xmlBytes, err := os.ReadFile(xmlPath)
				if err != nil {
					t.Fatalf("failed to read XML template: %v", err)
				}

				tmpl := document.NewTemplate("test")
				if _, err := tmpl.Parse(string(xmlBytes)); err != nil {
					t.Fatalf("failed to parse template: %v", err)
				}

				// Execute template to document
				doc, err = tmpl.Execute("", data)
				if err != nil {
					t.Fatalf("failed to parse XML output: %v", err)
				}
			} else {
				// Pure XML parse
				xmlBytes, err := os.ReadFile(xmlPath)
				if err != nil {
					t.Fatalf("failed to read XML: %v", err)
				}
				doc, err = document.Parse(bytes.NewReader(xmlBytes))
				if err != nil {
					t.Fatalf("failed to parse XML: %v", err)
				}
			}

			// Add base path so images resolve
			opts = append(opts, document.WithBasePath(filepath.Dir(xmlPath)))

			dump, err := doc.LayoutTreeToString(opts...)
			if err != nil {
				t.Fatalf("LayoutTreeToString failed: %v", err)
			}

			goldenPath := filepath.Join(goldenDir, baseName+".golden")
			if updateGolden {
				if err := os.WriteFile(goldenPath, []byte(dump), 0644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
				t.Logf("Updated golden file %s", goldenPath)
				return
			}

			expectedBytes, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v. Run with UPDATE_GOLDEN=1 to create.", goldenPath, err)
			}

			expected := string(expectedBytes)
			if dump != expected {
				t.Errorf("Layout mismatch for %s.\nRun UPDATE_GOLDEN=1 go test ./pkg/document/... to update.", name)
			}
		})
	}
}
