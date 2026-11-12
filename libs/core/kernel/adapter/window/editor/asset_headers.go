package editor

import (
	"net/http"
	"path/filepath"
	"strings"
)

// shownTypes are the content types a vault file is served under for display in the window. The
// type comes from the extension and never from the bytes, so a file named for one kind and holding
// another is not shown as a page or a script.
var shownTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".svg":  "image/svg+xml",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".pdf":  "application/pdf",
}

// setAssetHeaders sets the type of a vault file and the headers that keep the browser to it. A file
// of any other kind is served as bytes to download.
func setAssetHeaders(head http.Header, name string) {
	head.Set("X-Content-Type-Options", "nosniff")
	shown, isShown := shownTypes[strings.ToLower(filepath.Ext(name))]
	if !isShown {
		head.Set("Content-Type", "application/octet-stream")
		head.Set("Content-Disposition", "attachment")
		return
	}
	head.Set("Content-Type", shown)
	if shown == "image/svg+xml" {
		// An SVG opened as a page runs its scripts; as a picture it does not.
		head.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}
}
