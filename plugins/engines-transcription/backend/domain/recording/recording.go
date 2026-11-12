// Package recording knows which files are audio or video recordings. It is pure: it reads no file.
package recording

import (
	"path"
	"strings"
)

// IsRecording reports whether the extension of file names an audio or video format.
func IsRecording(file string) bool {
	switch strings.ToLower(path.Ext(file)) {
	case ".aac", ".ac3", ".aif", ".aifc", ".aiff", ".amr", ".ape", ".au", ".flac", ".m4a", ".m4b",
		".mka", ".mp2", ".mp3", ".mpc", ".oga", ".ogg", ".opus", ".ra", ".snd", ".spx", ".wav", ".weba",
		".wma", ".wv":
		return true
	case ".3g2", ".3gp", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".ogv", ".qt",
		".webm", ".wmv":
		return true
	}
	return false
}
