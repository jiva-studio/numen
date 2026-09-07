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

// videos is the provider for what a site publishes as a video, which it fetches
// by running yt-dlp. The sites it supports are the sites that tool knows, and a
// machine without it supports none.
//
// ffmpeg is beside it because a video's sound arrives in whatever container the
// site had and a transcriber opens one.
type videos struct {
	command program
	sound   program
	version string
}

func newVideos(ctx context.Context, c Config) *videos {
	command := resolved(c.Video, "yt-dlp")
	return &videos{
		command: command,
		sound:   resolved(c.Sound, "ffmpeg"),
		version: version(ctx, command),
	}
}

// Supports is a video, on a machine holding the tool that gets at one.
func (v *videos) Supports(at domain.WebAddress) bool { return at.IsVideo() && v.command.held() }

func (v *videos) Fetching(domain.WebAddress) port.FetchModel {
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
func (v *videos) Metadata(ctx context.Context, at domain.WebAddress) (port.Metadata, error) {
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
func (v *videos) Subtitles(
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
func (v *videos) Audio(ctx context.Context, at domain.WebAddress, into io.Writer) error {
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

// Download is the video as a person plays it, in the one container every player
// this window is drawn in opens.
func (v *videos) Download(
	ctx context.Context, at domain.WebAddress, into io.Writer,
) (port.Download, error) {
	// The one container is asked for by name. A site with nothing in it says so,
	// and what arrives is what the copy is served as.
	taking := v.command.started(ctx, "-f", "best[ext=mp4]/mp4", "--no-playlist", "-o", "-", at.URL)
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

// Article is a page's prose, which a video is not.
func (v *videos) Article(context.Context, domain.WebAddress) (port.Article, error) {
	return port.Article{}, port.ErrNothingFetched
}
