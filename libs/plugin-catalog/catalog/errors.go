package catalog

import "errors"

// SourceErrors returns every *SourceError that err holds, directly or inside the errors it joins or
// wraps, in the order they were joined.
func SourceErrors(err error) []*SourceError {
	var found []*SourceError
	collectSourceErrors(err, &found)
	return found
}

func collectSourceErrors(err error, found *[]*SourceError) {
	switch wrapped := err.(type) {
	case nil:
		return
	case *SourceError:
		*found = append(*found, wrapped)
	case interface{ Unwrap() []error }:
		for _, inner := range wrapped.Unwrap() {
			collectSourceErrors(inner, found)
		}
	default:
		collectSourceErrors(errors.Unwrap(err), found)
	}
}
