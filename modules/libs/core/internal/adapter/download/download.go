// Package download reaches an address and brings back what is at it.
//
// Each source of what is published at one is a provider of its own: it says
// which addresses it supports, and it is asked for nothing else. A page is
// downloaded in this process, and what a site publishes as a video is
// downloaded by running yt-dlp, with ffmpeg beside it where sound has to be
// brought to what a transcriber opens. A site with an API of its own is another
// provider and nothing more; whoever adds it adds it to the list.
//
// No tool is linked and none is shipped. Each is named by a setting holding a
// command and what it is started through, and an empty one asks the path.
package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNoTool is an address no provider on this machine supports, which is a
// machine missing a tool one of them runs.
var ErrNoTool = errors.New("nothing on this machine downloads that address")

// A provider is one source of what is published at an address.
//
// It answers for the addresses it supports and is asked for nothing else, so a
// provider that knows one site is written as the whole of that site's answer
// and nothing has to be taught about it elsewhere.
//
// What every provider does is here: say which addresses are its own, say what
// it downloads by, and hand back what the address publishes as words.
type provider interface {
	// CanHandle says whether this provider answers for an address. Each is asked
	// in turn, and the first that says so is the one that answers.
	CanHandle(at domain.URL) bool

	GetDownloadModel(at domain.URL) port.DownloadModel
	GetMetadata(ctx context.Context, at domain.URL) (port.Metadata, error)
	GetText(ctx context.Context, at domain.URL, want port.PreferredCaptions) (port.Text, error)
}

// A player is a provider that also reaches the bytes a person plays. A site
// publishing prose is not one, and says so by not being one rather than by
// answering that it has nothing.
type player interface {
	Download(ctx context.Context, at domain.URL, into io.Writer) (port.Copy, error)
}

// A Downloader is the providers this machine holds, asked in order.
type Downloader struct{ providers []provider }

// New is a downloader for this machine.
//
// The providers stand in the order they are asked, narrowest first: a video is
// downloaded by the tool that knows the site, and a page is what an address is
// when nothing knows it better, which is why pages stand last and why every
// machine downloads one.
func New(_ context.Context, c Config) (*Downloader, error) {
	return &Downloader{providers: []provider{newYtDLP(c), newPages()}}, nil
}

// providerFor is the provider that answers for an address, and ErrNoTool where
// this machine holds none.
func (f *Downloader) providerFor(at domain.URL) (provider, error) {
	for _, one := range f.providers {
		if one.CanHandle(at) {
			return one, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", string(at), ErrNoTool)
}

// GetDownloadModel is what this address is downloaded by, and nothing where
// nothing reaches it.
func (f *Downloader) GetDownloadModel(at domain.URL) port.DownloadModel {
	by, err := f.providerFor(at)
	if err != nil {
		return port.DownloadModel{}
	}
	return by.GetDownloadModel(at)
}

// GetMetadata is what stands at the address, taking none of it.
func (f *Downloader) GetMetadata(ctx context.Context, at domain.URL) (port.Metadata, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return port.Metadata{}, err
	}
	return by.GetMetadata(ctx, at)
}

// GetText is what the address publishes as words, as whichever provider answers
// for it produces them.
func (f *Downloader) GetText(
	ctx context.Context, at domain.URL, want port.PreferredCaptions,
) (port.Text, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return port.Text{}, err
	}
	return by.GetText(ctx, at, want)
}

// Download is what is at the address as a person plays it.
func (f *Downloader) Download(
	ctx context.Context, at domain.URL, into io.Writer,
) (port.Copy, error) {
	plays, err := f.playerFor(at)
	if err != nil {
		return port.Copy{}, err
	}
	return plays.Download(ctx, at, into)
}

// playerFor is the provider that reaches the bytes at an address, and
// ErrNothingDownloaded where whatever answers for it reaches only words.
func (f *Downloader) playerFor(at domain.URL) (player, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return nil, err
	}
	plays, is := by.(player)
	if !is {
		return nil, port.ErrNothingDownloaded
	}
	return plays, nil
}

// A program is a tool as it is started: the command, and the environment the
// settings name for it. Every run goes through buildCommand, so what a setting
// says about the environment cannot be forgotten at one of them.
type program struct {
	command []string
	env     []string
}

// isPresent says this machine has the tool.
func (p program) isPresent() bool { return len(p.command) > 0 }

// getPath is where the tool itself is, for another tool that runs it. A tool
// started through something else is not somewhere one path names.
func (p program) getPath() string {
	if len(p.command) != 1 {
		return ""
	}
	return p.command[0]
}

// buildCommand is one run of it, with the arguments of that run after its own.
func (p program) buildCommand(ctx context.Context, arguments ...string) *exec.Cmd {
	running := exec.CommandContext(ctx, p.command[0],
		append(append([]string(nil), p.command[1:]...), arguments...)...)
	running.Env = p.env
	setProcessGroup(running)
	return running
}

// stopGrace is how long a cancelled run has to stop on its own before its
// process group is killed, and how long Run waits on a pipe a child still holds.
const stopGrace = 5 * time.Second

// resolveProgram is the program to run, and nothing where this machine has no
// such tool. A named command is taken as it stands: a machine that writes the
// path afresh at every build names whatever does know where the tool is.
func resolveProgram(named Tool, tool string) program {
	if len(named.Command) > 0 {
		return program{
			command: append(append([]string(nil), named.Command...), named.Arguments...),
			env:     named.getEnvironment(),
		}
	}
	found, err := exec.LookPath(tool)
	if err != nil {
		return program{}
	}
	return program{command: append([]string{found}, named.Arguments...), env: named.getEnvironment()}
}

// run is one tool, waited for. What it wrote to its error stream is what a
// person is shown when it failed: the tool knows why, and nothing here is going
// to say it better.
func run(ctx context.Context, tool program, into io.Writer, arguments ...string) ([]byte, error) {
	running := tool.buildCommand(ctx, arguments...)
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
		return nil, fmt.Errorf("%w: %s", err, getLastLine(said.String()))
	}
	return out.Bytes(), nil
}

// getLastLine is what the tool said, as one line a person reads. A tool that
// refuses says why at length, and the last of it is the reason.
func getLastLine(said string) string {
	lines := strings.Split(strings.TrimSpace(said), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return "it said nothing"
}
