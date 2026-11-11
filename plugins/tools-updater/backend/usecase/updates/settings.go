// Package updates checks whether the endpoint a channel setting selects publishes a build newer
// than the running one, and keeps the result of the last check.
package updates

import "strings"

// Endpoints of the release feeds of the two channels.
const (
	StableEndpoint = "https://dl.numen.md/latest/latest.json"
	BetaEndpoint   = "https://dl.numen.md/beta/latest.json"
)

// channelBeta is the value of the channel setting that selects the beta feed.
const channelBeta = "beta"

// channelCustom is the value of the channel setting that selects the endpoint setting.
const channelCustom = "custom"

// GetEndpoint returns the feed the settings select: the endpoint setting on the custom channel, the
// beta feed on the beta channel and the stable feed otherwise.
func GetEndpoint(values map[string]string) string {
	switch strings.TrimSpace(values["channel"]) {
	case channelCustom:
		return strings.TrimSpace(values["endpoint"])
	case channelBeta:
		return BetaEndpoint
	default:
		return StableEndpoint
	}
}
