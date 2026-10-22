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
	"sync"
	"time"
)

type ytDLP struct {
	command      program
	sound        program
	getVersion   func() string
	probeTimeout time.Duration
	infoMu       sync.Mutex
	infoCache    map[string]string
}

const defaultProbeTimeout = 2 * time.Minute

func newYtDLP(c Config) *ytDLP {
	command := resolveProgram(c.Video, "yt-dlp")
	v := &ytDLP{
		command:      command,
		sound:        resolveProgram(c.Sound, "ffmpeg"),
		probeTimeout: defaultProbeTimeout,
		infoCache:    make(map[string]string),
	}
	v.getVersion = sync.OnceValue(v.readVersion)
	return v
}

func (v *ytDLP) CanHandle(at string) bool { return isVideoSite(at) && v.command.isPresent() }

func (v *ytDLP) GetDownloadModel(string) DownloadModel {
	return DownloadModel{Tool: "yt-dlp", Version: v.getVersion(), Producer: "captions"}
}

func (v *ytDLP) readVersion() string {
	if !v.command.isPresent() {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), v.probeTimeout)
	defer cancel()
	output, err := run(ctx, v.command, nil, "--version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (v *ytDLP) readInfo(ctx context.Context, pos string) ([]byte, error) {
	v.infoMu.Lock()
	if cachedPath, ok := v.infoCache[pos]; ok {
		v.infoMu.Unlock()
		if raw, err := os.ReadFile(cachedPath); err == nil {
			return raw, nil
		}
	} else {
		v.infoMu.Unlock()
	}

	probing, cancel := context.WithTimeout(ctx, v.probeTimeout)
	defer cancel()
	output, err := run(probing, v.command, nil, "--dump-single-json", "--no-playlist", pos)
	if err != nil {
		return nil, err
	}

	if tempFile, err := os.CreateTemp("", "numen-info-*.json"); err == nil {
		if _, writeErr := tempFile.Write(output); writeErr == nil {
			_ = tempFile.Close()
			v.infoMu.Lock()
			v.infoCache[pos] = tempFile.Name()
			v.infoMu.Unlock()
		} else {
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
		}
	}

	return output, nil
}

func (v *ytDLP) getTarget(pos string) []string {
	v.infoMu.Lock()
	saved, ok := v.infoCache[pos]
	v.infoMu.Unlock()
	if ok {
		if _, err := os.Stat(saved); err == nil {
			return []string{"--load-info-json", saved}
		}
	}
	return []string{pos}
}

func (v *ytDLP) GetMetadata(ctx context.Context, at string) (Metadata, error) {
	output, err := v.readInfo(ctx, at)
	if err != nil {
		return Metadata{}, err
	}
	var metadata struct {
		Title             string                `json:"title"`
		Language          string                `json:"language"`
		Duration          float64               `json:"duration"`
		Filesize          int64                 `json:"filesize"`
		Approximate       int64                 `json:"filesize_approx"`
		Subtitles         map[string][]struct{} `json:"subtitles"`
		AutomaticCaptions map[string][]struct{} `json:"automatic_captions"`
	}
	if err := json.Unmarshal(output, &metadata); err != nil {
		return Metadata{}, fmt.Errorf("what yt-dlp output about %s: %w", at, err)
	}
	return Metadata{
		Title:       strings.TrimSpace(metadata.Title),
		Length:      int(metadata.Duration * 1000),
		Language:    metadata.Language,
		Bytes:       max(metadata.Filesize, metadata.Approximate),
		Captions:    getLanguages(metadata.Subtitles),
		IsAutomatic: getLanguages(metadata.AutomaticCaptions),
	}, nil
}

func getLanguages(tracks map[string][]struct{}) []string {
	out := make([]string, 0, len(tracks))
	for language := range tracks {
		out = append(out, language)
	}
	sort.Strings(out)
	return out
}

func (v *ytDLP) GetText(ctx context.Context, at string, want PreferredCaptions) (Text, error) {
	meta, err := v.GetMetadata(ctx, at)
	if err != nil {
		return Text{}, err
	}
	cues, err := v.subtitles(ctx, at, chooseLanguage(meta, want.Languages, want.IsAutomatic))
	if err != nil {
		return Text{}, err
	}
	return Text{
		Producer: "captions",
		Cues:     cues,
		Title:    meta.Title,
		Length:   meta.Length,
	}, nil
}

func (v *ytDLP) subtitles(ctx context.Context, pos string, language string) ([]Cue, error) {
	if language == "" {
		return nil, ErrNothingDownloaded
	}
	into, err := os.MkdirTemp("", "numen-words-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(into) }()

	arguments := append([]string{
		"--skip-download", "--write-subs", "--write-auto-subs",
		"--sub-langs", language, "--sub-format", "json3",
		"--no-playlist", "-o", filepath.Join(into, "words"),
	}, v.getTarget(pos)...)
	if _, err := run(ctx, v.command, nil, arguments...); err != nil {
		return nil, err
	}
	found, err := filepath.Glob(filepath.Join(into, "words.*.json3"))
	if err != nil || len(found) == 0 {
		return nil, ErrNothingDownloaded
	}
	raw, err := os.ReadFile(found[0])
	if err != nil {
		return nil, err
	}
	cues, err := parseCues(raw)
	if err != nil {
		return nil, err
	}
	if len(cues) == 0 {
		return nil, ErrNothingDownloaded
	}
	return cues, nil
}

const copyFormat = "bestvideo[vcodec^=avc1]+bestaudio[acodec^=mp4a]/" +
	"best[vcodec!=none][acodec!=none]/bestvideo*+bestaudio/best"

func (v *ytDLP) Download(ctx context.Context, at string, into io.Writer) (Copy, error) {
	folder, err := os.MkdirTemp("", "numen-copy-")
	if err != nil {
		return Copy{}, err
	}
	defer os.RemoveAll(folder)

	arguments := []string{
		"-f", copyFormat, "--merge-output-format", "mp4", "--no-playlist",
		"-o", filepath.Join(folder, "copy.%(ext)s"),
	}
	arguments = append(arguments, v.getTarget(at)...)
	if where := v.sound.getPath(); where != "" {
		arguments = append([]string{"--ffmpeg-location", where}, arguments...)
	}
	taking := v.command.buildCommand(ctx, arguments...)
	var logBuffer bytes.Buffer
	taking.Stdout, taking.Stderr = &logBuffer, &logBuffer
	if err := taking.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Copy{}, ctxErr
		}
		return Copy{}, fmt.Errorf("%w: %s", err, getLastLine(logBuffer.String()))
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return Copy{}, err
	}
	if len(entries) != 1 {
		return Copy{}, fmt.Errorf("%w: %s", ErrNothingDownloaded, getLastLine(logBuffer.String()))
	}
	name := filepath.Join(folder, entries[0].Name())

	copied, err := os.Open(name)
	if err != nil {
		return Copy{}, err
	}
	defer copied.Close()
	if _, err := io.Copy(into, copied); err != nil {
		return Copy{}, err
	}
	extension := filepath.Ext(name)
	return Copy{MediaType: getMediaType(name), Extension: extension}, nil
}

var mediaTypes = map[string]string{
	".aac":  "audio/aac",
	".aif":  "audio/aiff",
	".aiff": "audio/aiff",
	".alac": "audio/alac",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".m4b":  "audio/mp4",
	".m4p":  "audio/mp4",
	".mp3":  "audio/mpeg",
	".mp4":  "video/mp4",
	".oga":  "audio/ogg",
	".ogg":  "audio/ogg",
	".opus": "audio/opus",
	".wav":  "audio/wav",
	".weba": "audio/webm",
	".webm": "video/webm",
}

func getMediaType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if mt, ok := mediaTypes[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}

func chooseLanguage(meta Metadata, languages []string, automatic bool) string {
	tracks := meta.Captions
	if len(tracks) == 0 && automatic {
		tracks = meta.IsAutomatic
	}
	if len(tracks) == 0 {
		return ""
	}
	for _, wanted := range append(append([]string(nil), languages...), meta.Language) {
		if one := slices.IndexFunc(tracks, matchLanguage(wanted)); one >= 0 {
			return tracks[one]
		}
	}
	if one := slices.IndexFunc(tracks, isOriginalTrack); one >= 0 {
		return tracks[one]
	}
	return tracks[0]
}

func matchLanguage(language string) func(string) bool {
	return func(track string) bool { return track == language || track == language+"-orig" }
}

func isOriginalTrack(track string) bool { return strings.HasSuffix(track, "-orig") }

var videoSites = map[string]bool{
	"youtube.com":          true,
	"m.youtube.com":        true,
	"music.youtube.com":    true,
	"youtube-nocookie.com": true,
	"youtu.be":             true,
}

func isVideoSite(pos string) bool {
	address, err := url.Parse(pos)
	if err != nil {
		return false
	}
	return videoSites[strings.TrimPrefix(address.Hostname(), "www.")]
}
