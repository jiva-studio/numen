package chunking

import "fmt"

// textVersionReader names the text a file source is cut from: the live text version of the file. It is
// part of every recipe of such a cut, and changes when the text or the offsets a version gives do.
const textVersionReader = "text-version-1"

// GetTextVersionRecipe names how a live text version is cut into chunks at sizes, so that sources cut by
// another recipe are cut again.
func GetTextVersionRecipe(sizes Sizes) string {
	s := sizes.Resolve()
	return fmt.Sprintf("%s/large=%d+%d/small=%d+%d/limit=%d",
		textVersionReader, s.Large, s.LargeOverlap, s.Small, s.SmallOverlap, s.Limit)
}
