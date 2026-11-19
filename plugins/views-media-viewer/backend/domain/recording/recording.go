// Package recording names captured audio files in a vault.
package recording

import (
	"errors"
	"io/fs"
	"mime"
	"path"
	"strings"
	"time"
)

// GetPath returns a vault path with the extension of the captured audio format.
func GetPath(folder, mediaType, startedAt string) (string, error) {
	if folder != "" && (!fs.ValidPath(folder) || folder == "." || strings.ContainsAny(folder, `\:`)) {
		return "", errors.New("invalid recording folder")
	}
	kind, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return "", err
	}
	extension := map[string]string{"audio/webm": ".webm", "audio/mp4": ".m4a", "audio/ogg": ".ogg"}[kind]
	if extension == "" {
		return "", errors.New("unsupported audio recording format")
	}
	timestamp, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return "", err
	}
	return path.Join(folder, "Recording "+timestamp.UTC().Format("2006-01-02 15-04-05.000")+extension), nil
}
