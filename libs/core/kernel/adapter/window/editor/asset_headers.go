package editor

import (
	"net/http"
	"path/filepath"
	"strings"
)

func getShownType(ext string) (string, bool) {
	shownTypes := map[string]string{
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
	shown, isShown := shownTypes[ext]
	return shown, isShown
}

func setAssetHeaders(head http.Header, name string) {
	head.Set("X-Content-Type-Options", "nosniff")
	shown, isShown := getShownType(strings.ToLower(filepath.Ext(name)))
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
