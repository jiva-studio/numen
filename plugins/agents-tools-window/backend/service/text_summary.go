package service

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// cfiStart is a segment of a book's text version that a CFI names, and the heading in force there.
type cfiStart struct {
	cfi     []int
	heading string
}

// textSummary is what the positions of a file are read from: the highest page and the CFI places of
// its live text version. The zero summary is a file with no text version.
type textSummary struct {
	lastPage int
	places   []cfiStart
}

// summaryKey names a file of a vault.
type summaryKey struct {
	vault string
	path  string
}

// cachedSummary is the summary of a file and the hash of the live text version it was read from.
type cachedSummary struct {
	version domain.TextVersionID
	hash    string
	summary textSummary
}

// textSummaries keeps the summary of each file for as long as its live text version stays the same,
// so a repeated read of a file costs one ListVersions. A version replaced under the same key has
// another hash.
type textSummaries struct {
	mu      sync.Mutex
	entries map[summaryKey]cachedSummary
}

func newTextSummaries() *textSummaries {
	return &textSummaries{mu: sync.Mutex{}, entries: map[summaryKey]cachedSummary{}}
}

// get is the summary of the file's live text version, and the zero summary where the file has none
// or it cannot be read.
func (s *textSummaries) get(ctx context.Context, texts host.Texts, vault, path string) textSummary {
	none := textSummary{lastPage: 0, places: nil}
	if texts == nil || path == "" {
		return none
	}
	live, isLive := readLiveVersion(ctx, texts, path)
	if !isLive {
		return none
	}
	key := summaryKey{vault: vault, path: path}
	s.mu.Lock()
	cached, isCached := s.entries[key]
	s.mu.Unlock()
	if isCached && cached.version == live.GetID() && cached.hash == live.TextHash {
		return cached.summary
	}
	text, err := host.ReadWholeText(ctx, texts, live)
	if err != nil {
		return none
	}
	summary := summarizeSegments(text.GetSegments())
	s.mu.Lock()
	s.entries[key] = cachedSummary{version: live.GetID(), hash: live.TextHash, summary: summary}
	s.mu.Unlock()
	return summary
}

// readLiveVersion is the live text version of the file, and false where it has none.
func readLiveVersion(ctx context.Context, texts host.Texts, path string) (domain.TextVersion, bool) {
	listing, err := texts.ListVersions(ctx, path)
	if err != nil || listing.Live == "" {
		return domain.TextVersion{}, false
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			return version, true
		}
	}
	return domain.TextVersion{}, false
}

// summarizeSegments is the highest page the selectors name, and each segment a CFI names with the
// heading in force at it. A heading holds from the segment that opens it until the next one.
func summarizeSegments(segments []domain.TextSegment) textSummary {
	summary := textSummary{lastPage: 0, places: nil}
	heading := ""
	for _, segment := range segments {
		if segment.Heading != "" {
			heading = segment.Heading
		}
		var selector struct {
			Page int    `json:"page"`
			CFI  string `json:"cfi"`
		}
		if json.Unmarshal([]byte(segment.Selector), &selector) != nil {
			continue
		}
		summary.lastPage = max(summary.lastPage, selector.Page)
		if selector.CFI != "" {
			summary.places = append(summary.places, cfiStart{cfi: parseCFI(selector.CFI), heading: heading})
		}
	}
	return summary
}
