package download

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrNoTool is an address no provider on this machine supports.
	ErrNoTool = errors.New("unsupported protocol scheme for download")

	// ErrNothingDownloaded is returned when an address responded but contains nothing of what was requested.
	ErrNothingDownloaded = errors.New("nothing of the sort is published there")
)

// producerCaptions names the text a video site publishes as captions.
const producerCaptions = "captions"

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

// Cue is one stretch of speech: the transcript, and the milliseconds of the recording it spans.
type Cue struct {
	Text string
	From int
	To   int
}

// Text is what an address publishes as words.
type Text struct {
	Producer string
	Cues     []Cue
	Prose    string
	Title    string
	Length   int
}

// Copy is what a downloaded copy came to: media type and extension.
type Copy struct {
	MediaType string
	Extension string
}

// Provider is one source of what is published at an address.
type Provider interface {
	CanHandle(address string) bool
	GetDownloadModel(address string) Model
	GetMetadata(ctx context.Context, address string) (Metadata, error)
	GetText(ctx context.Context, address string, want PreferredCaptions) (Text, error)
}

// Player is a provider that also fetches the bytes of the media to play.
type Player interface {
	Download(ctx context.Context, address string, into io.Writer) (Copy, error)
}
