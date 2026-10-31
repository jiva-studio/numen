// Package themecontract re-exports the plugin theme port contract.
package themecontract

import "github.com/jiva-studio/numen/libs/core/port/themeport"

type (
	PluginTheme  = themeport.PluginTheme
	PluginThemes = themeport.PluginThemes
)

// ErrNoTheme is a theme no enabled plugin contributes.
var ErrNoTheme = themeport.ErrNoTheme
