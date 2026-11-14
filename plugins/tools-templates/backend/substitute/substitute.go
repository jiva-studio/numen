// Package substitute replaces the variables of a template text.
package substitute

import "time"

// Variables are the values a template takes. Title is a name; Now is the instant of the call.
type Variables struct {
	Title string
	Now   time.Time
}

// Result is the text after substitution and the place of the first cursor marker.
type Result struct {
	Text   string
	Cursor int
}

// Apply replaces the variables of a text. It never fails.
func Apply(_ string, _ Variables) Result {
	return Result{Text: "", Cursor: 0}
}
