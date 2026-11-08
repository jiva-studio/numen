package catalog

import "errors"

// SourceErrors returns every *SourceError that err holds, directly or inside the errors it joins or
// wraps, in the order they were joined.
func SourceErrors(err error) []*SourceError {
	var found []*SourceError
	collectSourceErrors(err, &found)
	return found
}

type joinedErrors interface{ Unwrap() []error }

type sourceFailure interface{ getSourceError() *SourceError }

func (e *SourceError) getSourceError() *SourceError { return e }

func collectSourceErrors(err error, found *[]*SourceError) {
	if err == nil {
		return
	}
	if failure, ok := err.(sourceFailure); ok {
		*found = append(*found, failure.getSourceError())
		return
	}
	if joined, ok := err.(joinedErrors); ok {
		for _, inner := range joined.Unwrap() {
			collectSourceErrors(inner, found)
		}
		return
	}
	collectSourceErrors(errors.Unwrap(err), found)
}
