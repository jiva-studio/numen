// Package fetch holds the values of a fetch: what an address publishes as text and as media, the
// tool that fetched it, the programs this machine fetches with, and the user's fetch settings.
package fetch

import (
	"errors"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/ytdlp"
)

// ErrNothingDownloaded is returned when an address responded but contains nothing of what was requested.
var ErrNothingDownloaded = errors.New("nothing of the sort is published there")

// Model names what downloaded, and is kept along with what it brought back.
type Model struct {
	Tool     string
	Version  string
	Producer string
}

// Metadata is what a site reports about an address, before anything is downloaded.
type Metadata struct {
	Title       string
	Bytes       int64
	Length      int
	Language    string
	Captions    []string
	IsAutomatic []string
}

// PreferredCaptions is which of the words a site published the caller will take.
type PreferredCaptions struct {
	Languages   []string
	IsAutomatic bool
}

// Text is what an address publishes as words.
type Text struct {
	Producer string
	Cues     []ytdlp.Cue
	Prose    string
	Title    string
	Length   int
}

// Copy is what a downloaded copy came to: media type and extension.
type Copy struct {
	MediaType string
	Extension string
}

// Media is what a download of an address brings: the video with its sound, or the sound alone.
type Media string

const (
	MediaVideo Media = "video"
	MediaAudio Media = "audio"
)

// ParseMedia returns the media a name selects; the empty name selects the video. A name that
// selects no media gives the video and false.
func ParseMedia(name string) (Media, bool) {
	switch Media(name) {
	case "", MediaVideo:
		return MediaVideo, true
	case MediaAudio:
		return MediaAudio, true
	}
	return MediaVideo, false
}

// Settings are the user's fetch settings. CopyMaxSize is in bytes; zero means no limit.
type Settings struct {
	Captions    []string
	IsAutomatic bool
	CopyMaxSize int64
}
