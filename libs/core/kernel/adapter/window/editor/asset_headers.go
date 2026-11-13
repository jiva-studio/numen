package editor

import (
	"net/http"
	"slices"
)

func listShownTypes() []string {
	return []string{
		"image/png", "image/jpeg", "image/gif", "image/webp", "image/avif", "image/svg+xml",
		"audio/mpeg", "audio/mp4", "audio/vnd.wave", "audio/ogg", "audio/flac",
		"video/mp4", "video/webm",
		"application/pdf",
	}
}

func setAssetHeaders(head http.Header, mediaType string) {
	head.Set("X-Content-Type-Options", "nosniff")
	if !slices.Contains(listShownTypes(), mediaType) {
		head.Set("Content-Type", "application/octet-stream")
		head.Set("Content-Disposition", "attachment")
		return
	}
	head.Set("Content-Type", mediaType)
	if mediaType == "image/svg+xml" {
		// An SVG opened as a page runs its scripts; as a picture it does not.
		head.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}
}
