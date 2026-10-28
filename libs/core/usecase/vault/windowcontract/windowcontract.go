// Package windowcontract re-exports the window port contract.
package windowcontract

import "github.com/jiva-studio/numen/libs/core/port/windowport"

type (
	ErrorHandler = windowport.ErrorHandler
	OpenTabsStore = windowport.OpenTabsStore
	Window = windowport.Window
)

