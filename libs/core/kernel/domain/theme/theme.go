package theme

// PluginTheme is a theme an enabled plugin contributes.
type PluginTheme struct {
	PluginID   string
	ID         string
	Name       string
	Appearance string
}

// File is a theme as a store holds it: its title, its stylesheet, and whether it pins its color scheme.
type File struct {
	Title    string
	Text     string
	IsPinned bool
}

// Origin is where a theme comes from. A theme carries its origin in its name, and
// two themes on different origins can share a title.
type Origin string

const (
	// Preset ships inside the application.
	Preset Origin = "preset"

	// Mine is a `.css` file in the user's themes folder.
	Mine Origin = "mine"

	// Plugin is a theme an enabled plugin contributes.
	Plugin Origin = "plugin"
)

// Default is this product's own palette, and the theme applied when a name
// matches nothing. It is the theme of an installation with no selection.
const Default = "preset:numen"

// Theme is one theme as the list refers to it.
type Theme struct {
	Name string

	Title string

	Origin Origin

	IsPinned bool

	PluginID string

	Appearance string
}

// ColorScheme is which half of every color pair the tokens are read as.
type ColorScheme int

const (
	// System is both halves, and the machine decides between them.
	System ColorScheme = iota
	Light
	Dark
)

// Appearance is the window's appearance: the theme, the color scheme its
// tokens are read as, and the interface and text scales.
type Appearance struct {
	ThemeName string
	Mode      ColorScheme

	InterfaceScale, TextScale float64
}
