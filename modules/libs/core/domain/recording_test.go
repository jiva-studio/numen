package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// What a vault walks as a recording and what a player is told it is being given
// are the one table. A container named in the one and not the other is a source
// the vault holds and the player refuses.
func TestWhatIsWalkedIsWhatIsPlayed(t *testing.T) {
	for _, one := range domain.RecordingExtensions() {
		if domain.MediaType("talk"+one) == "" {
			t.Errorf("a vault walks %s and nothing says what it is played as", one)
		}
	}
}

// A name is read whatever case it is written in, and a name no recording is
// kept under is played as nothing.
func TestWhatAFileIsPlayedAs(t *testing.T) {
	for name, want := range map[string]string{
		"talks/one.mp3":  "audio/mpeg",
		"talks/ONE.MP3":  "audio/mpeg",
		"talks/one.wav":  "audio/wav",
		"talks/one.flac": "audio/flac",
		"talks/one.md":   "",
		"talks/one":      "",
	} {
		if got := domain.MediaType(name); got != want {
			t.Errorf("%s is played as %q, want %q", name, got, want)
		}
	}
}
