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

// GetEndpoint returns the feed the settings select: the endpoint setting on the custom channel, the
// nightly feed on the nightly channel and the stable feed otherwise.
func GetEndpoint(values map[string]string) string {
	switch strings.TrimSpace(values["channel"]) {
	case channelCustom:
		return strings.TrimSpace(values["endpoint"])
	case channelNightly:
		return NightlyEndpoint
	default:
		return StableEndpoint
	}
}
