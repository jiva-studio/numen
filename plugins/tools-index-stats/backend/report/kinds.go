// Package report turns what the host says of the index into what the title bar shows: the kind of
// each source, the speed of embedding and the time of the last change.
package report

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// Kind is the sort of source the file type of a source tells.
type Kind string

const (
	// KindNotes is the notes of the vault.
	KindNotes Kind = "notes"
	// KindPDF is the PDF documents.
	KindPDF Kind = "pdf"
	// KindEPUB is the EPUB books.
	KindEPUB Kind = "epub"
	// KindMedia is the audio and video files, which the index holds as transcripts.
	KindMedia Kind = "media"
	// KindOther is every other file.
	KindOther Kind = "other"
)

// KindCount is the sources of one kind and the chunks that can carry a vector among those they hold.
type KindCount struct {
	Kind    Kind
	Sources int64
	Chunks  int64
}

func getExtensionKinds() map[string]Kind {
	kinds := map[string]Kind{}
	for _, extension := range []string{"md", "markdown", "mdx", "txt"} {
		kinds[extension] = KindNotes
	}
	kinds["pdf"] = KindPDF
	kinds["epub"] = KindEPUB
	for _, extension := range []string{
		"mp3", "m4a", "m4b", "aac", "wav", "flac", "ogg", "oga", "opus", "wma",
		"mp4", "m4v", "mov", "mkv", "webm", "avi", "mpeg", "mpg",
	} {
		kinds[extension] = KindMedia
	}
	return kinds
}

func getKindOrder() []Kind {
	return []Kind{KindNotes, KindPDF, KindEPUB, KindMedia, KindOther}
}

// GetKind returns the kind of source a file extension tells, in lower case and without the dot.
func GetKind(extension string) Kind {
	if kind, ok := getExtensionKinds()[extension]; ok {
		return kind
	}
	return KindOther
}

// GroupByKind adds up the sources and chunks of the extensions by kind, in the fixed order of the
// kinds, and leaves out a kind that holds no source.
func GroupByKind(extensions []host.ExtensionStats) []KindCount {
	totals := map[Kind]*KindCount{}
	for _, one := range extensions {
		kind := GetKind(one.Extension)
		total, ok := totals[kind]
		if !ok {
			total = &KindCount{Kind: kind, Sources: 0, Chunks: 0}
			totals[kind] = total
		}
		total.Sources += one.SourceCount
		total.Chunks += one.ChunkCount
	}
	grouped := make([]KindCount, 0, len(totals))
	for _, kind := range getKindOrder() {
		if total, ok := totals[kind]; ok && total.Sources > 0 {
			grouped = append(grouped, *total)
		}
	}
	return grouped
}
