// Package fetch reaches an address and brings back what is at it.
//
// Each source of what is published at one is a provider of its own: it says
// which addresses it supports, and it is asked for nothing else. A page is
// fetched in this process, and what a site publishes as a video is fetched by
// running yt-dlp, with ffmpeg beside it where sound has to be brought to what a
// transcriber opens. A site with an API of its own is another provider and
// nothing more; whoever adds it adds it to the list.
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

// ErrNoTool is an address no provider on this machine supports, which is a
// machine missing a tool one of them runs.
var ErrNoTool = errors.New("nothing on this machine fetches that address")

// A provider is one source of what is published at an address.
//
// It answers for the addresses it supports and is asked for nothing else, so a
// provider that knows one site is written as the whole of that site's answer
// and nothing has to be taught about it elsewhere.
type provider interface {
	port.Fetcher

	// Supports says whether this provider answers for an address. Each is asked
	// in turn, and the first that says so is the one that answers.
	Supports(at domain.WebAddress) bool
}

// A Fetcher is the providers this machine holds, asked in order.
type Fetcher struct{ providers []provider }

// New is a fetcher for this machine.
//
// The providers stand in the order they are asked, narrowest first: a video is
// fetched by the tool that knows the site, and a page is what an address is
// when nothing knows it better, which is why pages stand last and why every
// machine fetches one.
func New(ctx context.Context, c Config) (*Fetcher, error) {
	return &Fetcher{providers: []provider{newYtDLP(ctx, c), newPages()}}, nil
}

// providerFor is the provider that answers for an address, and ErrNoTool where
// this machine holds none.
func (f *Fetcher) providerFor(at domain.WebAddress) (provider, error) {
	for _, one := range f.providers {
		if one.Supports(at) {
			return one, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", at.URL, ErrNoTool)
}

// Fetching is what this address is fetched by, and nothing where nothing
// reaches it.
func (f *Fetcher) Fetching(at domain.WebAddress) port.FetchModel {
	by, err := f.providerFor(at)
	if err != nil {
		return port.FetchModel{}
	}
	return by.Fetching(at)
}

// Metadata is what stands at the address, taking none of it.
func (f *Fetcher) Metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return port.Metadata{}, err
	}
	return by.Metadata(ctx, at)
}

// Subtitles are the words published with what is at the address.
func (f *Fetcher) Subtitles(
	ctx context.Context, at domain.WebAddress, language string,
) ([]transcript.Cue, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return nil, err
	}
	return by.Subtitles(ctx, at, language)
}

// Audio is the sound of what is at the address, as the container a transcriber
// opens.
func (f *Fetcher) Audio(ctx context.Context, at domain.WebAddress, into io.Writer) error {
	by, err := f.providerFor(at)
	if err != nil {
		return err
	}
	return by.Audio(ctx, at, into)
}

// Download is what is at the address as a person plays it.
func (f *Fetcher) Download(
	ctx context.Context, at domain.WebAddress, into io.Writer,
) (port.Download, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return port.Download{}, err
	}
	return by.Download(ctx, at, into)
}

// Article is the prose an address is written around.
func (f *Fetcher) Article(ctx context.Context, at domain.WebAddress) (port.Article, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return port.Article{}, err
	}
	return by.Article(ctx, at)
}

// A program is a tool as it is started: the command, and the environment the
// settings name for it. Every run goes through started, so what a setting says
// about the environment cannot be forgotten at one of them.
type program struct {
	command []string
	env     []string
}

// held says this machine has the tool.
func (p program) held() bool { return len(p.command) > 0 }

// at is where the tool itself is, for another tool that runs it. A tool started
// through something else is not somewhere one path names.
func (p program) at() string {
	if len(p.command) != 1 {
		return ""
	}
	return p.command[0]
}

// started is one run of it, with the arguments of that run after its own.
func (p program) started(ctx context.Context, arguments ...string) *exec.Cmd {
	running := exec.CommandContext(ctx, p.command[0],
		append(append([]string(nil), p.command[1:]...), arguments...)...)
	running.Env = p.env
	return running
}

// resolved is the program to run, and nothing where this machine has no such
// tool. A named command is taken as it stands: a machine that writes the path
// afresh at every build names whatever does know where the tool is.
func resolved(named Tool, tool string) program {
	if len(named.Command) > 0 {
		return program{
			command: append(append([]string(nil), named.Command...), named.Arguments...),
			env:     named.env(),
		}
	}
	found, err := exec.LookPath(tool)
	if err != nil {
		return program{}
	}
	return program{command: append([]string{found}, named.Arguments...), env: named.env()}
}

// run is one tool, waited for. What it wrote to its error stream is what a
// person is shown when it failed: the tool knows why, and nothing here is going
// to say it better.
func run(ctx context.Context, tool program, into io.Writer, arguments ...string) ([]byte, error) {
	running := tool.started(ctx, arguments...)
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
