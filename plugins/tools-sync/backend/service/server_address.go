package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

const keyInsecureServer = "tools-sync/insecure-server"

// readServerAddress returns the address the session runs over, and empty for an address it must not run over.
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
	_ = r.parts.health.Report(context.Background(), health.Item{
		Key: keyInsecureServer, Severity: health.Warning, Text: "Numen Sync is off: the sync server address is not https",
		Detail: address + " would send your session in clear text. Use an https address, or http on this computer only.",
		Action: health.OpenSettings(settingServer),
	})
}

func (r *runtime) clearInsecureServer() {
	if r.parts.health == nil {
		return
	}
	_ = r.parts.health.Clear(context.Background(), keyInsecureServer)
}
