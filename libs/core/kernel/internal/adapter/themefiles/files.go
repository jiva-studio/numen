// Package themefiles reads the themes that ship inside the binary and the themes in the user's
// folder, and watches that folder.
package themefiles

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

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
)

//go:embed presets/*.css
var shipped embed.FS

const shippedIn = "presets"

// Files is the presets inside the binary and the folder of the user's themes. A folder that could
// not be created or read leaves the presets, so the window opens with a theme regardless of disk
// state. The zero value has no folder.
type Files struct {
	dir string
	hub *watchhub.Hub
}

// Open returns the files at the folder where this desktop keeps its configuration. The themes
// folder is created empty if it does not exist.
func Open(hub *watchhub.Hub) (Files, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Files{dir: "", hub: nil}, err
	}
	return OpenAt(filepath.Join(dir, "numen", "themes"), hub)
}

// OpenAt is Open with an explicit path. A watch is placed through the hub. The files are usable
// when the error says the folder could not be made.
func OpenAt(dir string, hub *watchhub.Hub) (Files, error) {
	made := os.MkdirAll(dir, 0o755)
	// The folder is where the links lead. The paths the operating system
	// reports changes at are resolved, and they are named against this.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return Files{dir: dir, hub: hub}, made
}

// Dir is the folder the user's themes are read from.
func (f Files) Dir() string { return f.dir }

// ListPresets returns the themes that ship, ordered by title.
func (f Files) ListPresets() []themedomain.File {
	entries, err := fs.ReadDir(shipped, shippedIn)
	if err != nil {
		return nil
	}
	var files []themedomain.File
	for _, entry := range entries {
		text, err := shipped.ReadFile(path.Join(shippedIn, entry.Name()))
		if err != nil {
			continue
		}
		css := string(text)
		files = append(files, themedomain.File{
			Title:    strings.TrimSuffix(entry.Name(), themedomain.Extension),
			Text:     css,
			IsPinned: DeclaresColorScheme(css),
		})
	}
	return files
}

// ReadPreset returns the stylesheet of the preset.
func (f Files) ReadPreset(title string) (string, error) {
	text, err := shipped.ReadFile(path.Join(shippedIn, title+themedomain.Extension))
	if err != nil {
		return "", err
	}
	return string(text), nil
}

// ListOwnThemes returns the themes of the folder, ordered by name. A file that cannot be read is
// excluded: every title in this list is a theme that can be read.
func (f Files) ListOwnThemes() []themedomain.File {
	if f.dir == "" {
		return nil
	}
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return nil
	}
	var files []themedomain.File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), themedomain.Extension) {
			continue
		}
		title := strings.TrimSuffix(entry.Name(), themedomain.Extension)
		if !themedomain.IsPlainTitle(title) {
			continue
		}
		text, err := read(filepath.Join(f.dir, entry.Name()))
		if err != nil {
			continue
		}
		files = append(files, themedomain.File{
			Title:    title,
			Text:     text,
			IsPinned: DeclaresColorScheme(text),
		})
	}
	return files
}

// ReadOwnTheme returns the stylesheet of the title, themedomain.ErrNoTheme where the folder has no
// such file, or themedomain.ErrNoFolder where there is no folder.
func (f Files) ReadOwnTheme(title string) (string, error) {
	if f.dir == "" {
		return "", themedomain.ErrNoFolder
	}
	text, err := read(filepath.Join(f.dir, title+themedomain.Extension))
	if errors.Is(err, fs.ErrNotExist) {
		return "", themedomain.ErrNoTheme
	}
	return text, err
}

func read(path string) (_ string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeAndJoin(&err, file)

	text, err := io.ReadAll(io.LimitReader(file, themedomain.MaxSize+1))
	if err != nil {
		return "", err
	}
	if len(text) > themedomain.MaxSize {
		return "", fmt.Errorf("%s: a theme is read up to %d bytes", path, themedomain.MaxSize)
	}
	return string(text), nil
}

// DeclaresColorScheme reports whether the stylesheet pins `color-scheme`, outside its comments.
func DeclaresColorScheme(css string) bool { return hasDeclared(stripComments(css), "color-scheme") }

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
