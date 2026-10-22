package download

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrNoTool is an address no provider on this machine supports.
	ErrNoTool = errors.New("nothing on this machine downloads that address")

	// ErrNothingDownloaded is an address that answered, and carries nothing of what was asked for.
	ErrNothingDownloaded = errors.New("nothing of the sort is published there")
)

// DownloadModel names what downloaded, and is kept along with what it brought back.
type DownloadModel struct {
	Tool     string
	Version  string
	Producer string
}

// Metadata is what a site says about an address, before any of it is taken.
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
	CanHandle(at string) bool
	GetDownloadModel(at string) DownloadModel
	GetMetadata(ctx context.Context, at string) (Metadata, error)
	GetText(ctx context.Context, at string, want PreferredCaptions) (Text, error)
}

// Player is a provider that also reaches the bytes a person plays.
type Player interface {
	Download(ctx context.Context, at string, into io.Writer) (Copy, error)
}
