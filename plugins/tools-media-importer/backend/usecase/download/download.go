package download

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

// Downloader is the providers installed on this machine, tried in order.
type Downloader struct {
	providers []Provider
}

// New is a downloader for this machine: yt-dlp on the video sites, and every other page read as an
// article.
func New(c fetch.Config, runner Runner, files Files, paths Paths, web Web) *Downloader {
	return &Downloader{providers: []Provider{newYtDLP(c, runner, files, paths), NewPages(web)}}
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
		return nil, fetch.ErrNothingDownloaded
	}
	return plays, nil
}

// GetDownloadModel returns tool identity for the address.
func (f *Downloader) GetDownloadModel(address string) fetch.Model {
	delta, err := f.providerFor(address)
	if err != nil {
		return fetch.Model{Tool: "", Version: "", Producer: ""}
	}
	return delta.GetDownloadModel(address)
}

func (f *Downloader) GetMetadata(ctx context.Context, address string) (fetch.Metadata, error) {
	delta, err := f.providerFor(address)
	if err != nil {
		return fetch.Metadata{}, err
	}
	return delta.GetMetadata(ctx, address)
}

// GetText returns published words or cues.
func (f *Downloader) GetText(ctx context.Context, address string, want fetch.PreferredCaptions) (fetch.Text, error) {
	delta, err := f.providerFor(address)
	if err != nil {
		return fetch.Text{}, err
	}
	return delta.GetText(ctx, address, want)
}

// Download brings media bytes into the writer.
func (f *Downloader) Download(ctx context.Context, address string, media fetch.Media, into io.Writer) (fetch.Copy, error) {
	plays, err := f.playerFor(address)
	if err != nil {
		return fetch.Copy{}, err
	}
	return plays.Download(ctx, address, media, into)
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

func resolveProgram(named fetch.Tool, tool string, runner Runner) program {
	var baseEnv []string
	if len(named.Environment) > 0 && runner != nil {
		baseEnv = runner.Environ()
	}
	if len(named.Command) > 0 {
		return program{
			command: append(append([]string(nil), named.Command...), named.Arguments...),
			env:     getEnvironment(named, baseEnv),
		}
	}
	if runner == nil {
		return program{command: nil, env: nil}
	}
	found, err := runner.LookPath(tool)
	if err != nil {
		// gomutants:disable-next-line RETURN_ZERO reason="struct literal with nil fields equals zero value"
		return program{command: nil, env: nil}
	}
	return program{command: append([]string{found}, named.Arguments...), env: getEnvironment(named, baseEnv)}
}

func run(ctx context.Context, runner Runner, tool program, into io.Writer, arguments ...string) ([]byte, error) {
	out, combined, err := runner.Run(ctx, tool.command, tool.env, into, arguments...)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %s", err, getLastLine(combined))
	}
	return out, nil
}

func getLastLine(response string) string {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	for _, line := range slices.Backward(lines) {
		if line := strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return "it said nothing"
}

// CanCopy reports whether the provider of the address copies what is published there.
func (f *Downloader) CanCopy(address string) bool {
	_, err := f.playerFor(address)
	return err == nil
}

// Tools are the ports a downloader works through.
type Tools struct {
	Runner Runner
	Files  Files
	Paths  Paths
	Web    Web
}

// Open returns the downloader that fetches with programs, or the mock downloader when isMock is set.
func Open(isMock bool, tools Tools, programs fetch.Config) *Downloader {
	if isMock {
		return NewMock(tools.Web)
	}
	return New(programs, tools.Runner, tools.Files, tools.Paths, tools.Web)
}
