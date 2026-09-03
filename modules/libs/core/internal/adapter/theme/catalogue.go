// Package theme is the stylesheets a person may dress the window in.
//
// A theme is one CSS file redeclaring the tokens the interface draws with. Some
// ship inside the binary and the rest are files in a folder of the person's;
// they are one list, one format, and one line applies either. Nothing here
// reads inside a theme beyond whether it pins `color-scheme`.
package theme

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed presets/*.css
var shipped embed.FS

const shippedIn = "presets"

// Extension is what a file is called to be a theme.
const Extension = ".css"

// MaxSize is the largest file read as a theme. Its text is spliced into the
// page before the window draws, and the presets are under two kilobytes each.
const MaxSize = 256 << 10

// Default is this product's own palette, and what a name matching nothing
// wears. It is the theme an installation nobody has dressed is set to.
const Default = "preset:numen"

// Shelf is where a theme came off. A theme carries its shelf in its name, and
// two themes on different shelves can share a filename.
type Shelf string

const (
	// Preset ships inside the application.
	Preset Shelf = "preset"

	// Mine is a `.css` file in the person's themes folder.
	Mine Shelf = "mine"
)

// Theme is one theme as the list refers to it.
type Theme struct {
	// Name is the shelf and the filename, `preset:dracula` or `mine:dracula`,
	// and it is how the theme is asked for again.
	Name string

	// Title is what the file is called, without the shelf and without `.css`.
	Title string

	Shelf Shelf

	// Pinned is set for a theme declaring `color-scheme` itself. Light and dark
	// are that theme's own, and the mode has nothing left to choose.
	Pinned bool
}

// Catalogue is every theme this installation offers: the ones inside the
// binary, and the person's folder of them.
//
// A catalogue whose folder could not be made or read still offers the presets,
// so the window opens wearing something whatever the disk says.
type Catalogue struct{ dir string }

// Open is the catalogue at the folder this desktop keeps a person's
// configuration in. The themes folder is made, empty, if it is not there.
func Open() (Catalogue, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Catalogue{}, err
	}
	return At(filepath.Join(dir, "numen", "themes"))
}

// At is Open with an explicit path.
func At(dir string) (Catalogue, error) {
	made := os.MkdirAll(dir, 0o755)
	// The folder is where the links lead. The paths the operating system
	// reports changes at are resolved, and they are named against this.
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	return Catalogue{dir: dir}, made
}

// Dir is the folder the person's themes are read from.
func (c Catalogue) Dir() string { return c.dir }

// Themes is every theme there is, the shipped ones first and each shelf by
// name. A file that cannot be read is not among them: a name in this list is a
// theme that can be worn.
func (c Catalogue) Themes() []Theme {
	var themes []Theme
	entries, err := fs.ReadDir(shipped, shippedIn)
	if err == nil {
		for _, entry := range entries {
			text, err := shipped.ReadFile(path.Join(shippedIn, entry.Name()))
			if err != nil {
				continue
			}
			themes = append(themes, described(Preset, entry.Name(), string(text)))
		}
	}
	return append(themes, c.mine()...)
}

// mine is the person's themes, read flat: one level, names ending `.css`. A
// folder is not a theme and neither is `dracula.css.bak`.
func (c Catalogue) mine() []Theme {
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
		if !plain(strings.TrimSuffix(entry.Name(), Extension)) {
			continue
		}
		text, err := read(filepath.Join(c.dir, entry.Name()))
		if err != nil {
			continue
		}
		themes = append(themes, described(Mine, entry.Name(), text))
	}
	return themes
}

func described(shelf Shelf, filename, text string) Theme {
	title := strings.TrimSuffix(filename, Extension)
	return Theme{
		Name:   string(shelf) + ":" + title,
		Title:  title,
		Shelf:  shelf,
		Pinned: pins(text),
	}
}

// Text is one theme's file, as the file stands when it is asked for.
func (c Catalogue) Text(name string) (string, error) {
	shelf, title, err := split(name)
	if err != nil {
		return "", err
	}
	if shelf == Preset {
		text, err := shipped.ReadFile(path.Join(shippedIn, title+Extension))
		if err != nil {
			return "", fmt.Errorf("%s: this build ships no such theme", name)
		}
		return string(text), nil
	}
	if c.dir == "" {
		return "", fmt.Errorf("%s: %w", name, ErrNoFolder)
	}
	text, err := read(filepath.Join(c.dir, title+Extension))
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

// Applied is the theme a name asks for. A name matching nothing wears this
// product's own palette, and is given back as the name that was not found.
func (c Catalogue) Applied(name string) (applied, missing string) {
	if name == "" {
		return Default, ""
	}
	if _, err := c.Text(name); err != nil {
		return Default, name
	}
	return name, ""
}

// split takes a name apart into the shelf it came off and the file it names.
//
// A name resolves to a file inside the themes folder and nowhere else, so the
// half after the shelf is a filename and nothing else: `mine:../../.ssh/id_rsa`
// names no theme.
func split(name string) (Shelf, string, error) {
	shelf, title, found := strings.Cut(name, ":")
	if found && plain(title) {
		switch Shelf(shelf) {
		case Preset, Mine:
			return Shelf(shelf), title, nil
		}
	}
	return "", "", fmt.Errorf("%s: a theme is named by its shelf and one file in the themes folder", name)
}

// plain is a name that stays in the folder it is joined to.
func plain(title string) bool {
	if title == "" || title == "." || title == ".." {
		return false
	}
	if strings.ContainsAny(title, `/\`) || strings.ContainsRune(title, 0) {
		return false
	}
	return title == filepath.Base(title)
}

// read is a theme's file, bounded.
func read(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	text, err := io.ReadAll(io.LimitReader(file, MaxSize+1))
	if err != nil {
		return "", err
	}
	if len(text) > MaxSize {
		return "", fmt.Errorf("%s: a theme is read up to %d bytes", path, MaxSize)
	}
	return string(text), nil
}

// pins says whether a theme declares `color-scheme` itself. The comments are
// cut away first: a preset that pins says so in one of them.
func pins(css string) bool { return declares(uncommented(css), "color-scheme") }

func uncommented(css string) string {
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

// declares looks for a property being set: the name, and a colon after it.
func declares(css, property string) bool {
	for at := 0; ; {
		found := strings.Index(css[at:], property)
		if found < 0 {
			return false
		}
		at += found + len(property)
		if strings.HasPrefix(strings.TrimLeft(css[at:], " \t\r\n"), ":") {
			return true
		}
	}
}
