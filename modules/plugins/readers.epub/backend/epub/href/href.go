package href

import (
	"net/url"
	"path"
	"strings"
)

// Resolve turns an href into an archive path, relative to the directory of the document holding it.
func Resolve(base, href string) string {
	if href == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(href); err == nil {
		href = unescaped
	}
	if strings.HasPrefix(href, "/") {
		return path.Clean(strings.TrimPrefix(href, "/"))
	}
	return path.Join(base, href)
}

// Split separates the document an href names from the place inside it.
func Split(href string) (target, fragment string) {
	target, fragment, _ = strings.Cut(href, "#")
	return target, fragment
}

// ResolveWithFragment resolves an href against the directory of the document holding it,
// keeping the fragment.
func ResolveWithFragment(base, href string) string {
	target, fragment := Split(href)
	pos := Resolve(base, target)
	if pos == "" {
		return ""
	}
	if fragment == "" {
		return pos
	}
	return pos + "#" + fragment
}
