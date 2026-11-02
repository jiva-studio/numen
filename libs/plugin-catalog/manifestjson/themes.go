package manifestjson

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// ReadThemeFile returns the stylesheet of a theme declared by the plugin in dir. The file is
// resolved through symbolic links and must be a regular file inside the folder, of at most
// manifest.MaxThemeSize bytes.
func ReadThemeFile(dir, file string) (_ string, err error) {
	if err := manifest.CheckThemeFile(file); err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file %q leaves the plugin folder", file)
	}
	handle, err := os.Open(resolved)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, handle.Close()) }()
	info, err := handle.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("file %q is not a regular file", file)
	}
	text, err := io.ReadAll(io.LimitReader(handle, manifest.MaxThemeSize+1))
	if err != nil {
		return "", err
	}
	if len(text) > manifest.MaxThemeSize {
		return "", fmt.Errorf("file %q is larger than %d bytes", file, manifest.MaxThemeSize)
	}
	return string(text), nil
}
