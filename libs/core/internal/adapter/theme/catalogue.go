// Package theme lists and applies the stylesheets that restyle the window.
//
// A theme is one CSS file redeclaring the tokens the interface renders with.
// Some ship inside the binary and the rest are files in the user's themes
// folder; both form one list with one format, and one name applies either.
// Nothing here reads inside a theme beyond whether it pins `color-scheme`.
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

// Origin is where a theme comes from. A theme carries its origin in its name, and
// two themes on different origins can share a filename.
type Origin string

const (
	// Preset ships inside the application.
	Preset Origin = "preset"

	// Mine is a `.css` file in the user's themes folder.
	Mine Origin = "mine"
)

// Theme is one theme as the list refers to it.
type Theme struct {
	// Name is the origin and the filename, `preset:dracula` or `mine:dracula`,
	// and it identifies the theme in later requests.
	Name string

	// Title is what the file is called, without the origin and without `.css`.
	Title string

	Origin Origin

	// IsPinned is set for a theme declaring `color-scheme` itself. Light and dark
	// are that theme's own, and the mode has nothing left to choose.
	IsPinned bool
}

// Catalogue is every theme this installation offers: the ones inside the
// binary, and the ones in the user's folder.
//
// A catalogue whose folder could not be created or read still offers the
// presets, so the window opens with a theme regardless of disk state.
type Catalogue struct{ dir string }

// Open returns the catalogue at the folder where this desktop keeps its
// configuration. The themes folder is created empty if it does not exist.
func Open() (Catalogue, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Catalogue{}, err
	}
	return OpenAt(filepath.Join(dir, "numen", "themes"))
}

// OpenAt is Open with an explicit path.
func OpenAt(dir string) (Catalogue, error) {
	made := os.MkdirAll(dir, 0o755)
	// The folder is where the links lead. The paths the operating system
	// reports changes at are resolved, and they are named against this.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return Catalogue{dir: dir}, made
}

// Dir is the folder the user's themes are read from.
func (c Catalogue) Dir() string { return c.dir }

// Themes is every theme, the shipped ones first and each origin ordered by
// name. A file that cannot be read is excluded: every name in this list is a
// theme that can be applied.
func (c Catalogue) Themes() []Theme {
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
	return append(themes, c.getOwnThemes()...)
}

// getOwnThemes returns the user's themes, read flat: one level, names ending
// `.css`. A folder is not a theme and neither is `dracula.css.bak`.
func (c Catalogue) getOwnThemes() []Theme {
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

func (c Catalogue) Text(name string) (string, error) {
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

// GetApplied is the theme a name selects. A name matching nothing yields this
// product's own palette, and the name that was not found is returned.
func (c Catalogue) GetApplied(name string) (applied, missing string) {
	if name == "" {
		return Default, ""
	}
	if _, err := c.Text(name); err != nil {
		return Default, name
	}
	return name, ""
}

// splitName takes a name apart into the origin it comes from and the file it names.
//
// A name resolves to a file inside the themes folder and nowhere else, so the
// half after the origin is a filename and nothing else: `mine:../../.ssh/id_rsa`
// names no theme.
func splitName(name string) (Origin, string, error) {
	origin, title, found := strings.Cut(name, ":")
	if found && isPlainName(title) {
		switch Origin(origin) {
		case Preset, Mine:
			return Origin(origin), title, nil
		}
	}
	return "", "", fmt.Errorf("%s: %w", name, ErrNoTheme)
}

// isPlainName is a name that stays in the folder it is joined to.
func isPlainName(title string) bool {
	if title == "" || title == "." || title == ".." {
		return false
	}
	if strings.ContainsAny(title, `/\`) || strings.ContainsRune(title, 0) {
		return false
	}
	return title == filepath.Base(title)
}

// read is a theme's file, bounded.
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

// declaresColorScheme reports whether a theme declares `color-scheme` itself.
// Comments are stripped first, so a mention in a comment does not count.
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

// hasDeclared looks for a property being set: the name, and a colon after it.
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
