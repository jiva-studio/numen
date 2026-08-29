//go:build tools

// The binding tool compiles this module against its own runtime, so the
// dependency is declared here where nothing else needs it.
package mobile

import _ "golang.org/x/mobile/bind"
