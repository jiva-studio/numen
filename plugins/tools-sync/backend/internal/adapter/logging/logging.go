// Package logging writes the failures of the plugin to the default log of the process.
package logging

import (
	"context"
	"log/slog"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Warn writes message and the error at the warning level.
func Warn(message string, err error) {
	slog.Warn(message, "error", err)
}

// ReportPassFailure writes a failed sync pass over a vault at the warning level.
func ReportPassFailure(ctx context.Context, vault domain.VaultID, err error) {
	slog.WarnContext(ctx, "a pass over a vault failed", "operation", "sync pass", "vault", vault, "error", err)
}
