package testsupport

import "github.com/jiva-studio/numen/modules/libs/core/highlight"

// Box is one run of prose on a page: where it stands in the text, and the
// fraction of the page it covers.
func Box(page, start, length int, over highlight.Rect) highlight.Box {
	return highlight.Box{
		Page:    page,
		Stretch: highlight.Stretch{Start: start, Length: length},
		Rect:    over,
	}
}
