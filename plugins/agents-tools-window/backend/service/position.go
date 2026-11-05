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

// positionReader reads where the tabs of one call are open, reading the text version of each file
// at most once.
type positionReader struct {
	texts     host.Texts
	vault     string
	summaries *textSummaries
	read      map[string]textSummary
}

// newPositionReader is the reader of the positions of the call's tabs, over the text versions of
// the vault the call names. Nil texts read no text version.
func newPositionReader(ctx context.Context, texts host.Texts, summaries *textSummaries) *positionReader {
	vault := tools.GetVault(ctx)
	if texts != nil && vault != "" {
		texts = texts.With(domain.VaultID(vault))
	}
	return &positionReader{texts: texts, vault: vault, summaries: summaries, read: map[string]textSummary{}}
}

// getSummary is what the positions of the file are read from, read once per call.
func (r *positionReader) getSummary(ctx context.Context, path string) textSummary {
	if summary, isRead := r.read[path]; isRead {
		return summary
	}
	summary := r.summaries.get(ctx, r.texts, r.vault, path)
	r.read[path] = summary
	return summary
}

// readPosition is where in its file a tab is open, read from the keys of the state its kind saved:
// {"page":N} a page, {"t":[start,end]} seconds of a recording, {"cfi":"..."} a place of an EPUB. A
// state of other keys, or one that is not JSON, has no position.
func (r *positionReader) readPosition(ctx context.Context, tab host.OpenTab) string {
	var state map[string]json.RawMessage
	if err := json.Unmarshal([]byte(tab.State), &state); err != nil {
		return ""
	}
	if raw, isPage := state["page"]; isPage {
		return r.readPagePosition(ctx, tab.Path, raw)
	}
	if raw, isTime := state["t"]; isTime {
		return readTimePosition(raw)
	}
	if raw, isCFI := state["cfi"]; isCFI {
		return r.readChapterPosition(ctx, tab.Path, raw)
	}
	return ""
}

// readPagePosition is "page N of M", M the highest page of the file's live text version, or "page N"
// where the file has no text version.
func (r *positionReader) readPagePosition(ctx context.Context, path string, raw json.RawMessage) string {
	var page int
	if err := json.Unmarshal(raw, &page); err != nil || page < 1 {
		return ""
	}
	if last := r.getSummary(ctx, path).lastPage; last > 0 {
		return fmt.Sprintf("page %d of %d", page, last)
	}
	return fmt.Sprintf("page %d", page)
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

// readChapterPosition is the chapter of the live text version the CFI falls in: the heading in force
// at the last segment whose CFI is at or before it. A CFI before every segment, and a file with no
// text version, have none.
func (r *positionReader) readChapterPosition(ctx context.Context, path string, raw json.RawMessage) string {
	var cfi string
	if err := json.Unmarshal(raw, &cfi); err != nil || cfi == "" {
		return ""
	}
	target := parseCFI(cfi)
	chapter := ""
	for _, start := range r.getSummary(ctx, path).places {
		if slices.Compare(start.cfi, target) <= 0 {
			chapter = start.heading
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
