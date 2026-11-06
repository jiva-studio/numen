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
	Query      string   `json:"query" jsonschema:"words to search for, as they appear in the text"`
	Limit      int      `json:"limit,omitempty" jsonschema:"how many passages to return (default: 20, max: 100)"`
	Kinds      []string `json:"kinds,omitempty" jsonschema:"the kinds of file to search: note for markdown notes, file for every other file (books, PDFs, recordings, scans); all kinds when left out"`
	PathPrefix string   `json:"pathPrefix,omitempty" jsonschema:"search only the files whose path starts with this prefix, relative to the vault folder, such as library/"`
}

// LibraryMatch is a passage that answers a library search.
type LibraryMatch struct {
	Path     string `json:"path" jsonschema:"the path of the file the text is read from, relative to the vault folder"`
	Title    string `json:"title" jsonschema:"the title of the note, else the name of the file"`
	Location string `json:"location,omitempty" jsonschema:"the location in the numbering of the source, such as a page, a chapter or a time; absent when the format has none"`
	Text     string `json:"text" jsonschema:"the text around the hit, truncated to a fixed size, often mid-sentence"`
	Start    int    `json:"start" jsonschema:"the byte offset of the passage in the text of the file"`
	Length   int    `json:"length" jsonschema:"the length of the passage in bytes"`
	Line     int    `json:"line,omitempty" jsonschema:"the line of the hit in a note, counted from the first line of its body"`
	URI      string `json:"uri" jsonschema:"the address of this passage; write it into the answer as a markdown link to the passage"`
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
	Start  int    `json:"start,omitempty" jsonschema:"the byte offset in the text to start reading from; the nextStart of the previous answer continues it"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes of text to read; at most 50000, and 20000 when left out"`
}

// LibraryLocation is a range of the text and its location in the numbering of the source.
type LibraryLocation struct {
	Start       int     `json:"start" jsonschema:"the byte offset of the range in the text"`
	Length      int     `json:"length" jsonschema:"the length of the range in bytes"`
	Page        int     `json:"page,omitempty" jsonschema:"the page, counted from 1"`
	CFI         string  `json:"cfi,omitempty" jsonschema:"the EPUB CFI of the range in a book"`
	FromSeconds float64 `json:"fromSeconds,omitempty" jsonschema:"the start time of the range in the recording, in seconds"`
	ToSeconds   float64 `json:"toSeconds,omitempty" jsonschema:"the end time of the range in the recording, in seconds"`
}

// LibraryReadOutput is the output of the library_read tool.
type LibraryReadOutput struct {
	Path      string            `json:"path" jsonschema:"the path of the file, relative to the vault folder"`
	Kind      string            `json:"kind,omitempty" jsonschema:"the kind of the text version, as the plugin that made it names it; absent for a text file"`
	Version   string            `json:"version,omitempty" jsonschema:"the text version read"`
	Text      string            `json:"text" jsonschema:"the text read"`
	Start     int               `json:"start" jsonschema:"the byte offset of the returned text in the whole text"`
	Length    int               `json:"length" jsonschema:"the length of the returned text in bytes"`
	Size      int               `json:"size" jsonschema:"the length of the whole text in bytes"`
	NextStart *int              `json:"nextStart,omitempty" jsonschema:"the start to read next; absent at the end of the text"`
	Locations []LibraryLocation `json:"locations,omitempty" jsonschema:"the locations of the source the returned text covers, in order"`
}

// WithLibrary gives the tools the text versions and the search index of the vaults, which the tools
// that read the library need.
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
		Description: "Search the text index of everything the vault holds: the notes, and the books, " +
			"PDFs, scans and recordings stored with them, by the words of the query and by their meaning. " +
			"Returns passages, best first: the text around each hit, the path of the file, its location " +
			"(a page, a chapter, a time) and its address. One file returns at most three passages. " +
			"note_search searches only notes, by title, path and text; use library_search for the " +
			"text of a book, a PDF or the transcript of a recording. " +
			"A passage is truncated to a fixed size, often mid-sentence: read on with library_read " +
			"before concluding that a source does not cover a subject. For a question about a book, " +
			"pass kinds [\"file\"]: a vault holds far more notes than books, and a search of every " +
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
		Query: in.Query, Limit: in.Limit, Kinds: kinds, PathPrefix: getFolderPrefix(in.PathPrefix),
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
			Path:     passage.Path,
			Title:    titles[passage.Path],
			Location: passage.Location,
			Text:     passage.Text,
			Start:    passage.Start,
			Length:   passage.Length,
			Line:     passage.Line,
			URI:      formatAddress(passage.Path, passage.Start, passage.Length),
		})
	}
	return out, nil
}

// getTitles is the title of each source: a note's own, else the name of the file without its extension.
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

// formatAddress is where a passage is reached, as the window opens one.
func formatAddress(source string, start, length int) string {
	return fmt.Sprintf("numen:%s?start=%d&length=%d", url.PathEscape(source), start, length)
}

func (t *Tools) addLibraryRead(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Name:  "library_read",
		Title: "Read the text of a file",
		Description: "Read the text of one file of the vault from its text version: the pages of a PDF, " +
			"the chapters of a book, the transcript of a recording, the recognised text of a scan. " +
			"Give a page, or a start and a length in bytes; the answer names the places the text " +
			"covers, the length of the whole text, and the start to read next. " +
			"A text file is read as it is. A file with no text version returns an error that names " +
			"what makes one. file_read reads the bytes of a text file and refuses a binary; " +
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

// getReadLength is how many bytes a call asks for: the default when it names none, and at most the
// most one call reads.
func getReadLength(in LibraryReadInput) int {
	if in.Length > 0 {
		return min(in.Length, readLengthMax)
	}
	return readLengthDefault
}

// readPlainText reads a file that has no text version through a ranged read of the host: a text file
// is its own text, and any other file has no text to read.
func (t *Tools) readPlainText(ctx context.Context, in LibraryReadInput) (LibraryReadOutput, error) {
	read, err := t.vault.ReadRange(ctx, in.Path, in.Start, getReadLength(in))
	if errors.Is(err, host.ErrNotText) {
		return LibraryReadOutput{}, fmt.Errorf("%s has no text version; recognition_create or transcript_create makes one", in.Path)
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
	to := read.Start + len(read.Text)
	return LibraryReadOutput{
		Path: in.Path, Kind: "", Version: "", Text: read.Text, Start: read.Start, Length: len(read.Text), Size: size,
		NextStart: getNextStart(to, size), Locations: nil,
	}, nil
}

// checkStart refuses a start at or past the end of a text that is not empty.
func checkStart(start, size int) error {
	if start > size || (start == size && size > 0) {
		return fmt.Errorf("start %d is past the end of the text, which is %d bytes long", start, size)
	}
	return nil
}

// readVersion reads the range of a text version a call asks for: a page, or a start and a length.
func (t *Tools) readVersion(ctx context.Context, in LibraryReadInput, live domain.TextVersion) (LibraryReadOutput, error) {
	start, length := in.Start, getReadLength(in)
	if in.Page > 0 {
		from, end, err := t.findVersionPage(ctx, in.Path, live.GetID(), in.Page)
		if err != nil {
			return LibraryReadOutput{}, err
		}
		if end == from {
			return t.readEmptyPage(ctx, in.Path, live, from)
		}
		start, length = from, min(end-from, length)
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
	to := read.Start + text.Len()
	return LibraryReadOutput{
		Path: in.Path, Kind: string(live.Kind), Version: string(live.GetID()),
		Text: text.String(), Start: read.Start, Length: text.Len(), Size: read.Size,
		NextStart: getNextStart(to, read.Size), Locations: getLocations(places, read.Start, to),
	}, nil
}

// findVersionPage is the byte range of a page of a text version. It reads the pages of segments by
// cursor up to the first page past the one asked for, and to the end where the page is not there.
func (t *Tools) findVersionPage(ctx context.Context, path string, id domain.TextVersionID, page int) (from, to int, err error) {
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

// isPastPage reports whether the places hold the page and a numbered page after it.
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

// place is a segment of a text, its byte range, and the position the source numbers it by.
type place struct {
	start, end int
	selector   selector
}

// selector is the keys of a segment's selector that name a place: a page of a PDF counted from 1,
// the position in a book, and the span of a recording in seconds.
type selector struct {
	Page int       `json:"page"`
	CFI  string    `json:"cfi"`
	T    []float64 `json:"t"`
}

// getPlaces is the segments with the byte range each holds, the first beginning at offset.
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

// getNextStart is where to read on after a range that ends at to, nil at the end of the text.
func getNextStart(to, size int) *int {
	if to >= size {
		return nil
	}
	return &to
}

// findPage is the byte range of the segments of a page.
func findPage(page int, places []place) (from, to int, err error) {
	first, last := 0, 0
	from, to = -1, -1
	for _, p := range places {
		if p.selector.Page == 0 {
			continue
		}
		if first == 0 || p.selector.Page < first {
			first = p.selector.Page
		}
		last = max(last, p.selector.Page)
		if p.selector.Page == page {
			if from < 0 {
				from = p.start
			}
			to = p.end
		}
	}
	switch {
	case first == 0:
		return 0, 0, errors.New("the text has no pages; read it by start and length")
	case from < 0:
		return 0, 0, fmt.Errorf("the text has no page %d: its pages are %d to %d", page, first, last)
	}
	return from, to, nil
}

// getLocations is the places the range [from, to) covers, in order. Consecutive segments of one
// page are one location.
func getLocations(places []place, from, to int) []LibraryLocation {
	var out []LibraryLocation
	for _, p := range places {
		start, end := max(p.start, from), min(p.end, to)
		if start >= end {
			continue
		}
		s := p.selector
		location := LibraryLocation{Start: start, Length: end - start, Page: s.Page, CFI: s.CFI, FromSeconds: 0, ToSeconds: 0}
		if len(s.T) == 2 {
			location.FromSeconds, location.ToSeconds = s.T[0], s.T[1]
		}
		if s.Page == 0 && s.CFI == "" && len(s.T) != 2 {
			continue
		}
		if last := len(out) - 1; last >= 0 && s.Page > 0 && out[last].Page == s.Page {
			out[last].Length = end - out[last].Start
			continue
		}
		out = append(out, location)
	}
	return out
}

// checkReadRange refuses a range a call cannot mean: a negative number, or a page given with a start.
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

// readEmptyPage is the answer for a page whose segments hold no text: no text, at the start of the page.
func (t *Tools) readEmptyPage(ctx context.Context, path string, live domain.TextVersion, start int) (LibraryReadOutput, error) {
	read, err := t.texts.ReadTextRange(ctx, path, live.GetID(), start, 1)
	if err != nil {
		return LibraryReadOutput{}, err
	}
	return LibraryReadOutput{
		Path: path, Kind: string(live.Kind), Version: string(live.GetID()), Text: "", Start: start, Length: 0, Size: read.Size,
		NextStart: getNextStart(start, read.Size), Locations: nil,
	}, nil
}
