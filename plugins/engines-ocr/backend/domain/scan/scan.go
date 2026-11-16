// Package scan knows which files the plugin recognizes. It is pure: it reads no file.
package scan

import (
	"path"
	"strings"
)

// IsScan reports whether the extension of file names a PDF or an image the plugin recognizes: the
// content types of the handler of the manifest.
func IsScan(file string) bool {
	switch strings.ToLower(path.Ext(file)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".jpe", ".gif", ".webp":
		return true
	}
	return false
}
