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
// AtOnceImageLimit at once, every other file on tap. The media type is the one the file's extension names.
func ClassifyLayer(path string, size int64, mime string) Layer {
	if strings.EqualFold(pathpkg.Ext(path), ".md") {
		return AtOnce
	}
	if strings.HasPrefix(mime, "image/") && size <= AtOnceImageLimit {
		return AtOnce
	}
	return OnTap
}
