package download

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

// ErrNoTool is an address no provider on this machine supports.
var ErrNoTool = errors.New("unsupported protocol scheme for download")

// producerCaptions names the text a video site publishes as captions.
const producerCaptions = "captions"

// Provider is one source of what is published at an address.
type Provider interface {
	CanHandle(address string) bool
	GetDownloadModel(address string) fetch.Model
	GetMetadata(ctx context.Context, address string) (fetch.Metadata, error)
	GetText(ctx context.Context, address string, want fetch.PreferredCaptions) (fetch.Text, error)
}

// Player is a provider that also fetches the bytes of the media to play.
type Player interface {
	Download(ctx context.Context, address string, media fetch.Media, into io.Writer) (fetch.Copy, error)
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
