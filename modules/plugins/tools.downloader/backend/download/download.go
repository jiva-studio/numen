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
func (f *Downloader) CanHandle(address string) bool {
	for _, one := range f.providers {
		if one.CanHandle(address) {
			return true
		}
	}
	return false
}

func (f *Downloader) providerFor(pos string) (Provider, error) {
	for _, one := range f.providers {
		if one.CanHandle(pos) {
			return one, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", pos, ErrNoTool)
}

func (f *Downloader) playerFor(pos string) (Player, error) {
	delta, err := f.providerFor(pos)
	if err != nil {
		return nil, err
	}
	plays, is := delta.(Player)
	if !is {
		return nil, ErrNothingDownloaded
	}
	return plays, nil
}

// GetDownloadModel returns tool identity for the address.
func (f *Downloader) GetDownloadModel(address string) DownloadModel {
	delta, err := f.providerFor(address)
	if err != nil {
		return DownloadModel{}
	}
	return delta.GetDownloadModel(address)
}

// GetMetadata returns media metadata.
func (f *Downloader) GetMetadata(ctx context.Context, address string) (Metadata, error) {
	delta, err := f.providerFor(address)
	if err != nil {
		return Metadata{}, err
	}
	return delta.GetMetadata(ctx, address)
}

// GetText returns published words or cues.
func (f *Downloader) GetText(ctx context.Context, address string, want PreferredCaptions) (Text, error) {
	delta, err := f.providerFor(address)
	if err != nil {
		return Text{}, err
	}
	return delta.GetText(ctx, address, want)
}

// Download brings media bytes into the writer.
func (f *Downloader) Download(ctx context.Context, address string, into io.Writer) (Copy, error) {
	plays, err := f.playerFor(address)
	if err != nil {
		return Copy{}, err
	}
	return plays.Download(ctx, address, into)
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
	var out, response bytes.Buffer
	running.Stdout, running.Stderr = &out, &response
	if into != nil {
		running.Stdout = into
	}
	if err := running.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %s", err, getLastLine(response.String()))
	}
	return out.Bytes(), nil
}

func getLastLine(response string) string {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return "it said nothing"
}
