// Package theme holds the themes of the window, the rules that name and read them, and the errors that fail reading one.
package theme

import "errors"

// ErrNoTheme is a theme name no theme answers for.
var ErrNoTheme = errors.New("no such theme")

// ErrNoFolder is returned where there is no folder to read the user's themes from.
var ErrNoFolder = errors.New("there is no themes folder")
