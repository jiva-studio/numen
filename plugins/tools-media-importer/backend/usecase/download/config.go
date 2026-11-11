// Package download retrieves what an address publishes: the captions of a video site through
// yt-dlp, or the article of any other page, and copies the audio of a video.
package download

import (
	"maps"
	"slices"
)

// Config lists the programs that this machine uses to access a video site.
type Config struct {
	Video Tool
	Sound Tool
}

// Tool is a program this machine holds.
type Tool struct {
	Command     []string
	Arguments   []string
	Environment map[string]string
}

func (t Tool) getEnvironment(base []string) []string {
	if len(t.Environment) == 0 {
		return nil
	}
	out := slices.Clone(base)
	for _, name := range slices.Sorted(maps.Keys(t.Environment)) {
		out = append(out, name+"="+t.Environment[name])
	}
	return out
}

// Defaults is the programs PATH and the host name.
func Defaults() Config {
	// gomutants:disable-next-line RETURN_ZERO reason="explicit nil fields equal zero Config"
	return Config{
		Video: Tool{Command: nil, Arguments: nil, Environment: nil},
		Sound: Tool{Command: nil, Arguments: nil, Environment: nil},
	}
}
