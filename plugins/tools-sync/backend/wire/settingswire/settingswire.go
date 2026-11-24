// Package settingswire reports the settings of the sync server address and of the epoch root key through a health item
// when one of them is not valid.
package settingswire

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

const (
	// SettingServer is the name of the setting of the sync server address.
	SettingServer = "server"
	// SettingEpochRoot is the name of the setting of the epoch root key.
	SettingEpochRoot = "epochRootKey"

	keyEpochRoot      = "tools-sync/epoch-root"
	keyInsecureServer = "tools-sync/insecure-server"
)

// Settings reports the two settings.
type Settings struct {
	health *health.Client
	warn   func(message string, err error)
}

// New returns the settings that report to the health client, which may be nil, and write a failed report through warn.
func New(client *health.Client, warn func(message string, err error)) *Settings {
	return &Settings{health: client, warn: warn}
}

// ReportEpochRoot reports the error of reading the epoch root key, and clears the report when the error is nil.
func (s *Settings) ReportEpochRoot(ctx context.Context, err error) {
	if s.health == nil {
		return
	}
	if err != nil {
		reportErr := s.health.Report(ctx, health.Item{
			Key: keyEpochRoot, Severity: health.Warning, Text: "The epoch root key of Numen Sync is not 32 bytes of hex",
			Detail: "Restore announcements of the sync server are ignored until the key is fixed.", Action: health.OpenSettings(SettingEpochRoot),
		})
		if reportErr != nil {
			s.warn("the health report of the epoch root key failed", reportErr)
		}
		return
	}
	if clearErr := s.health.Clear(ctx, keyEpochRoot); clearErr != nil {
		s.warn("clearing the health report of the epoch root key failed", clearErr)
	}
}

// ReadServerAddress returns the address when it is https or http on this computer, and an empty address otherwise.
func (s *Settings) ReadServerAddress(address string) string {
	if address == "" {
		return ""
	}
	if err := syncprotocol.CheckServerAddress(address); err != nil {
		s.reportInsecureServer(address)
		return ""
	}
	s.clearInsecureServer()
	return address
}

func (s *Settings) reportInsecureServer(address string) {
	if s.health == nil {
		return
	}
	err := s.health.Report(context.Background(), health.Item{
		Key: keyInsecureServer, Severity: health.Warning, Text: "Numen Sync is off: the sync server address is not https",
		Detail: address + " would send your session in clear text. Use an https address, or http on this computer only.",
		Action: health.OpenSettings(SettingServer),
	})
	if err != nil {
		s.warn("the health report of the insecure server failed", err)
	}
}

func (s *Settings) clearInsecureServer() {
	if s.health == nil {
		return
	}
	if err := s.health.Clear(context.Background(), keyInsecureServer); err != nil {
		s.warn("clearing the health report of the insecure server failed", err)
	}
}
