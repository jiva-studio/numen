// Package fetch reaches an address and brings back what is at it, by running
// the tools a person already has: yt-dlp for what a site publishes as a video,
// and ffmpeg where sound has to be brought to what a transcriber opens.
//
// Neither tool is linked and neither is shipped. Each is named by a setting
// holding a command and what it is started through, and an empty one asks the
// path; a machine with neither has no fetcher, and what asks for one is
// answered that this build cannot do it.
package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// ErrNoTool is a machine without the tool that reaches an address, and is what
// says a build cannot fetch at all.
var ErrNoTool = errors.New("yt-dlp is not on this machine")

// A Fetcher reaches addresses with the two commands, resolved once.
type Fetcher struct {
	video   []string
	sound   []string
	version string
	pages   *pages
}

// New is a fetcher for this machine, and ErrNoTool where it holds no yt-dlp.
//
// yt-dlp is what reaches an address; ffmpeg only brings what it took to the
// container a transcriber opens. A machine with ffmpeg alone can fetch nothing,
// and a fetcher that can fetch nothing is one nobody wants offered.
func New(ctx context.Context, c Config) (*Fetcher, error) {
	video := resolved(c.Video, "yt-dlp")
	if video == nil {
		return nil, ErrNoTool
	}
	sound := resolved(c.Sound, "ffmpeg")
	return &Fetcher{video: video, sound: sound, version: version(ctx, video), pages: newPages()}, nil
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

// version is what the tool answers when asked which it is. A tool that will not
// say is still a tool, and what it produced is claimed by its name alone.
func version(ctx context.Context, video []string) string {
	if video == nil {
		return ""
	}
	said, err := run(ctx, video, nil, "--version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(said))
}

// Fetching is what this address is fetched by: a video by the tool this machine
// holds, a page by the reader in this process.
func (f *Fetcher) Fetching(at domain.WebAddress) port.FetchModel {
	if !at.IsVideo() {
		return port.FetchModel{Tool: readerName}
	}
	return port.FetchModel{Tool: "yt-dlp", Version: f.version}
}

// Metadata is what stands at the address, taking none of it.
func (f *Fetcher) Metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
	if !at.IsVideo() {
		return f.pages.metadata(ctx, at)
	}
	if f.video == nil {
		return port.Metadata{}, ErrNoTool
	}
	said, err := run(ctx, f.video, nil, "--dump-single-json", "--no-playlist", at.URL)
	if err != nil {
		return port.Metadata{}, err
	}
	var held struct {
		Title              string                `json:"title"`
		Language           string                `json:"language"`
		Duration           float64               `json:"duration"`
		Filesize           int64                 `json:"filesize"`
		Approximate        int64                 `json:"filesize_approx"`
		Subtitles          map[string][]struct{} `json:"subtitles"`
		AutomaticCaptions  map[string][]struct{} `json:"automatic_captions"`
		RequestedSubtitles map[string][]struct{} `json:"requested_subtitles"`
	}
	if err := json.Unmarshal(said, &held); err != nil {
		return port.Metadata{}, fmt.Errorf("what yt-dlp said about %s: %w", at.URL, err)
	}
	found := port.Metadata{
		Title:    strings.TrimSpace(held.Title),
		Length:   int(held.Duration * 1000),
		Language: held.Language,
		Bytes:    max(held.Filesize, held.Approximate),
	}
	found.Captions = languages(held.Subtitles)
	found.Automatic = languages(held.AutomaticCaptions)
	return found, nil
}

// languages are the languages of one set of tracks, in one order however the tool
// listed them.
func languages(tracks map[string][]struct{}) []string {
	out := make([]string, 0, len(tracks))
	for language := range tracks {
		out = append(out, language)
	}
	sort.Strings(out)
	return out
}

// Subtitles are the words published with a video.
//
// They are asked for as the format that carries one stretch of speech to a
// cue. What a site draws as two lines scrolling is one stretch said once, and
// asking for the format a player is fed would put every line into the index
// twice.
func (f *Fetcher) Subtitles(
	ctx context.Context, at domain.WebAddress, language string,
) ([]transcript.Cue, error) {
	if !at.IsVideo() || language == "" {
		return nil, port.ErrNothingFetched
	}
	if f.video == nil {
		return nil, ErrNoTool
	}
	into, err := os.MkdirTemp("", "numen-words-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(into) }()

	if _, err := run(ctx, f.video, nil,
		"--skip-download", "--write-subs", "--write-auto-subs",
		"--sub-langs", language, "--sub-format", "json3",
		"--no-playlist", "-o", filepath.Join(into, "words"), at.URL,
	); err != nil {
		return nil, err
	}
	found, err := filepath.Glob(filepath.Join(into, "words.*.json3"))
	if err != nil || len(found) == 0 {
		return nil, port.ErrNothingFetched
	}
	raw, err := os.ReadFile(found[0])
	if err != nil {
		return nil, err
	}
	cues, err := cued(raw)
	if err != nil {
		return nil, err
	}
	if len(cues) == 0 {
		return nil, port.ErrNothingFetched
	}
	return cues, nil
}

// Audio is a video's sound as the container a transcriber opens: one channel at
// 16 kHz, which is what a model takes.
func (f *Fetcher) Audio(ctx context.Context, at domain.WebAddress, into io.Writer) error {
	if f.video == nil || f.sound == nil {
		return ErrNoTool
	}
	taking := exec.CommandContext(ctx, f.video[0],
		append(append([]string(nil), f.video[1:]...),
			"-f", "bestaudio", "--no-playlist", "-o", "-", at.URL)...)
	bringing := exec.CommandContext(ctx, f.sound[0],
		append(append([]string(nil), f.sound[1:]...),
			"-hide_banner", "-loglevel", "error", "-i", "pipe:0",
			"-vn", "-ac", "1", "-ar", "16000", "-f", "wav", "pipe:1")...)

	sound, err := taking.StdoutPipe()
	if err != nil {
		return err
	}
	bringing.Stdin = sound
	bringing.Stdout = into
	var said, saidToo bytes.Buffer
	taking.Stderr, bringing.Stderr = &said, &saidToo

	if err := taking.Start(); err != nil {
		return err
	}
	if err := bringing.Start(); err != nil {
		_ = taking.Process.Kill()
		_ = taking.Wait()
		return err
	}
	if err := bringing.Wait(); err != nil {
		_ = taking.Process.Kill()
		_ = taking.Wait()
		return fmt.Errorf("%w: %s", err, lastLine(saidToo.String()))
	}
	if err := taking.Wait(); err != nil {
		return fmt.Errorf("%w: %s", err, lastLine(said.String()))
	}
	return nil
}

// Download is the video as a person plays it, in the one container every player
// this window is drawn in opens.
func (f *Fetcher) Download(
	ctx context.Context, at domain.WebAddress, into io.Writer,
) (port.Download, error) {
	if !at.IsVideo() {
		return port.Download{}, port.ErrNothingFetched
	}
	if f.video == nil {
		return port.Download{}, ErrNoTool
	}
	// The one container is asked for by name. A site with nothing in it says so,
	// and what arrives is what the copy is served as.
	taking := exec.CommandContext(ctx, f.video[0],
		append(append([]string(nil), f.video[1:]...),
			"-f", "best[ext=mp4]/mp4", "--no-playlist", "-o", "-", at.URL)...)
	var said bytes.Buffer
	taking.Stdout, taking.Stderr = into, &said
	if err := taking.Run(); err != nil {
		if stopped := ctx.Err(); stopped != nil {
			return port.Download{}, stopped
		}
		return port.Download{}, fmt.Errorf("%w: %s", err, lastLine(said.String()))
	}
	return port.Download{MediaType: "video/mp4", Extension: ".mp4"}, nil
}

// Article is what an address that plays nothing says.
func (f *Fetcher) Article(ctx context.Context, at domain.WebAddress) (port.Article, error) {
	return f.pages.article(ctx, at)
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
