// Package log writes the failures of the plugin to the process log.
package log

import "log/slog"

// Slog writes to the default slog logger.
type Slog struct{}

func (Slog) Error(message string, err error) {
	slog.Error(message, "error", err)
}
