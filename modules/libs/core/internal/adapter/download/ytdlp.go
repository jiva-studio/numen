package download

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
func (v *ytDLP) Supports(at domain.URL) bool { return carries(at) && v.command.held() }

func (v *ytDLP) Downloading(domain.URL) port.DownloadModel {
	return port.DownloadModel{Tool: "yt-dlp", Version: v.version, Producer: text.Captions}
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
func (v *ytDLP) Metadata(ctx context.Context, at domain.URL) (port.Metadata, error) {
	said, err := run(ctx, v.command, nil, "--dump-single-json", "--no-playlist", string(at))
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
		return port.Metadata{}, fmt.Errorf("what yt-dlp said about %s: %w", string(at), err)
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

// Text is the words published with a video, against the times they were said
// at, beside what it calls itself and how long it runs.
//
// They are asked for as the format that carries one stretch of speech to a cue.
// What a site draws as two lines scrolling is one stretch said once, and asking
// for the format a player is fed would put every line into the index twice.
func (v *ytDLP) Text(
	ctx context.Context, at domain.URL, want port.PreferredCaptions,
) (port.Text, error) {
	meta, err := v.Metadata(ctx, at)
	if err != nil {
		return port.Text{}, err
	}
	cues, err := v.subtitles(ctx, at, language(meta, want.Languages, want.Automatic))
	if err != nil {
		return port.Text{}, err
	}
	return port.Text{
		Producer: text.Captions,
		Cues:     cues,
		Title:    meta.Title,
		Length:   meta.Length,
	}, nil
}

// subtitles are the words published in one language, as the site names that
// language.
func (v *ytDLP) subtitles(
	ctx context.Context, at domain.URL, language string,
) ([]transcript.Cue, error) {
	if language == "" {
		return nil, port.ErrNothingDownloaded
	}
	into, err := os.MkdirTemp("", "numen-words-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(into) }()

	if _, err := run(ctx, v.command, nil,
		"--skip-download", "--write-subs", "--write-auto-subs",
		"--sub-langs", language, "--sub-format", "json3",
		"--no-playlist", "-o", filepath.Join(into, "words"), string(at),
	); err != nil {
		return nil, err
	}
	found, err := filepath.Glob(filepath.Join(into, "words.*.json3"))
	if err != nil || len(found) == 0 {
		return nil, port.ErrNothingDownloaded
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
		return nil, port.ErrNothingDownloaded
	}
	return cues, nil
}

// Download is what is at the address as a person plays it, in the one container
// every player this window is drawn in opens.
//
// Picture and sound are asked for separately and put in one file: a site that
// serves them apart has no single stream to take, and one that serves them
// together is taken as it stands. Joining them is a file's work, so the copy
// lands beside this run before it is handed on.
func (v *ytDLP) Download(
	ctx context.Context, at domain.URL, into io.Writer,
) (port.Copy, error) {
	folder, err := os.MkdirTemp("", "numen-copy-")
	if err != nil {
		return port.Copy{}, err
	}
	defer os.RemoveAll(folder)

	arguments := []string{
		"-f", copyFormat, "--merge-output-format", "mp4", "--no-playlist",
		"-o", filepath.Join(folder, "copy.%(ext)s"), string(at),
	}
	if where := v.sound.at(); where != "" {
		arguments = append([]string{"--ffmpeg-location", where}, arguments...)
	}
	taking := v.command.started(ctx, arguments...)
	var said bytes.Buffer
	taking.Stdout, taking.Stderr = &said, &said
	if err := taking.Run(); err != nil {
		if stopped := ctx.Err(); stopped != nil {
			return port.Copy{}, stopped
		}
		return port.Copy{}, fmt.Errorf("%w: %s", err, lastLine(said.String()))
	}

	// What the container ended up being is read off the folder: codecs that mp4
	// cannot hold are written to a container that can, and the name says which.
	written, err := os.ReadDir(folder)
	if err != nil {
		return port.Copy{}, err
	}
	if len(written) != 1 {
		return port.Copy{}, fmt.Errorf("%w: %s", port.ErrNothingDownloaded, lastLine(said.String()))
	}
	name := filepath.Join(folder, written[0].Name())

	copied, err := os.Open(name)
	if err != nil {
		return port.Copy{}, err
	}
	defer copied.Close()
	if _, err := io.Copy(into, copied); err != nil {
		return port.Copy{}, err
	}
	extension := filepath.Ext(name)
	return port.Copy{MediaType: domain.MediaType(name), Extension: extension}, nil
}

// copyFormat is what a copy is asked for: picture and sound in one file, in the
// codecs a player opens where the site has them, and in whatever it has where
// it does not. A stream carrying both is taken whole; one carrying picture
// alone is never the answer, because a copy nobody can hear is not a copy.
const copyFormat = "bestvideo[vcodec^=avc1]+bestaudio[acodec^=mp4a]/" +
	"best[vcodec!=none][acodec!=none]/bestvideo*+bestaudio/best"

// language is the one the words are asked for in.
//
// What a person published is preferred over what a machine wrote; among those,
// the languages this installation named, and then the language it was spoken
// in. Something translated into thirty languages publishes words in all thirty,
// and what was said in it is one of them.
func language(meta port.Metadata, languages []string, automatic bool) string {
	tracks := meta.Captions
	if len(tracks) == 0 && automatic {
		tracks = meta.Automatic
	}
	if len(tracks) == 0 {
		return ""
	}
	for _, wanted := range append(append([]string(nil), languages...), meta.Language) {
		if one := slices.IndexFunc(tracks, in(wanted)); one >= 0 {
			return tracks[one]
		}
	}
	if one := slices.IndexFunc(tracks, original); one >= 0 {
		return tracks[one]
	}
	return tracks[0]
}

// in says whether a track is in one language. A machine's own is that language
// with a word after it, and its translations of that one are other languages.
func in(language string) func(string) bool {
	return func(track string) bool { return track == language || track == language+"-orig" }
}

// original says whether a track is the language it was spoken in.
func original(track string) bool { return strings.HasSuffix(track, "-orig") }

// videoSites are the hosts this provider answers for. yt-dlp knows far more
// than these; what is listed is what this application hands it rather than
// reading as a page, and adding a site is adding a line.
//
// A host is written without `www.`, which is trimmed before the lookup.
var videoSites = map[string]bool{
	"youtube.com":          true,
	"m.youtube.com":        true,
	"music.youtube.com":    true,
	"youtube-nocookie.com": true,
	"youtu.be":             true,
}

// carries says whether this provider answers for an address, which is whether
// the site is one of its own and the address names something there.
func carries(at domain.URL) bool {
	address, err := url.Parse(string(at))
	if err != nil {
		return false
	}
	return videoSites[strings.TrimPrefix(address.Hostname(), "www.")]
}
