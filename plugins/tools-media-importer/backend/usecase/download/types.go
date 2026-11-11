package download

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/ytdlp"
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

// Provider is one source of what is published at an address.
type Provider interface {
	CanHandle(address string) bool
	GetDownloadModel(address string) Model
	GetMetadata(ctx context.Context, address string) (Metadata, error)
	GetText(ctx context.Context, address string, want PreferredCaptions) (Text, error)
}

// Media is what a download of an address brings: the video with its sound, or the sound alone.
type Media string

const (
	MediaVideo Media = "video"
	MediaAudio Media = "audio"
)

// Player is a provider that also fetches the bytes of the media to play.
type Player interface {
	Download(ctx context.Context, address string, media Media, into io.Writer) (Copy, error)
}

// Runner finds and runs external programs on the machine.
type Runner interface {
	LookPath(file string) (string, error)
	Environ() []string
	Run(ctx context.Context, command, env []string, into io.Writer, arguments ...string) (stdout []byte, combined string, err error)
	ReportError(what string, err error)
}

// Paths writes the paths of the machine.
type Paths interface {
	// Join returns the path of the file called name in dir.
	Join(dir, name string) string
}

// Web fetches the page published at an address.
type Web interface {
	Fetch(ctx context.Context, target string) (io.ReadCloser, error)
}

// Files manages temporary files and directories for a download.
type Files interface {
	WriteTemp(pattern string, content []byte) (string, error)
	ReadFile(path string) ([]byte, error)
	ReadDir(dir string) ([]string, error)
	CopyFile(path string, into io.Writer) error
	WithTempDir(pattern string, fn func(dir string) error) error
}

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
