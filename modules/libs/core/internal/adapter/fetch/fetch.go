// Package fetch reaches an address and brings back what is at it.
//
// Each way of reaching one is a strategy of its own: it says which addresses it
// answers for, and it is asked for nothing else. A page is fetched in this
// process, and what a site publishes as a video is fetched by running yt-dlp,
// with ffmpeg beside it where sound has to be brought to what a transcriber
// opens. Another site with an API of its own is another strategy and nothing
// more; whoever adds it adds it to the list.
//
// No tool is linked and none is shipped. Each is named by a setting holding a
// command and what it is started through, and an empty one asks the path.
package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// ErrNoTool is an address nothing on this machine reaches. Every address a
// browser would go to is reached by something here, so this is a machine
// missing a tool one of the strategies runs.
var ErrNoTool = errors.New("nothing on this machine reaches that address")

// A reach is one way of getting at what is published at an address.
//
// It answers for the addresses it says it does and is asked for nothing else,
// so a strategy that knows one site is written as the whole of that site's
// answer and nothing has to be taught about it elsewhere.
type reach interface {
	port.Fetcher

	// Reaches says whether this is the one for an address. It is asked of each
	// in turn, and the first that says so is the one that answers.
	Reaches(at domain.WebAddress) bool
}

// A Fetcher is the strategies this machine holds, asked in order.
type Fetcher struct{ through []reach }

// New is a fetcher for this machine.
//
// The strategies stand in the order they are asked, narrowest first: a video
// site is reached by the tool that knows it, and a page is what is left, which
// is why it stands last and why every machine reaches one.
func New(ctx context.Context, c Config) (*Fetcher, error) {
	return &Fetcher{through: []reach{newVideos(ctx, c), newPages()}}, nil
}

// reaching is the strategy for an address, and ErrNoTool where this machine
// holds none.
func (f *Fetcher) reaching(at domain.WebAddress) (reach, error) {
	for _, one := range f.through {
		if one.Reaches(at) {
			return one, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", at.URL, ErrNoTool)
}

// Fetching is what this address is fetched by, and nothing where nothing
// reaches it.
func (f *Fetcher) Fetching(at domain.WebAddress) port.FetchModel {
	through, err := f.reaching(at)
	if err != nil {
		return port.FetchModel{}
	}
	return through.Fetching(at)
}

// Metadata is what stands at the address, taking none of it.
func (f *Fetcher) Metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
	through, err := f.reaching(at)
	if err != nil {
		return port.Metadata{}, err
	}
	return through.Metadata(ctx, at)
}

// Subtitles are the words published with what is at the address.
func (f *Fetcher) Subtitles(
	ctx context.Context, at domain.WebAddress, language string,
) ([]transcript.Cue, error) {
	through, err := f.reaching(at)
	if err != nil {
		return nil, err
	}
	return through.Subtitles(ctx, at, language)
}

// Audio is the sound of what is at the address, as the container a transcriber
// opens.
func (f *Fetcher) Audio(ctx context.Context, at domain.WebAddress, into io.Writer) error {
	through, err := f.reaching(at)
	if err != nil {
		return err
	}
	return through.Audio(ctx, at, into)
}

// Download is what is at the address as a person plays it.
func (f *Fetcher) Download(
	ctx context.Context, at domain.WebAddress, into io.Writer,
) (port.Download, error) {
	through, err := f.reaching(at)
	if err != nil {
		return port.Download{}, err
	}
	return through.Download(ctx, at, into)
}

// Article is the prose an address is written around.
func (f *Fetcher) Article(ctx context.Context, at domain.WebAddress) (port.Article, error) {
	through, err := f.reaching(at)
	if err != nil {
		return port.Article{}, err
	}
	return through.Article(ctx, at)
}

// resolved is the command to run, and nothing where this machine has no such
// tool. A named command is taken as it stands: a machine that writes the path
// afresh at every build names whatever does know where the tool is.
func resolved(named Tool, tool string) []string {
	if len(named.Command) > 0 {
		return append(append([]string(nil), named.Command...), named.Arguments...)
	}
	found, err := exec.LookPath(tool)
	if err != nil {
		return nil
	}
	return append([]string{found}, named.Arguments...)
}

// run is one tool, waited for. What it wrote to its error stream is what a
// person is shown when it failed: the tool knows why, and nothing here is going
// to say it better.
func run(ctx context.Context, command []string, into io.Writer, arguments ...string) ([]byte, error) {
	running := exec.CommandContext(ctx, command[0],
		append(append([]string(nil), command[1:]...), arguments...)...)
	var out, said bytes.Buffer
	running.Stdout, running.Stderr = &out, &said
	if into != nil {
		running.Stdout = into
	}
	if err := running.Run(); err != nil {
		// A run this machine stopped is that, and the tool's last words are
		// about being killed.
		if stopped := ctx.Err(); stopped != nil {
			return nil, stopped
		}
		return nil, fmt.Errorf("%w: %s", err, lastLine(said.String()))
	}
	return out.Bytes(), nil
}

// lastLine is what the tool said, as one line a person reads. A tool that
// refuses says why at length, and the last of it is the reason.
func lastLine(said string) string {
	lines := strings.Split(strings.TrimSpace(said), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return "it said nothing"
}
