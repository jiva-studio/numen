// Package report turns what the host says of the index into what the title bar shows: the kind of
// each source, the speed of embedding and the time of the last change.
package report

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/api"
)

func getExtensionKinds() map[string]api.Kind {
	kinds := map[string]api.Kind{}
	for _, extension := range []string{"md", "markdown", "mdx", "txt"} {
		kinds[extension] = api.KindNotes
	}
	kinds["pdf"] = api.KindPDF
	kinds["epub"] = api.KindEPUB
	for _, extension := range []string{
		"mp3", "m4a", "m4b", "aac", "wav", "flac", "ogg", "oga", "opus", "wma",
		"mp4", "m4v", "mov", "mkv", "webm", "avi", "mpeg", "mpg",
	} {
		kinds[extension] = api.KindMedia
	}
	return kinds
}

func getKindOrder() []api.Kind {
	return []api.Kind{api.KindNotes, api.KindPDF, api.KindEPUB, api.KindMedia, api.KindOther}
}

// GetKind returns the kind of source a file extension tells, in lower case and without the dot.
func GetKind(extension string) api.Kind {
	if kind, ok := getExtensionKinds()[extension]; ok {
		return kind
	}
	return api.KindOther
}

// GroupByKind adds up the sources and chunks of the extensions by kind, in the fixed order of the
// kinds, and leaves out a kind that holds no source.
func GroupByKind(extensions []host.ExtensionStats) []api.KindCount {
	totals := map[api.Kind]*api.KindCount{}
	for _, one := range extensions {
		kind := GetKind(one.Extension)
		total, ok := totals[kind]
		if !ok {
			total = &api.KindCount{Kind: kind, Sources: 0, Chunks: 0}
			totals[kind] = total
		}
		total.Sources += one.SourceCount
		total.Chunks += one.ChunkCount
	}
	grouped := make([]api.KindCount, 0, len(totals))
	for _, kind := range getKindOrder() {
		if total, ok := totals[kind]; ok && total.Sources > 0 {
			grouped = append(grouped, *total)
		}
	}
	return grouped
}
