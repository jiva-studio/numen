// Package theme holds the themes plugins contribute and the errors that fail reading one.
package theme

import "errors"

// ErrNoTheme is a theme no enabled plugin contributes.
var ErrNoTheme = errors.New("no such plugin theme")
