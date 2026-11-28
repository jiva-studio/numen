// Package layer says which files of a synced vault a phone downloads at once and which it downloads when the
// person opens them.
package layer

import (
	pathpkg "path"
	"strings"
)

// AtOnceImageLimit is the size in bytes up to which an image is downloaded at once.
const AtOnceImageLimit = 5 << 20

// Layer is when a file is downloaded to a phone.
type Layer int

const (
	// AtOnce is a file downloaded in the pass that finds it.
	AtOnce Layer = iota + 1
	// OnTap is a file downloaded when the person opens it.
	OnTap
)

// ClassifyLayer returns when the file is downloaded: a markdown file of any size and an image up to
// AtOnceImageLimit at once, every other file on tap. A file is an image when its extension names one, without regard to case.
func ClassifyLayer(path string, size int64) Layer {
	ext := strings.ToLower(pathpkg.Ext(path))
	if ext == ".md" {
		return AtOnce
	}
	if isImageExtension(ext) && size <= AtOnceImageLimit {
		return AtOnce
	}
	return OnTap
}

func isImageExtension(ext string) bool {
	switch ext {
	case ".avif", ".bmp", ".gif", ".heic", ".jpeg", ".jpg", ".png", ".svg", ".webp":
		return true
	}
	return false
}
