package domain

import "errors"

// ErrUnavailable means whatever answers cannot be reached right now.
var ErrUnavailable = errors.New("unavailable")
