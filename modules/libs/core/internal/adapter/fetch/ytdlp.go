package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// ytDLP is the provider that fetches by running yt-dlp. The sites it supports
// are the sites that tool knows, and a machine without it supports none.
//
// ffmpeg is beside it because sound arrives in whatever container the site had
// and a transcriber opens one, and because picture and sound served apart are
// one file only once they are joined.
type ytDLP struct {
	command program
	sound   program
	version string
}

func newYtDLP(ctx context.Context, c Config) *ytDLP {
	command := resolved(c.Video, "yt-dlp")
	return &ytDLP{
		command: command,
		sound:   resolved(c.Sound, "ffmpeg"),
		version: version(ctx, command),
	}
}

// Supports is a video, on a machine holding the tool that gets at one.
func (v *ytDLP) Supports(at domain.WebAddress) bool { return at.IsVideo() && v.command.held() }

func (v *ytDLP) Fetching(domain.WebAddress) port.FetchModel {
	return port.FetchModel{Tool: "yt-dlp", Version: v.version}
}

// version is what the tool answers when asked which it is. A tool that will not
// say is still a tool, and what it produced is claimed by its name alone.
func version(ctx context.Context, tool program) string {
	if !tool.held() {
		return ""
	}
	said, err := run(ctx, tool, nil, "--version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(said))
}

// Metadata is what the site says about the video, taking none of it.
func (v *ytDLP) Metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
	said, err := run(ctx, v.command, nil, "--dump-single-json", "--no-playlist", at.URL)
	if err != nil {
		return port.Metadata{}, err
	}
	var held struct {
		Title             string                `json:"title"`
		Language          string                `json:"language"`
		Duration          float64               `json:"duration"`
		Filesize          int64                 `json:"filesize"`
		Approximate       int64                 `json:"filesize_approx"`
		Subtitles         map[string][]struct{} `json:"subtitles"`
		AutomaticCaptions map[string][]struct{} `json:"automatic_captions"`
	}
	if err := json.Unmarshal(said, &held); err != nil {
		return port.Metadata{}, fmt.Errorf("what yt-dlp said about %s: %w", at.URL, err)
	}
	return port.Metadata{
		Title:     strings.TrimSpace(held.Title),
		Length:    int(held.Duration * 1000),
		Language:  held.Language,
		Bytes:     max(held.Filesize, held.Approximate),
		Captions:  languages(held.Subtitles),
		Automatic: languages(held.AutomaticCaptions),
	}, nil
}

// languages are the languages of one set of tracks, in one order however the
// tool listed them.
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
// They are asked for as the format that carries one stretch of speech to a cue.
// What a site draws as two lines scrolling is one stretch said once, and asking
// for the format a player is fed would put every line into the index twice.
func (v *ytDLP) Subtitles(
	ctx context.Context, at domain.WebAddress, language string,
) ([]transcript.Cue, error) {
	if language == "" {
		return nil, port.ErrNothingFetched
	}
	into, err := os.MkdirTemp("", "numen-words-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(into) }()

	if _, err := run(ctx, v.command, nil,
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
func (v *ytDLP) Audio(ctx context.Context, at domain.WebAddress, into io.Writer) error {
	if !v.sound.held() {
		return ErrNoTool
	}
	taking := v.command.started(ctx, "-f", "bestaudio", "--no-playlist", "-o", "-", at.URL)
	bringing := v.sound.started(ctx,
		"-hide_banner", "-loglevel", "error", "-i", "pipe:0",
		"-vn", "-ac", "1", "-ar", "16000", "-f", "wav", "pipe:1")

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

// Download is what is at the address as a person plays it, in the one container
// every player this window is drawn in opens.
//
// Picture and sound are asked for separately and put in one file: a site that
// serves them apart has no single stream to take, and one that serves them
// together is taken as it stands. Joining them is a file's work, so the copy
// lands beside this run before it is handed on.
func (v *ytDLP) Download(
	ctx context.Context, at domain.WebAddress, into io.Writer,
) (port.Download, error) {
	folder, err := os.MkdirTemp("", "numen-copy-")
	if err != nil {
		return port.Download{}, err
	}
	defer os.RemoveAll(folder)

	arguments := []string{
		"-f", copyFormat, "--merge-output-format", "mp4", "--no-playlist",
		"-o", filepath.Join(folder, "copy.%(ext)s"), at.URL,
	}
	if where := v.sound.at(); where != "" {
		arguments = append([]string{"--ffmpeg-location", where}, arguments...)
	}
	taking := v.command.started(ctx, arguments...)
	var said bytes.Buffer
	taking.Stdout, taking.Stderr = &said, &said
	if err := taking.Run(); err != nil {
		if stopped := ctx.Err(); stopped != nil {
			return port.Download{}, stopped
		}
		return port.Download{}, fmt.Errorf("%w: %s", err, lastLine(said.String()))
	}

	// What the container ended up being is read off the folder: codecs that mp4
	// cannot hold are written to a container that can, and the name says which.
	written, err := os.ReadDir(folder)
	if err != nil {
		return port.Download{}, err
	}
	if len(written) != 1 {
		return port.Download{}, fmt.Errorf("%w: %s", port.ErrNothingFetched, lastLine(said.String()))
	}
	name := filepath.Join(folder, written[0].Name())

	copied, err := os.Open(name)
	if err != nil {
		return port.Download{}, err
	}
	defer copied.Close()
	if _, err := io.Copy(into, copied); err != nil {
		return port.Download{}, err
	}
	extension := filepath.Ext(name)
	return port.Download{MediaType: domain.MediaType(name), Extension: extension}, nil
}

// copyFormat is what a copy is asked for: picture and sound in one file, in the
// codecs a player opens where the site has them, and in whatever it has where
// it does not. A stream carrying both is taken whole; one carrying picture
// alone is never the answer, because a copy nobody can hear is not a copy.
const copyFormat = "bestvideo[vcodec^=avc1]+bestaudio[acodec^=mp4a]/" +
	"best[vcodec!=none][acodec!=none]/bestvideo*+bestaudio/best"

// Article is a page's prose, which a video is not.
func (v *ytDLP) Article(context.Context, domain.WebAddress) (port.Article, error) {
	return port.Article{}, port.ErrNothingFetched
}
