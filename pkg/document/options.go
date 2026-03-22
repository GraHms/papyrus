package document

// Options holds configuration for document generation.
type Options struct {
	// DPI controls how px units are converted to points.
	// Default: 96 (1px = 0.75pt)
	DPI float64

	// Debug enables debug mode: box outlines, baseline markers, verbose output.
	Debug bool

	// DefaultFontFamily is the font used when no font-family is specified.
	DefaultFontFamily string

	// Fonts maps font family names to file paths for additional fonts.
	Fonts map[string]string

	// PageSize overrides the page size from the document.
	// Empty string means use the document's <style> page size.
	PageSize string

	// DataFile is a path to a JSON file for template variable interpolation.
	DataFile string

	// Data is inline JSON data for template interpolation (takes precedence over DataFile).
	Data map[string]interface{}
}

// Option is a functional option for configuring document generation.
type Option func(*Options)

func defaultOptions() Options {
	return Options{
		DPI:               96,
		Debug:             false,
		DefaultFontFamily: "Liberation Sans",
		Fonts:             make(map[string]string),
	}
}

// WithDPI sets the DPI for px-to-pt conversion.
func WithDPI(dpi float64) Option {
	return func(o *Options) {
		o.DPI = dpi
	}
}

// WithDebug enables debug rendering mode.
func WithDebug() Option {
	return func(o *Options) {
		o.Debug = true
	}
}

// WithFont registers an additional font family.
func WithFont(family, path string) Option {
	return func(o *Options) {
		o.Fonts[family] = path
	}
}

// WithPageSize overrides the document page size.
func WithPageSize(size string) Option {
	return func(o *Options) {
		o.PageSize = size
	}
}

// WithData sets template interpolation data.
func WithData(data map[string]interface{}) Option {
	return func(o *Options) {
		o.Data = data
	}
}

// WithDataFile sets the path to a JSON data file for template interpolation.
func WithDataFile(path string) Option {
	return func(o *Options) {
		o.DataFile = path
	}
}
