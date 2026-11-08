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
	{
		var wrapped *SourceError
		var wrapped1 interface{ Unwrap() []error }
		switch {
		case err == nil:
			return
		case errors.As(err, &wrapped):
			*found = append(*found, wrapped)
		case errors.As(err, &wrapped1):
			for _, inner := range wrapped1.Unwrap() {
				collectSourceErrors(inner, found)
			}
		default:
			collectSourceErrors(errors.Unwrap(err), found)
		}
	}
}
