package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Downloader is the providers this machine holds, asked in order.
type Downloader struct {
	providers []Provider
}

// New is a downloader for this machine.
func New(_ context.Context, c Config) (*Downloader, error) {
	return &Downloader{providers: []Provider{newYtDLP(c), NewPages()}}, nil
}

// CanHandle reports whether any provider handles the address.
func (f *Downloader) CanHandle(at string) bool {
	for _, one := range f.providers {
		if one.CanHandle(at) {
			return true
		}
	}
	return false
}

func (f *Downloader) providerFor(at string) (Provider, error) {
	for _, one := range f.providers {
		if one.CanHandle(at) {
			return one, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", at, ErrNoTool)
}

func (f *Downloader) playerFor(at string) (Player, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return nil, err
	}
	plays, is := by.(Player)
	if !is {
		return nil, ErrNothingDownloaded
	}
	return plays, nil
}

// GetDownloadModel returns tool identity for the address.
func (f *Downloader) GetDownloadModel(at string) DownloadModel {
	by, err := f.providerFor(at)
	if err != nil {
		return DownloadModel{}
	}
	return by.GetDownloadModel(at)
}

// GetMetadata returns media metadata.
func (f *Downloader) GetMetadata(ctx context.Context, at string) (Metadata, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return Metadata{}, err
	}
	return by.GetMetadata(ctx, at)
}

// GetText returns published words or cues.
func (f *Downloader) GetText(ctx context.Context, at string, want PreferredCaptions) (Text, error) {
	by, err := f.providerFor(at)
	if err != nil {
		return Text{}, err
	}
	return by.GetText(ctx, at, want)
}

// Download brings media bytes into the writer.
func (f *Downloader) Download(ctx context.Context, at string, into io.Writer) (Copy, error) {
	plays, err := f.playerFor(at)
	if err != nil {
		return Copy{}, err
	}
	return plays.Download(ctx, at, into)
}

type program struct {
	command []string
	env     []string
}

func (p program) isPresent() bool { return len(p.command) > 0 }

func (p program) getPath() string {
	if len(p.command) != 1 {
		return ""
	}
	return p.command[0]
}

func (p program) buildCommand(ctx context.Context, arguments ...string) *exec.Cmd {
	running := exec.CommandContext(ctx, p.command[0],
		append(append([]string(nil), p.command[1:]...), arguments...)...)
	running.Env = p.env
	setProcessGroup(running)
	return running
}

const stopGrace = 5 * time.Second

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

func run(ctx context.Context, tool program, into io.Writer, arguments ...string) ([]byte, error) {
	running := tool.buildCommand(ctx, arguments...)
	var out, said bytes.Buffer
	running.Stdout, running.Stderr = &out, &said
	if into != nil {
		running.Stdout = into
	}
	if err := running.Run(); err != nil {
		if stopped := ctx.Err(); stopped != nil {
			return nil, stopped
		}
		return nil, fmt.Errorf("%w: %s", err, getLastLine(said.String()))
	}
	return out.Bytes(), nil
}

func getLastLine(said string) string {
	lines := strings.Split(strings.TrimSpace(said), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return "it said nothing"
}
