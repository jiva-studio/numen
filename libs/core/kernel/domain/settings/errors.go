// Package settings holds the sentinel errors of the settings.
package settings

import "errors"

// ErrNotASetting is a value the settings cannot be read out of again: one that
// is not JSON, one of the wrong shape, or a number past what its setting goes
// to. Nothing is written, and the value is the caller's to correct.
var ErrNotASetting = errors.New("the settings cannot be read out of that value")
