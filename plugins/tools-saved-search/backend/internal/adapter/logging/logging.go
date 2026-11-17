// Package logging writes the messages of the use cases to the default log of the process.
package logging

import "log/slog"

// Log is the default slog logger of the process at the time of each message.
type Log struct{}

// Debug writes msg and its key-value pairs at the debug level.
func (Log) Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}
