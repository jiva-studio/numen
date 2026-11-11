package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
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
	runner       Runner
	files        Files
	getVersion   func() string
	probeTimeout time.Duration
	infoMu       sync.Mutex
	infoCache    map[string]string
}

const defaultProbeTimeout = 2 * time.Minute

func newYtDLP(c Config, runner Runner, files Files) *ytDLP {
	command := resolveProgram(c.Video, "yt-dlp", runner)
	v := &ytDLP{
		command:      command,
		sound:        resolveProgram(c.Sound, "ffmpeg", runner),
		runner:       runner,
		files:        files,
		getVersion:   nil,
		probeTimeout: defaultProbeTimeout,
		infoMu:       sync.Mutex{},
		infoCache:    make(map[string]string),
	}
	v.getVersion = sync.OnceValue(v.readVersion)
	return v
}

func (v *ytDLP) CanHandle(target string) bool { return isVideoSite(target) && v.command.isPresent() }

func (v *ytDLP) GetDownloadModel(string) Model {
	return Model{Tool: "yt-dlp", Version: v.getVersion(), Producer: producerCaptions}
}

func (v *ytDLP) readVersion() string {
	if !v.command.isPresent() {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), v.probeTimeout)
	defer cancel()
	output, err := run(ctx, v.runner, v.command, nil, "--version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (v *ytDLP) readInfo(ctx context.Context, pos string) ([]byte, error) {
	v.infoMu.Lock()
	if cachedPath, ok := v.infoCache[pos]; ok {
		v.infoMu.Unlock()
		if raw, err := v.files.ReadFile(cachedPath); err == nil {
			return raw, nil
		}
	} else {
		v.infoMu.Unlock()
	}

	probing, cancel := context.WithTimeout(ctx, v.probeTimeout)
	defer cancel()
	output, err := run(probing, v.runner, v.command, nil, "--dump-single-json", "--no-playlist", pos)
	if err != nil {
		return nil, err
	}

	if err := v.cacheInfo(pos, output); err != nil && v.runner != nil {
		v.runner.ReportError("keep the info of "+pos, err)
	}

	return output, nil
}

func (v *ytDLP) cacheInfo(pos string, output []byte) error {
	saved, err := v.files.WriteTemp("numen-info-*.json", output)
	if err != nil {
		return err
	}
	v.infoMu.Lock()
	v.infoCache[pos] = saved
	v.infoMu.Unlock()
	return nil
}

func (v *ytDLP) getTarget(pos string) []string {
	v.infoMu.Lock()
	saved, ok := v.infoCache[pos]
	v.infoMu.Unlock()
	if ok {
		if _, err := v.files.ReadFile(saved); err == nil {
			return []string{"--load-info-json", saved}
		}
	}
	return []string{pos}
}

func (v *ytDLP) GetMetadata(ctx context.Context, target string) (Metadata, error) {
	output, err := v.readInfo(ctx, target)
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
		return Metadata{}, fmt.Errorf("what yt-dlp output about %s: %w", target, err)
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

func (v *ytDLP) GetText(ctx context.Context, target string, want PreferredCaptions) (Text, error) {
	meta, err := v.GetMetadata(ctx, target)
	if err != nil {
		return Text{}, err
	}
	cues, err := v.subtitles(ctx, target, chooseLanguage(meta, want.Languages, want.IsAutomatic))
	if err != nil {
		return Text{}, err
	}
	return Text{
		Producer: producerCaptions,
		Cues:     cues,
		Prose:    "",
		Title:    meta.Title,
		Length:   meta.Length,
	}, nil
}

func (v *ytDLP) subtitles(ctx context.Context, pos string, language string) ([]Cue, error) {
	if language == "" {
		return nil, ErrNothingDownloaded
	}
	var raw []byte
	err := v.files.WithTempDir("numen-words-", func(into string) error {
		arguments := append([]string{
			"--skip-download", "--write-subs", "--write-auto-subs",
			"--sub-langs", language, "--sub-format", "json3",
			"--no-playlist", "-o", filepath.Join(into, "words"),
		}, v.getTarget(pos)...)
		if _, err := run(ctx, v.runner, v.command, nil, arguments...); err != nil {
			return err
		}
		names, err := v.files.ReadDir(into)
		if err != nil {
			return ErrNothingDownloaded
		}
		found := findWordsFile(names)
		if found == "" {
			return ErrNothingDownloaded
		}
		raw, err = v.files.ReadFile(filepath.Join(into, found))
		return err
	})
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

func findWordsFile(names []string) string {
	for _, name := range names {
		if strings.HasPrefix(name, "words.") && strings.HasSuffix(name, ".json3") {
			return name
		}
	}
	return ""
}

const copyFormat = "bestvideo[vcodec^=avc1]+bestaudio[acodec^=mp4a]/" +
	"best[vcodec!=none][acodec!=none]/bestvideo*+bestaudio/best"

// soundFormat is the sound alone, in an MPEG-4 container where the site has one.
const soundFormat = "bestaudio[acodec^=mp4a]/bestaudio"

func getFormatArguments(media Media) []string {
	if media == MediaAudio {
		return []string{"-f", soundFormat}
	}
	return []string{"-f", copyFormat, "--merge-output-format", "mp4"}
}

func (v *ytDLP) Download(ctx context.Context, target string, media Media, into io.Writer) (Copy, error) {
	var result Copy
	err := v.files.WithTempDir("numen-copy-", func(folder string) error {
		arguments := append(getFormatArguments(media), "--no-playlist", "-o", filepath.Join(folder, "copy.%(ext)s"))
		arguments = append(arguments, v.getTarget(target)...)
		if where := v.sound.getPath(); where != "" {
			arguments = append([]string{"--ffmpeg-location", where}, arguments...)
		}
		_, combined, err := v.runner.Run(ctx, v.command.command, v.command.env, nil, arguments...)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return fmt.Errorf("%w: %s", err, getLastLine(combined))
		}
		entries, err := v.files.ReadDir(folder)
		if err != nil {
			return err
		}
		if len(entries) != 1 {
			return fmt.Errorf("%w: %s", ErrNothingDownloaded, getLastLine(combined))
		}
		name := filepath.Join(folder, entries[0])
		if err := v.files.CopyFile(name, into); err != nil {
			return err
		}
		result = Copy{MediaType: getMediaType(name), Extension: filepath.Ext(name)}
		return nil
	})
	if err != nil {
		return Copy{}, err
	}
	return result, nil
}

// mediaTypeMP4Audio is the media type of the audio containers of MPEG-4.
const mediaTypeMP4Audio = "audio/mp4"

func getMediaType(name string) string {
	extension := strings.ToLower(filepath.Ext(name))
	if mediaType := getAudioMediaType(extension); mediaType != "" {
		return mediaType
	}
	switch extension {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	}
	return "application/octet-stream"
}

func getAudioMediaType(extension string) string {
	switch extension {
	case ".aac":
		return "audio/aac"
	case ".aif", ".aiff":
		return "audio/aiff"
	case ".alac":
		return "audio/alac"
	case ".flac":
		return "audio/flac"
	case ".m4a", ".m4b", ".m4p":
		return mediaTypeMP4Audio
	case ".mp3":
		return "audio/mpeg"
	case ".oga", ".ogg":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	case ".wav":
		return "audio/wav"
	case ".weba":
		return "audio/webm"
	}
	return ""
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

func isVideoSite(pos string) bool {
	address, err := url.Parse(pos)
	if err != nil {
		return false
	}
	switch strings.TrimPrefix(address.Hostname(), "www.") {
	case "youtube.com", "m.youtube.com", "music.youtube.com", "youtube-nocookie.com", "youtu.be":
		return true
	}
	return false
}
