package style

// PropertyDef defines a CSS property's metadata.
type PropertyDef struct {
	// Inherited indicates if the property is inherited by default.
	Inherited bool
	// Initial is the initial (default) value string.
	Initial string
}

// KnownProperties is the set of CSS properties supported by Papyrus.
var KnownProperties = map[string]PropertyDef{
	// Typography (inherited)
	"font-family":     {Inherited: true, Initial: "Liberation Sans"},
	"font-size":       {Inherited: true, Initial: "10pt"},
	"font-weight":     {Inherited: true, Initial: "normal"},
	"font-style":      {Inherited: true, Initial: "normal"},
	"color":           {Inherited: true, Initial: "#000000"},
	"line-height":     {Inherited: true, Initial: "1.2"},
	"text-align":      {Inherited: true, Initial: "left"},
	"text-decoration": {Inherited: false, Initial: "none"},
	"letter-spacing":  {Inherited: true, Initial: "normal"},
	"text-transform":  {Inherited: true, Initial: "none"},
	"white-space":     {Inherited: true, Initial: "normal"},
	"text-indent":     {Inherited: true, Initial: "0"},

	// Box model (not inherited)
	"width":          {Inherited: false, Initial: "auto"},
	"height":         {Inherited: false, Initial: "auto"},
	"min-width":      {Inherited: false, Initial: "0"},
	"max-width":      {Inherited: false, Initial: "none"},
	"min-height":     {Inherited: false, Initial: "0"},
	"max-height":     {Inherited: false, Initial: "none"},
	"margin":         {Inherited: false, Initial: "0"},
	"margin-top":     {Inherited: false, Initial: "0"},
	"margin-right":   {Inherited: false, Initial: "0"},
	"margin-bottom":  {Inherited: false, Initial: "0"},
	"margin-left":    {Inherited: false, Initial: "0"},
	"padding":        {Inherited: false, Initial: "0"},
	"padding-top":    {Inherited: false, Initial: "0"},
	"padding-right":  {Inherited: false, Initial: "0"},
	"padding-bottom": {Inherited: false, Initial: "0"},
	"padding-left":   {Inherited: false, Initial: "0"},

	// Borders (not inherited)
	"border":              {Inherited: false, Initial: "none"},
	"border-width":        {Inherited: false, Initial: "0"},
	"border-style":        {Inherited: false, Initial: "none"},
	"border-color":        {Inherited: false, Initial: "#000000"},
	"border-top":          {Inherited: false, Initial: "none"},
	"border-right":        {Inherited: false, Initial: "none"},
	"border-bottom":       {Inherited: false, Initial: "none"},
	"border-left":         {Inherited: false, Initial: "none"},
	"border-top-width":    {Inherited: false, Initial: "0"},
	"border-right-width":  {Inherited: false, Initial: "0"},
	"border-bottom-width": {Inherited: false, Initial: "0"},
	"border-left-width":   {Inherited: false, Initial: "0"},
	"border-top-color":    {Inherited: false, Initial: "#000000"},
	"border-right-color":  {Inherited: false, Initial: "#000000"},
	"border-bottom-color": {Inherited: false, Initial: "#000000"},
	"border-left-color":   {Inherited: false, Initial: "#000000"},
	"border-top-style":    {Inherited: false, Initial: "none"},
	"border-right-style":  {Inherited: false, Initial: "none"},
	"border-bottom-style": {Inherited: false, Initial: "none"},
	"border-left-style":   {Inherited: false, Initial: "none"},
	"border-radius":       {Inherited: false, Initial: "0"},

	// Colors and backgrounds (not inherited, except color above)
	"background-color": {Inherited: false, Initial: "transparent"},
	"background-image": {Inherited: false, Initial: "none"},
	"opacity":          {Inherited: false, Initial: "1"},

	// Layout (not inherited)
	"display":        {Inherited: false, Initial: "block"},
	"vertical-align": {Inherited: false, Initial: "top"},
	"overflow":       {Inherited: false, Initial: "visible"},

	// Table (not inherited)
	"border-collapse": {Inherited: true, Initial: "separate"},
	"border-spacing":  {Inherited: true, Initial: "0"},
	"table-layout":    {Inherited: false, Initial: "auto"},

	// Page break (not inherited)
	"page-break-before": {Inherited: false, Initial: "auto"},
	"page-break-after":  {Inherited: false, Initial: "auto"},
	"page-break-inside": {Inherited: false, Initial: "auto"},
	"orphans":           {Inherited: true, Initial: "2"},
	"widows":            {Inherited: true, Initial: "2"},

	// Page rule pseudo-properties
	"size":        {Inherited: false, Initial: "A4"},
	"orientation": {Inherited: false, Initial: "portrait"},
}

// IsKnownProperty returns true if the property is in the supported set.
func IsKnownProperty(name string) bool {
	_, ok := KnownProperties[name]
	return ok
}

// IsInherited returns true if the property is inherited.
func IsInherited(name string) bool {
	if def, ok := KnownProperties[name]; ok {
		return def.Inherited
	}
	return false
}

// InitialValue returns the initial CSS value string for a property.
func InitialValue(name string) string {
	if def, ok := KnownProperties[name]; ok {
		return def.Initial
	}
	return ""
}
