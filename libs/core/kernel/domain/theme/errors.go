// Package theme holds the sentinel errors of the themes plugins contribute.
package theme

import "errors"

// ErrNoTheme is a theme no enabled plugin contributes.
var ErrNoTheme = errors.New("no such plugin theme")
