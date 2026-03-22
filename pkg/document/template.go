package document

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"
)

// DefaultFuncMap provides a set of common template functions useful for document generation.
var DefaultFuncMap = template.FuncMap{
	"currency": func(amount interface{}) string {
		switch v := amount.(type) {
		case float64:
			return fmt.Sprintf("$%.2f", v)
		case float32:
			return fmt.Sprintf("$%.2f", v)
		case int:
			return fmt.Sprintf("$%d.00", v)
		case int64:
			return fmt.Sprintf("$%d.00", v)
		default:
			return fmt.Sprintf("$%v", v)
		}
	},
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"date": func(format string, t interface{}) string {
		switch v := t.(type) {
		case time.Time:
			return v.Format(format)
		case string:
			parsed, err := time.Parse(time.RFC3339, v)
			if err == nil {
				return parsed.Format(format)
			}
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	},
	"default": func(defaultValue, value interface{}) interface{} {
		// Try to detect empty values
		empty := false
		switch v := value.(type) {
		case string:
			empty = v == ""
		case int:
			empty = v == 0
		case float64:
			empty = v == 0
		case nil:
			empty = true
		}
		if empty {
			return defaultValue
		}
		return value
	},
}

// Template is a wrapper around text/template that outputs Papyrus documents.
type Template struct {
	tmpl *template.Template
}

// NewTemplate creates a new empty Template with the default function map.
func NewTemplate(name string) *Template {
	return &Template{
		tmpl: template.New(name).Funcs(DefaultFuncMap),
	}
}

// Parse parses the given template string into the template.
func (t *Template) Parse(text string) (*Template, error) {
	_, err := t.tmpl.Parse(text)
	if err != nil {
		return nil, fmt.Errorf("papyrus template parse error: %w", err)
	}
	return t, nil
}

// ParseFiles parses the given template files into the template.
func (t *Template) ParseFiles(filenames ...string) (*Template, error) {
	_, err := t.tmpl.ParseFiles(filenames...)
	if err != nil {
		return nil, fmt.Errorf("papyrus template parse files error: %w", err)
	}
	return t, nil
}

// Execute applies the template to the data, generating XML, and then parses it into a Document.
// If name is empty, the base template is executed.
func (t *Template) Execute(name string, data interface{}) (*Document, error) {
	var buf bytes.Buffer
	var err error

	if name != "" {
		err = t.tmpl.ExecuteTemplate(&buf, name, data)
	} else {
		err = t.tmpl.Execute(&buf, data)
	}

	if err != nil {
		return nil, fmt.Errorf("papyrus template execution error: %w", err)
	}

	return Parse(&buf)
}

// ParseTemplate reads a text template from an io.Reader and returns a compiled Template.
func ParseTemplate(r io.Reader) (*Template, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	t := NewTemplate("doc")
	return t.Parse(string(data))
}
