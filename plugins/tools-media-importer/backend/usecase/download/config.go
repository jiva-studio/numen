// Package download retrieves what an address publishes: the captions of a video site through
// yt-dlp, or the article of any other page, and copies the audio of a video.
package download

import (
	"maps"
	"slices"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

func getEnvironment(t fetch.Tool, base []string) []string {
	if len(t.Environment) == 0 {
		return nil
	}
	out := slices.Clone(base)
	for _, name := range slices.Sorted(maps.Keys(t.Environment)) {
		out = append(out, name+"="+t.Environment[name])
	}
	return out
}
