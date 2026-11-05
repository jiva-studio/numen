package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// getCallTexts is the text versions of the vault the call names, nil where the host offers none.
func getCallTexts(ctx context.Context, texts host.Texts) host.Texts {
	if texts == nil {
		return nil
	}
	if id := tools.GetVault(ctx); id != "" {
		return texts.With(domain.VaultID(id))
	}
	return texts
}

// readPosition is where in its file a tab is open, read from the keys of the state its kind saved:
// {"page":N} a page, {"t":[start,end]} seconds of a recording, {"cfi":"..."} a place of an EPUB. A
// state of other keys, or one that is not JSON, has no position.
func readPosition(ctx context.Context, texts host.Texts, tab host.OpenTab) string {
	var state map[string]json.RawMessage
	if err := json.Unmarshal([]byte(tab.State), &state); err != nil {
		return ""
	}
	if raw, isPage := state["page"]; isPage {
		return readPagePosition(ctx, texts, tab.Path, raw)
	}
	if raw, isTime := state["t"]; isTime {
		return readTimePosition(raw)
	}
	if raw, isCFI := state["cfi"]; isCFI {
		return readChapterPosition(ctx, texts, tab.Path, raw)
	}
	return ""
}

// readPagePosition is "page N of M", M the highest page of the file's live text version, or "page N"
// where the file has no text version.
func readPagePosition(ctx context.Context, texts host.Texts, path string, raw json.RawMessage) string {
	var page int
	if err := json.Unmarshal(raw, &page); err != nil || page < 1 {
		return ""
	}
	last := 0
	for _, segment := range readLiveSegments(ctx, texts, path) {
		var selector struct {
			Page int `json:"page"`
		}
		if json.Unmarshal([]byte(segment.Selector), &selector) == nil {
			last = max(last, selector.Page)
		}
	}
	if last < 1 {
		return fmt.Sprintf("page %d", page)
	}
	return fmt.Sprintf("page %d of %d", page, last)
}

// readTimePosition is "at M:SS", or "at H:MM:SS" from one hour, of the start of the selection.
func readTimePosition(raw json.RawMessage) string {
	var span []float64
	if err := json.Unmarshal(raw, &span); err != nil || len(span) == 0 || span[0] < 0 {
		return ""
	}
	seconds := int(math.Floor(span[0]))
	hours, minutes, rest := seconds/3600, seconds/60%60, seconds%60
	if hours > 0 {
		return fmt.Sprintf("at %d:%02d:%02d", hours, minutes, rest)
	}
	return fmt.Sprintf("at %d:%02d", minutes, rest)
}

// readChapterPosition is the chapter of the live text version the CFI falls in: the heading of the
// segment at the CFI, or of the closest one before it. A CFI before every segment, and a file with
// no text version, have none.
func readChapterPosition(ctx context.Context, texts host.Texts, path string, raw json.RawMessage) string {
	var cfi string
	if err := json.Unmarshal(raw, &cfi); err != nil || cfi == "" {
		return ""
	}
	target := parseCFI(cfi)
	heading, chapter := "", ""
	for _, segment := range readLiveSegments(ctx, texts, path) {
		if segment.Heading != "" {
			heading = segment.Heading
		}
		var selector struct {
			CFI string `json:"cfi"`
		}
		if json.Unmarshal([]byte(segment.Selector), &selector) != nil || selector.CFI == "" {
			continue
		}
		if slices.Compare(parseCFI(selector.CFI), target) <= 0 {
			chapter = heading
		}
	}
	if chapter == "" {
		return ""
	}
	return fmt.Sprintf("chapter %q", chapter)
}

// parseCFI is the numbers of an EPUB CFI in document order: its steps and its character offset. Two
// CFIs of one book compare in reading order as these numbers compare. Assertions in brackets are
// left out, and a range is read as its start.
func parseCFI(cfi string) []int {
	body := strings.TrimSuffix(strings.TrimPrefix(cfi, "epubcfi("), ")")
	if parts := strings.Split(body, ","); len(parts) == 3 {
		body = parts[0] + parts[1]
	}
	var numbers []int
	current, isNumber, depth := 0, false, 0
	for _, char := range body {
		switch {
		case char == '[':
			depth++
		case char == ']':
			depth--
		case depth > 0:
		case char >= '0' && char <= '9':
			current, isNumber = current*10+int(char-'0'), true
		case isNumber:
			numbers, current, isNumber = append(numbers, current), 0, false
		}
	}
	if isNumber {
		numbers = append(numbers, current)
	}
	return numbers
}

// readLiveSegments is the segments of the file's live text version, and none where the file has no
// text version or it cannot be read.
func readLiveSegments(ctx context.Context, texts host.Texts, path string) []domain.TextSegment {
	if texts == nil || path == "" {
		return nil
	}
	listing, err := texts.ListVersions(ctx, path)
	if err != nil || listing.Live == "" {
		return nil
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		text, err := host.ReadWholeText(ctx, texts, version)
		if err != nil {
			return nil
		}
		return text.GetSegments()
	}
	return nil
}
