// Package logging writes the failures of the plugin to the default log of the process.
package logging

import "log/slog"

// Warn writes message and the error at the warning level.
func Warn(message string, err error) {
	slog.Warn(message, "error", err)
}
