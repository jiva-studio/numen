// Package theme lists and applies the stylesheets that restyle the window.
//
// A theme is one CSS file redeclaring the tokens the interface renders with.
// The default theme ships inside the binary, enabled plugins contribute
// themes through their manifests, and the rest are files in the user's themes
// folder; all form one list with one format, and one name applies any of them.
// Nothing here reads inside a theme beyond whether a user's theme pins
// `color-scheme`.
package theme

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugintheme/themecontract"
)

// ErrNoTheme is a theme name no theme file answers for.
var ErrNoTheme = errors.New("no such theme")

//go:embed presets/*.css
var shipped embed.FS

const shippedIn = "presets"

// Extension is what a file is called to be a theme.
const Extension = ".css"

// MaxSize is the largest file read as a theme. Its text is spliced into the
// page before the window draws, and the presets are a few kilobytes each.
const MaxSize = 256 << 10

// Default is this product's own palette, and the theme applied when a name
// matches nothing. It is the theme of an installation with no selection.
const Default = "preset:numen"

// pluginPrefix starts the name of a plugin theme: `plugin:<pluginId>/<themeId>`.
const pluginPrefix = string(Plugin) + ":"

// Origin is where a theme comes from. A theme carries its origin in its name, and
// two themes on different origins can share a filename.
type Origin string

const (
	// Preset ships inside the application.
	Preset Origin = "preset"

	// Mine is a `.css` file in the user's themes folder.
	Mine Origin = "mine"

	// Plugin is a theme an enabled plugin contributes.
	Plugin Origin = "plugin"
)

// PluginName is the name of the theme a plugin contributes.
func PluginName(pluginID, themeID string) string {
	return pluginPrefix + pluginID + "/" + themeID
}

// Theme is one theme as the list refers to it.
type Theme struct {
	Name string

	Title string

	Origin Origin

	IsPinned bool

	PluginID string

	Appearance string
}

// Catalog is every theme this installation offers: the one inside the
// binary, the ones enabled plugins contribute, and the ones in the user's folder.
// A folder that could not be created or read still leaves the default theme,
// so the window opens with a theme regardless of disk state.
type Catalog struct {
	dir     string
	plugins themecontract.PluginThemes
	hub     *watchhub.Hub
}

// Open returns the catalog at the folder where this desktop keeps its
// configuration. The themes folder is created empty if it does not exist.
func Open(hub *watchhub.Hub) (Catalog, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Catalog{}, err
	}
	return OpenAt(filepath.Join(dir, "numen", "themes"), hub)
}

// OpenAt is Open with an explicit path. Watch places its watch through the hub.
func OpenAt(dir string, hub *watchhub.Hub) (Catalog, error) {
	made := os.MkdirAll(dir, 0o755)
	// The folder is where the links lead. The paths the operating system
	// reports changes at are resolved, and they are named against this.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return Catalog{dir: dir, hub: hub}, made
}

// WithPlugins returns the catalog that also offers the themes of the plugins
// the source lists. A nil source offers none.
func (c Catalog) WithPlugins(source themecontract.PluginThemes) Catalog {
	c.plugins = source
	return c
}

// Dir is the folder the user's themes are read from.
func (c Catalog) Dir() string { return c.dir }

// Themes is every theme: the shipped one first, then the plugin themes in the
// order of their plugin ids, then the user's themes ordered by name. A file that cannot be read is excluded: every name in this list is a
// theme that can be applied.
func (c Catalog) Themes() []Theme {
	var themes []Theme
	entries, err := fs.ReadDir(shipped, shippedIn)
	if err == nil {
		for _, entry := range entries {
			text, err := shipped.ReadFile(path.Join(shippedIn, entry.Name()))
			if err != nil {
				continue
			}
			themes = append(themes, describeTheme(Preset, entry.Name(), string(text)))
		}
	}
	themes = append(themes, c.getPluginThemes()...)
	return append(themes, c.getOwnThemes()...)
}

func (c Catalog) getPluginThemes() []Theme {
	if c.plugins == nil {
		return nil
	}
	declared := c.plugins.ListThemes()
	themes := make([]Theme, 0, len(declared))
	for _, one := range declared {
		themes = append(themes, Theme{
			Name:       PluginName(one.PluginID, one.ID),
			Title:      one.Name,
			Origin:     Plugin,
			IsPinned:   one.Appearance != "",
			PluginID:   one.PluginID,
			Appearance: one.Appearance,
		})
	}
	return themes
}

// ListPluginThemeNames returns the names of the plugin themes, in the order
// the plugins list them.
func (c Catalog) ListPluginThemeNames() []string {
	themes := c.getPluginThemes()
	names := make([]string, 0, len(themes))
	for _, one := range themes {
		names = append(names, one.Name)
	}
	return names
}

// HasPlugins reports whether the catalog reads plugin themes from a source.
func (c Catalog) HasPlugins() bool { return c.plugins != nil }

func (c Catalog) getOwnThemes() []Theme {
	if c.dir == "" {
		return nil
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return nil
	}
	var themes []Theme
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), Extension) {
			continue
		}
		if !isPlainName(strings.TrimSuffix(entry.Name(), Extension)) {
			continue
		}
		text, err := read(filepath.Join(c.dir, entry.Name()))
		if err != nil {
			continue
		}
		themes = append(themes, describeTheme(Mine, entry.Name(), text))
	}
	return themes
}

func describeTheme(origin Origin, filename, text string) Theme {
	title := strings.TrimSuffix(filename, Extension)
	return Theme{
		Name:     string(origin) + ":" + title,
		Title:    title,
		Origin:   origin,
		IsPinned: declaresColorScheme(text),
	}
}

// Text returns the stylesheet a name selects, or an error wrapping ErrNoTheme
// where no theme answers for the name.
func (c Catalog) Text(name string) (string, error) {
	if strings.HasPrefix(name, pluginPrefix) {
		return c.readPluginTheme(name)
	}
	origin, title, err := splitName(name)
	if err != nil {
		return "", err
	}
	if origin == Preset {
		text, err := shipped.ReadFile(path.Join(shippedIn, title+Extension))
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, ErrNoTheme)
		}
		return string(text), nil
	}
	if c.dir == "" {
		return "", fmt.Errorf("%s: %w", name, ErrNoFolder)
	}
	text, err := read(filepath.Join(c.dir, title+Extension))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%s: %w", name, ErrNoTheme)
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

func (c Catalog) readPluginTheme(name string) (string, error) {
	pluginID, themeID, found := strings.Cut(strings.TrimPrefix(name, pluginPrefix), "/")
	if !found || pluginID == "" || themeID == "" || c.plugins == nil {
		return "", fmt.Errorf("%s: %w", name, ErrNoTheme)
	}
	text, err := c.plugins.ReadTheme(pluginID, themeID)
	if errors.Is(err, themecontract.ErrNoTheme) {
		return "", fmt.Errorf("%s: %w", name, ErrNoTheme)
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

// GetApplied is the theme a name selects. A name matching nothing yields this
// product's own palette, and the name that was not found is returned.
func (c Catalog) GetApplied(name string) (applied, missing string) {
	if name == "" {
		return Default, ""
	}
	if _, err := c.Text(name); err != nil {
		return Default, name
	}
	return name, ""
}

func splitName(name string) (Origin, string, error) {
	origin, title, found := strings.Cut(name, ":")
	if found && isPlainName(title) {
		switch Origin(origin) {
		case Preset, Mine:
			return Origin(origin), title, nil
		case Plugin:
			// A plugin theme is named by plugin and theme id and is resolved by readPluginTheme.
		}
	}
	return "", "", fmt.Errorf("%s: %w", name, ErrNoTheme)
}

func isPlainName(title string) bool {
	if title == "" || title == "." || title == ".." {
		return false
	}
	if strings.ContainsAny(title, `/\`) || strings.ContainsRune(title, 0) {
		return false
	}
	return title == filepath.Base(title)
}

func read(path string) (_ string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeAndJoin(&err, file)

	text, err := io.ReadAll(io.LimitReader(file, MaxSize+1))
	if err != nil {
		return "", err
	}
	if len(text) > MaxSize {
		return "", fmt.Errorf("%s: a theme is read up to %d bytes", path, MaxSize)
	}
	return string(text), nil
}

func declaresColorScheme(css string) bool { return hasDeclared(stripComments(css), "color-scheme") }

func stripComments(css string) string {
	var text strings.Builder
	for {
		start := strings.Index(css, "/*")
		if start < 0 {
			text.WriteString(css)
			return text.String()
		}
		text.WriteString(css[:start])
		end := strings.Index(css[start+2:], "*/")
		if end < 0 {
			return text.String()
		}
		css = css[start+2+end+2:]
	}
}

func hasDeclared(css, property string) bool {
	for pos := 0; ; {
		found := strings.Index(css[pos:], property)
		if found < 0 {
			return false
		}
		pos += found + len(property)
		if strings.HasPrefix(strings.TrimLeft(css[pos:], " \t\r\n"), ":") {
			return true
		}
	}
}
