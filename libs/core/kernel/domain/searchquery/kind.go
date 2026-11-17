package searchquery

import (
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// The kinds of a document.
const (
	KindNote  = "note"
	KindPDF   = "pdf"
	KindEPUB  = "epub"
	KindAudio = "audio"
	KindVideo = "video"
	KindImage = "image"
	KindFile  = "file"
)

// ListKinds returns every kind a query may compare kind with.
func ListKinds() []string {
	return []string{KindNote, KindPDF, KindEPUB, KindAudio, KindVideo, KindImage, KindFile}
}

// IsKind reports whether the text is a kind of ListKinds.
func IsKind(text string) bool {
	return slices.Contains(ListKinds(), text)
}

// GetKind returns the kind of a document from its source kind and its MIME type.
func GetKind(sourceKind domain.SourceKind, contentType domain.ContentType) string {
	if sourceKind == domain.KindNote {
		return KindNote
	}
	switch mime := string(contentType); {
	case mime == "application/pdf":
		return KindPDF
	case mime == "application/epub+zip":
		return KindEPUB
	case strings.HasPrefix(mime, "audio/"):
		return KindAudio
	case strings.HasPrefix(mime, "video/"):
		return KindVideo
	case strings.HasPrefix(mime, "image/"):
		return KindImage
	}
	return KindFile
}

// GetSourceKind returns the source kind every document of the kind has.
func GetSourceKind(kind string) domain.SourceKind {
	if kind == KindNote {
		return domain.KindNote
	}
	return domain.KindFile
}
