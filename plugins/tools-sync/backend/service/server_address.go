package service

import (
	"context"
	"log/slog"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

const keyInsecureServer = "tools-sync/insecure-server"

func (r *runtime) readServerAddress(address string) string {
	if address == "" {
		return ""
	}
	if err := syncprotocol.CheckServerAddress(address); err != nil {
		r.reportInsecureServer(address)
		return ""
	}
	r.clearInsecureServer()
	return address
}

func (r *runtime) reportInsecureServer(address string) {
	if r.parts.health == nil {
		return
	}
	err := r.parts.health.Report(context.Background(), health.Item{
		Key: keyInsecureServer, Severity: health.Warning, Text: "Numen Sync is off: the sync server address is not https",
		Detail: address + " would send your session in clear text. Use an https address, or http on this computer only.",
		Action: health.OpenSettings(settingServer),
	})
	if err != nil {
		slog.Warn("the health report of the insecure server failed", "error", err)
	}
}

func (r *runtime) clearInsecureServer() {
	if r.parts.health == nil {
		return
	}
	if err := r.parts.health.Clear(context.Background(), keyInsecureServer); err != nil {
		slog.Warn("clearing the health report of the insecure server failed", "error", err)
	}
}
