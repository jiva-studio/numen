package pluginport

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidPlayback is a playback declaration that names no file type, or names one wrongly.
var ErrInvalidPlayback = errors.New("invalid playback")

// PluginPlayback declares that the plugin keeps a copy to play in place of each file of the types
// Files names, by extension with its dot. The copy is a file of the plugin's store folder named by
// the content hash of the file and the extension of its media type: `<hash>.mp4`.
type PluginPlayback struct {
	Files []string `json:"files"`
}

// listPlaybackIssues checks that a playback declaration names a file type, each as an extension.
func (m PluginManifest) listPlaybackIssues() []error {
	in := m.Playback
	if in == nil {
		return nil
	}
	if len(in.Files) == 0 {
		return []error{fmt.Errorf("%w: it names no file type", ErrInvalidPlayback)}
	}
	var issues []error
	for _, ext := range in.Files {
		if len(ext) < 2 || !strings.HasPrefix(ext, ".") || strings.ContainsAny(ext[1:], `./\ `) {
			issues = append(issues, fmt.Errorf("%w: %q is not an extension", ErrInvalidPlayback, ext))
		}
	}
	return issues
}
