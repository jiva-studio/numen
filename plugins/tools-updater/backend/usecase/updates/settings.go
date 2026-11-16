// Package updates checks whether the endpoint a channel setting selects publishes a build newer
// than the running one, and keeps the result of the last check.
package updates

import "strings"

// Endpoints of the release feeds of the two channels.
const (
	StableEndpoint  = "https://dl.numen.md/latest/latest.json"
	NightlyEndpoint = "https://dl.numen.md/nightly/latest.json"
)

// channelNightly is the value of the channel setting that selects the nightly feed.
const channelNightly = "nightly"

// channelCustom is the value of the channel setting that selects the endpoint setting.
const channelCustom = "custom"

// nightlyMark is what the version of a nightly build holds, as in `2026.11.5-nightly.2113`.
const nightlyMark = "-nightly."

type selection struct {
	channel  string
	endpoint string
}

func newSelection(values map[string]string) selection {
	return selection{channel: strings.TrimSpace(values["channel"]), endpoint: strings.TrimSpace(values["endpoint"])}
}

func (s selection) getEndpoint(version string) string {
	channel := s.channel
	if channel == "" {
		channel = getBuildChannel(version)
	}
	switch channel {
	case channelCustom:
		return s.endpoint
	case channelNightly:
		return NightlyEndpoint
	default:
		return StableEndpoint
	}
}

func getBuildChannel(version string) string {
	if strings.Contains(version, nightlyMark) {
		return channelNightly
	}
	return "stable"
}

// GetEndpoint returns the feed the settings of the build of that version select.
func GetEndpoint(values map[string]string, version string) string {
	return newSelection(values).getEndpoint(version)
}
