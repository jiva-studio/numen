package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// LibrarySearchInput is the input of the library_search tool.
type LibrarySearchInput struct {
	Query  string   `json:"query" jsonschema:"words to search for, as they appear in the text"`
	Limit  int      `json:"limit,omitempty" jsonschema:"how many passages to return (default: 20, max: 100)"`
	Kinds  []string `json:"kinds,omitempty" jsonschema:"the kinds of file to search: note for markdown notes, file for every other file (books, PDFs, recordings, scans); all kinds when left out"`
	Folder string   `json:"folder,omitempty" jsonschema:"limit the result to this folder and its subfolders, relative to the vault folder; the whole vault when left out"`
}

// Place is a place in a source: a page of a PDF, a position or range in a recording, or a position in
// an EPUB.
type Place struct {
	Page            int      `json:"page,omitempty" jsonschema:"PDF: the page, from 1"`
	PageCount       int      `json:"pageCount,omitempty" jsonschema:"PDF: the number of pages"`
	PositionSeconds *float64 `json:"positionSeconds,omitempty" jsonschema:"recording: seconds from the start"`
	EndSeconds      *float64 `json:"endSeconds,omitempty" jsonschema:"recording: seconds from the start to the end of the range"`
	CFI             string   `json:"cfi,omitempty" jsonschema:"EPUB: the CFI of the place"`
	Chapter         string   `json:"chapter,omitempty" jsonschema:"EPUB: the table-of-contents title at the place"`
}

// LibraryMatch is a passage that matches a library search.
type LibraryMatch struct {
	Path   string `json:"path" jsonschema:"the path of the source file, relative to the vault folder"`
	Title  string `json:"title" jsonschema:"the title of the note, else the name of the file"`
	Place  *Place `json:"place,omitempty" jsonschema:"the place of the passage in the source; absent for a source with no pages, times or EPUB positions, such as a note"`
	Text   string `json:"text" jsonschema:"the text around the hit, truncated to a fixed size, often mid-sentence"`
	Start  int    `json:"start" jsonschema:"the byte offset of the passage in the text of the file"`
	Length int    `json:"length" jsonschema:"the length of the passage in bytes"`
	Line   int    `json:"line" jsonschema:"the line of the hit, counted from 1 at the first line of the text; for a note, the first line of its body"`
	URI    string `json:"uri" jsonschema:"the address of this passage; use it as the target of a markdown link to the passage"`
}

// LibrarySearchOutput is the output of the library_search tool.
type LibrarySearchOutput struct {
	Query   string         `json:"query" jsonschema:"the query searched for"`
	Matches []LibraryMatch `json:"matches" jsonschema:"the matching passages, best first"`
}

// LibraryReadInput is the input of the library_read tool.
type LibraryReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault folder"`
	Page   int    `json:"page,omitempty" jsonschema:"the page to read, counted from 1; for a source whose text has pages"`
	Start  int    `json:"start,omitempty" jsonschema:"the byte offset in the text to start reading from; pass the nextStart of the previous result to continue"`
	Length int    `json:"length,omitempty" jsonschema:"bytes of text to read: default 20000 and at most 50000 for a text version; at most 16000 for a plain text file"`
}

// LibraryRange is a range of the text and the place in the source where it starts.
type LibraryRange struct {
	Start  int    `json:"start" jsonschema:"the byte offset of the range in the text"`
	Length int    `json:"length" jsonschema:"the length of the range in bytes"`
	Place  *Place `json:"place" jsonschema:"the place in the source where the range starts"`
}

// LibraryReadOutput is the output of the library_read tool.
type LibraryReadOutput struct {
	Path      string         `json:"path" jsonschema:"the path of the file, relative to the vault folder"`
	Kind      string         `json:"kind,omitempty" jsonschema:"the kind of the text version, as set by the plugin that created it; absent for a text file"`
	Version   string         `json:"version,omitempty" jsonschema:"the text version read"`
	Text      string         `json:"text" jsonschema:"the text read"`
	Start     int            `json:"start" jsonschema:"the byte offset of the returned text in the whole text"`
	Length    int            `json:"length" jsonschema:"the length of the returned text in bytes"`
	Size      int            `json:"size" jsonschema:"the length of the whole text in bytes"`
	NextStart *int           `json:"nextStart,omitempty" jsonschema:"the start to read next; absent at the end of the text"`
	Ranges    []LibraryRange `json:"ranges,omitempty" jsonschema:"the ranges of the returned text with their places in the source, in order"`
}

// WithLibrary sets the text versions and the search index of the vaults that the library tools use.
func (t *Tools) WithLibrary(texts host.Texts, search host.Search) *Tools {
	t.texts, t.search = texts, search
	return t
}

const (
	readLengthDefault = 20000
	readLengthMax     = 50000
)

var errNoLibrary = errors.New("the plugin was given neither text versions nor a search index")

func (t *Tools) addLibrarySearch(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Name:  "library_search",
		Title: "Search the library",
		Description: "Search the text index of all files in the vault: the notes, and the books, " +
			"PDFs, scans and recordings stored with them, by the words of the query and by their meaning. " +
			"Returns passages, best first: the text around each hit, the path of the file, its place " +
			"(a page, a chapter, a time) and its address. One file returns at most three passages. " +
			"note_search searches only notes, by title, path and text; use library_search for the " +
			"text of a book, a PDF or the transcript of a recording. " +
			"A passage is truncated to a fixed size, often mid-sentence: read more with library_read " +
			"before concluding that a source does not cover a subject. For a question about a book, " +
			"pass kinds [\"file\"]: a vault has far more notes than books, and a search of every " +
			"kind returns mostly notes.",
		Effect:       tools.ReadOnly,
		Step:         tools.Search,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{noteLinkSentence, passageLinkSentence, noFileLinkSentence, searchScopeSentence, userWordsSentence, truncatedSentence, bookKindSentence},
		Requires:     []string{host.PermissionIndexRead},
	}, inVault(t, (*Tools).searchLibrary))
}

func (t *Tools) searchLibrary(ctx context.Context, in LibrarySearchInput) (LibrarySearchOutput, error) {
	out := LibrarySearchOutput{Query: in.Query, Matches: []LibraryMatch{}}
	if strings.TrimSpace(in.Query) == "" {
		return out, nil
	}
	kinds := make([]domain.SourceKind, len(in.Kinds))
	for i, kind := range in.Kinds {
		kinds[i] = domain.SourceKind(kind)
	}
	if t.search == nil {
		return out, errNoLibrary
	}
	found, err := t.search.SearchPassages(ctx, host.PassageQuery{
		Query: in.Query, Limit: in.Limit, Kinds: kinds, PathPrefix: getFolderPrefix(in.Folder),
	})
	if err != nil {
		return out, fmt.Errorf("search the library: %w", err)
	}
	titles, err := t.getTitles(ctx, found)
	if err != nil {
		return out, err
	}
	for _, passage := range found {
		out.Matches = append(out.Matches, LibraryMatch{
			Path:   passage.Path,
			Title:  titles[passage.Path],
			Place:  getPassagePlace(passage),
			Text:   passage.Text,
			Start:  passage.Start,
			Length: passage.Length,
			Line:   passage.Line + 1,
			URI:    formatAddress(passage.Path, passage.Start, passage.Length),
		})
	}
	return out, nil
}

func getPassagePlace(passage host.Passage) *Place {
	var s selector
	if passage.Selector != "" && json.Unmarshal([]byte(passage.Selector), &s) != nil {
		s = selector{Page: 0, CFI: "", T: nil}
	}
	place := newPlace(s)
	if place != nil && place.CFI != "" {
		place.Chapter = passage.Location
	}
	return place
}

func newPlace(s selector) *Place {
	if s.Page == 0 && s.CFI == "" && len(s.T) == 0 {
		return nil
	}
	place := &Place{Page: s.Page, PageCount: 0, PositionSeconds: nil, EndSeconds: nil, CFI: s.CFI, Chapter: ""}
	if len(s.T) > 0 {
		place.PositionSeconds = &s.T[0]
	}
	if len(s.T) > 1 {
		place.EndSeconds = &s.T[1]
	}
	return place
}

func (t *Tools) getTitles(ctx context.Context, found []host.Passage) (map[string]string, error) {
	titles := make(map[string]string, len(found))
	var notes []string
	for _, passage := range found {
		base := path.Base(passage.Path)
		titles[passage.Path] = strings.TrimSuffix(base, path.Ext(base))
		if strings.HasSuffix(passage.Path, domain.NoteExtension) {
			notes = append(notes, passage.Path)
		}
	}
	refs, err := t.vault.GetNotes(ctx, notes)
	if err != nil {
		return nil, fmt.Errorf("read the titles of the notes: %w", err)
	}
	for notePath, ref := range refs {
		if ref.Title != "" {
			titles[notePath] = ref.Title
		}
	}
	return titles, nil
}

func formatAddress(source string, start, length int) string {
	return fmt.Sprintf("numen:%s?start=%d&length=%d", url.PathEscape(source), start, length)
}

func (t *Tools) addLibraryRead(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Name:  "library_read",
		Title: "Read the text of a file",
		Description: "Read the text of one file of the vault from its text version: the pages of a PDF, " +
			"the chapters of a book, the transcript of a recording, the recognized text of a scan. " +
			"Give a page, or a start and a length in bytes; the result lists the ranges of the text with " +
			"their places in the source, the length of the whole text, and the start to read next. " +
			"A text file is read as it is. A file with no text version returns an error: it must be " +
			"recognized or transcribed first. file_read reads the bytes of a text file and returns an error " +
			"for a binary file; " +
			"library_search finds the passage to read.",
		Effect:       tools.ReadOnly,
		Step:         tools.ByEffect,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{passageLinkSentence, truncatedSentence, libraryReadSentence},
		Requires:     []string{host.PermissionIndexRead},
	}, inVault(t, (*Tools).readLibrary))
}

func (t *Tools) readLibrary(ctx context.Context, in LibraryReadInput) (LibraryReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return LibraryReadOutput{}, errors.New("path is required")
	}
	if err := checkReadRange(in); err != nil {
		return LibraryReadOutput{}, err
	}
	if t.texts == nil {
		return LibraryReadOutput{}, errNoLibrary
	}
	listing, err := t.texts.ListVersions(ctx, in.Path)
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("list the text versions of %s: %w", in.Path, err)
	}
	if listing.Live == "" {
		return t.readPlainText(ctx, in)
	}
	var live domain.TextVersion
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			live = version
		}
	}
	out, err := t.readVersion(ctx, in, live)
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	return out, nil
}

func getReadLength(in LibraryReadInput) int {
	if in.Length > 0 {
		return min(in.Length, readLengthMax)
	}
	return readLengthDefault
}

func (t *Tools) readPlainText(ctx context.Context, in LibraryReadInput) (LibraryReadOutput, error) {
	read, err := t.vault.ReadRange(ctx, in.Path, in.Start, getReadLength(in))
	if errors.Is(err, host.ErrNotText) {
		return LibraryReadOutput{}, fmt.Errorf("%s has no text version; a recognition or a transcription creates one", in.Path)
	}
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read %s: %w", in.Path, err)
	}
	if in.Page > 0 {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: a text file has no pages; read it by start and length", in.Path)
	}
	size := int(read.Stat.Size)
	if err := checkStart(in.Start, size); err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	end := read.Start + len(read.Text)
	return LibraryReadOutput{
		Path: in.Path, Kind: "", Version: "", Text: read.Text, Start: read.Start, Length: len(read.Text), Size: size,
		NextStart: getNextStart(end, size), Ranges: nil,
	}, nil
}

func checkStart(start, size int) error {
	if start > size || (start == size && size > 0) {
		return fmt.Errorf("start %d is past the end of the text, which is %d bytes long", start, size)
	}
	return nil
}

func (t *Tools) readVersion(ctx context.Context, in LibraryReadInput, live domain.TextVersion) (LibraryReadOutput, error) {
	start, length := in.Start, getReadLength(in)
	if in.Page > 0 {
		pageStart, pageEnd, err := t.findVersionPage(ctx, in.Path, live.GetID(), in.Page)
		if err != nil {
			return LibraryReadOutput{}, err
		}
		if pageEnd == pageStart {
			return t.readEmptyPage(ctx, in.Path, live, pageStart)
		}
		start, length = pageStart, min(pageEnd-pageStart, length)
	}
	read, err := t.texts.ReadTextRange(ctx, in.Path, live.GetID(), start, length)
	if err != nil {
		return LibraryReadOutput{}, err
	}
	if err := checkStart(start, read.Size); err != nil {
		return LibraryReadOutput{}, err
	}
	places, err := getPlaces(read.Segments, read.Start)
	if err != nil {
		return LibraryReadOutput{}, err
	}
	var text strings.Builder
	for _, segment := range read.Segments {
		text.WriteString(segment.Text)
	}
	end := read.Start + text.Len()
	return LibraryReadOutput{
		Path: in.Path, Kind: string(live.Kind), Version: string(live.GetID()),
		Text: text.String(), Start: read.Start, Length: text.Len(), Size: read.Size,
		NextStart: getNextStart(end, read.Size), Ranges: getRanges(places, read.Start, end),
	}, nil
}

func (t *Tools) findVersionPage(ctx context.Context, path string, id domain.TextVersionID, page int) (start, end int, err error) {
	var places []place
	cursor, offset := "", 0
	for {
		read, err := t.texts.ReadText(ctx, path, id, cursor)
		if err != nil {
			return 0, 0, err
		}
		more, err := getPlaces(read.Segments, offset)
		if err != nil {
			return 0, 0, err
		}
		places = append(places, more...)
		if len(places) > 0 {
			offset = places[len(places)-1].end
		}
		if read.NextCursor == "" || isPastPage(places, page) {
			return findPage(page, places)
		}
		if read.NextCursor == cursor {
			return 0, 0, fmt.Errorf("the host returned the page at %q twice", cursor)
		}
		cursor = read.NextCursor
	}
}

func isPastPage(places []place, page int) bool {
	isSeen := false
	for _, p := range places {
		isSeen = isSeen || p.selector.Page == page
		if isSeen && p.selector.Page > page {
			return true
		}
	}
	return false
}

// place is a segment of a text, its byte range, and its location in the numbering of the source.
type place struct {
	start, end int
	selector   selector
}

// selector has the keys of a segment's selector that identify a location: a page of a PDF counted from 1,
// the position in a book, and the span of a recording in seconds.
type selector struct {
	Page int       `json:"page"`
	CFI  string    `json:"cfi"`
	T    []float64 `json:"t"`
}

func getPlaces(segments []domain.TextSegment, offset int) ([]place, error) {
	places := make([]place, 0, len(segments))
	for _, segment := range segments {
		var s selector
		if segment.Selector != "" {
			if err := json.Unmarshal([]byte(segment.Selector), &s); err != nil {
				return nil, fmt.Errorf("segment at byte %d has an invalid selector %q: %w", offset, segment.Selector, err)
			}
		}
		places = append(places, place{start: offset, end: offset + len(segment.Text), selector: s})
		offset += len(segment.Text)
	}
	return places, nil
}

func getNextStart(end, size int) *int {
	if end >= size {
		return nil
	}
	return &end
}

func findPage(page int, places []place) (start, end int, err error) {
	first, last := 0, 0
	start, end = -1, -1
	for _, p := range places {
		if p.selector.Page == 0 {
			continue
		}
		if first == 0 || p.selector.Page < first {
			first = p.selector.Page
		}
		last = max(last, p.selector.Page)
		if p.selector.Page == page {
			if start < 0 {
				start = p.start
			}
			end = p.end
		}
	}
	switch {
	case first == 0:
		return 0, 0, errors.New("the text has no pages; read it by start and length")
	case start < 0:
		return 0, 0, fmt.Errorf("the text has no page %d: its pages are %d to %d", page, first, last)
	}
	return start, end, nil
}

func getRanges(places []place, rangeStart, rangeEnd int) []LibraryRange {
	var out []LibraryRange
	for _, p := range places {
		start, end := max(p.start, rangeStart), min(p.end, rangeEnd)
		if start >= end {
			continue
		}
		segmentPlace := newPlace(p.selector)
		if segmentPlace == nil {
			continue
		}
		if last := len(out) - 1; last >= 0 && segmentPlace.Page > 0 && out[last].Place.Page == segmentPlace.Page {
			out[last].Length = end - out[last].Start
			continue
		}
		out = append(out, LibraryRange{Start: start, Length: end - start, Place: segmentPlace})
	}
	return out
}

func checkReadRange(in LibraryReadInput) error {
	for _, field := range []struct {
		name  string
		value int
	}{{"page", in.Page}, {"start", in.Start}, {"length", in.Length}} {
		if field.value < 0 {
			return fmt.Errorf("%s is %d: give a number of zero or more", field.name, field.value)
		}
	}
	if in.Page > 0 && in.Start > 0 {
		return errors.New("give a page, or a start and a length, not a page and a start")
	}
	return nil
}

func (t *Tools) readEmptyPage(ctx context.Context, path string, live domain.TextVersion, start int) (LibraryReadOutput, error) {
	read, err := t.texts.ReadTextRange(ctx, path, live.GetID(), start, 1)
	if err != nil {
		return LibraryReadOutput{}, err
	}
	return LibraryReadOutput{
		Path: path, Kind: string(live.Kind), Version: string(live.GetID()), Text: "", Start: start, Length: 0, Size: read.Size,
		NextStart: getNextStart(start, read.Size), Ranges: nil,
	}, nil
}
