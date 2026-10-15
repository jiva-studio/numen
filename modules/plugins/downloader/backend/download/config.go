package download

import (
	"maps"
	"os"
	"slices"
)

// Config is what an installation says about reaching an address.
type Config struct {
	Captions          []string `json:"captions"`
	AutomaticCaptions *bool    `json:"automatic_captions"`
	CopyMaxSizeMB     int      `json:"copy_max_size_mb"`
	CopiesToVault     *bool    `json:"copies_to_vault"`
	Video             Tool     `json:"yt_dlp"`
	Sound             Tool     `json:"ffmpeg"`
}

// Tool is a program this machine holds.
type Tool struct {
	Command     []string          `json:"command"`
	Arguments   []string          `json:"arguments"`
	Environment map[string]string `json:"environment"`
}

func (t Tool) getEnvironment() []string {
	if len(t.Environment) == 0 {
		return nil
	}
	out := os.Environ()
	for _, name := range slices.Sorted(maps.Keys(t.Environment)) {
		out = append(out, name+"="+t.Environment[name])
	}
	return out
}

// Defaults returns default settings.
func Defaults() Config {
	automatic := true
	off := false
	return Config{
		AutomaticCaptions: &automatic,
		CopyMaxSizeMB:     500,
		CopiesToVault:     &off,
	}
}
