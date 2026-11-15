package query

import (
	"path"
	"slices"
	"strings"
)

// The kinds of files a query compares kind with.
const (
	KindNote  = "note"
	KindPDF   = "pdf"
	KindEPUB  = "epub"
	KindAudio = "audio"
	KindVideo = "video"
	KindImage = "image"
)

// The kinds of source the host filters a search by.
const (
	hostKindNote = "note"
	hostKindFile = "file"
)

func listKinds() []string {
	return []string{KindNote, KindPDF, KindEPUB, KindAudio, KindVideo, KindImage, ""}
}

func isKind(kind string) bool {
	return kind != "" && slices.Contains(listKinds(), kind)
}

// GetKind returns the kind of the file at the path, by its extension; empty for a file of no kind.
func GetKind(filePath string) string {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".md":
		return KindNote
	case ".pdf":
		return KindPDF
	case ".epub":
		return KindEPUB
	case ".mp3", ".m4a", ".wav", ".ogg", ".opus", ".flac":
		return KindAudio
	case ".mp4", ".mkv", ".webm", ".mov":
		return KindVideo
	case ".png", ".jpg", ".jpeg", ".webp", ".tiff":
		return KindImage
	}
	return ""
}

func getHostKind(kind string) string {
	if kind == KindNote {
		return hostKindNote
	}
	return hostKindFile
}
