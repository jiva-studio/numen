package manifestjson

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
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
	// gomutants:disable-next-line EXPRESSION_REMOVE,INVERT_LOGICAL reason="filepath.Rel between two absolute paths from EvalSymlinks does not fail"
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file %q leaves the plugin folder", file)
	}
	handle, err := os.Open(resolved)
	// gomutants:disable-next-line BRANCH_IF reason="file resolved by EvalSymlinks opens unless permissions change concurrently"
	if err != nil {
		return "", err
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="closing read-only file does not fail"
	defer func() { err = errors.Join(err, handle.Close()) }()
	info, err := handle.Stat()
	// gomutants:disable-next-line BRANCH_IF reason="Stat on open file descriptor does not fail"
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("file %q is not a regular file", file)
	}
	// gomutants:disable-next-line INTEGER_INCREMENT reason="any limit above MaxThemeSize detects oversized files"
	text, err := io.ReadAll(io.LimitReader(handle, manifest.MaxThemeSize+1))
	// gomutants:disable-next-line BRANCH_IF reason="reading open regular file does not fail"
	if err != nil {
		return "", err
	}
	if len(text) > manifest.MaxThemeSize {
		return "", fmt.Errorf("file %q is larger than %d bytes", file, manifest.MaxThemeSize)
	}
	return string(text), nil
}
