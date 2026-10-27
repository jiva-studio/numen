package pluginport

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidPlayback is a playback declaration that names no content type, or names one wrongly.
var ErrInvalidPlayback = errors.New("invalid playback")

// PluginPlayback declares that the plugin keeps a copy to play in place of files of the content
// types ContentTypes names. The copy is a file of the plugin's store folder named by the content hash
// of the file and an extension of an audio or video type: `<hash>.mp4`.
type PluginPlayback struct {
	ContentTypes []domain.ContentType `json:"contentTypes"`
}

// IsCopying reports whether the plugin keeps copies of files of the type.
func (p PluginPlayback) IsCopying(contentType domain.ContentType) bool {
	return isAnyMatching(p.ContentTypes, contentType)
}

// listPlaybackIssues checks that a playback declaration names a content type, each a type or a mask.
func (m PluginManifest) listPlaybackIssues() []error {
	in := m.Playback
	if in == nil {
		return nil
	}
	if len(in.ContentTypes) == 0 {
		return []error{fmt.Errorf("%w: it names no content type", ErrInvalidPlayback)}
	}
	return listContentTypesIssues(ErrInvalidPlayback, in.ContentTypes)
}
