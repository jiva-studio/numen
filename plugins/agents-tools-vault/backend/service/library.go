package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// LibrarySearchInput is the input of the library_search tool.
type LibrarySearchInput struct {
	Query      string   `json:"query" jsonschema:"words to look for, in the words of the text itself"`
	Limit      int      `json:"limit,omitempty" jsonschema:"how many passages to return (default: 20, max: 100)"`
	Kinds      []string `json:"kinds,omitempty" jsonschema:"which sorts of file to look in: note for markdown notes, file for every other file (books, PDFs, recordings, scans); all of them when left out"`
	PathPrefix string   `json:"path_prefix,omitempty" jsonschema:"only look in the files whose path starts with this, relative to the vault folder, such as library/"`
}

// LibraryMatch is a passage that answers a library search.
type LibraryMatch struct {
	Source   string `json:"source" jsonschema:"the file the text is read from, relative to the vault folder"`
	Title    string `json:"title" jsonschema:"the title of the note, else the name of the file"`
	Location string `json:"location,omitempty" jsonschema:"the position in the source own numbering, such as a page, a chapter or a time; absent when the format offers none"`
	Text     string `json:"text" jsonschema:"the words around the hit, cut to a size; it ends where it was cut, often mid-sentence"`
	Start    int    `json:"start" jsonschema:"where the passage begins in the source's text, in bytes"`
	Length   int    `json:"length" jsonschema:"how long the passage is, in bytes"`
	Line     int    `json:"line,omitempty" jsonschema:"the line of the hit in a note, counted from the first line of its prose"`
	At       string `json:"at" jsonschema:"the address of this passage; write it into your answer as a markdown link where you speak about the passage, so that the person can go to it"`
}

// LibrarySearchOutput is the output of the library_search tool.
type LibrarySearchOutput struct {
	Query   string         `json:"query"`
	Matches []LibraryMatch `json:"matches"`
}

// LibraryReadInput is the input of the library_read tool.
type LibraryReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault folder"`
	Page   int    `json:"page,omitempty" jsonschema:"read this page, counted from 1; for a source whose text has pages"`
	Start  int    `json:"start,omitempty" jsonschema:"byte offset in the text to start reading from; the next of the previous answer continues it"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes of text to read; at most 50000, and 20000 when left out"`
}

// LibraryLocation is a stretch of the text and the place the source numbers it by.
type LibraryLocation struct {
	Start       int     `json:"start" jsonschema:"where the stretch begins in the text, in bytes"`
	Length      int     `json:"length" jsonschema:"how long the stretch is, in bytes"`
	Page        int     `json:"page,omitempty" jsonschema:"the page, counted from 1"`
	CFI         string  `json:"cfi,omitempty" jsonschema:"the position in a book"`
	FromSeconds float64 `json:"from_seconds,omitempty" jsonschema:"where in the recording the stretch is spoken"`
	ToSeconds   float64 `json:"to_seconds,omitempty" jsonschema:"where in the recording the stretch ends"`
}

// LibraryReadOutput is the output of the library_read tool.
type LibraryReadOutput struct {
	Path      string            `json:"path"`
	Kind      string            `json:"kind,omitempty" jsonschema:"what the text is, as the plugin that made it names it; absent for a text file"`
	Version   string            `json:"version,omitempty" jsonschema:"the text version read"`
	Text      string            `json:"text"`
	Start     int               `json:"start" jsonschema:"where the text returned begins in the whole text, in bytes"`
	Length    int               `json:"length" jsonschema:"how many bytes were returned"`
	Size      int               `json:"size" jsonschema:"how long the whole text is, in bytes"`
	Next      *int              `json:"next,omitempty" jsonschema:"the start to read next; absent at the end of the text"`
	Locations []LibraryLocation `json:"locations,omitempty" jsonschema:"the places of the source that the returned text covers, in order"`
}

// WithLibrary gives the tools the text versions and the search index of the vaults, which the tools
// that read the library need.
func (t *Tools) WithLibrary(texts host.Texts, search host.Search) *Tools {
	t.texts, t.search = texts, search
	return t
}

const (
	searchedLimitDefault = 20
	searchedLimitMax     = 100
	readLengthDefault    = 20000
	readLengthMax        = 50000
	// sampleBytes is how many bytes of a file decide whether it is text.
	sampleBytes = 8192
)

var (
	errNoLibrary = errors.New("the plugin was given neither text versions nor a search index")
	// noTextVersionHelp names what makes the text version of a file.
	noTextVersionHelp = "The reader plugin of the format makes one for a PDF or a book, " +
		"the recognition makes one for a scan, and the transcription makes one for a recording."
)

func (t *Tools) addLibrarySearch(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "library_search",
		Title: "Search the library",
		Description: "Search everything the vault holds — the notes, and the books, PDFs, scans and " +
			"recordings filed beside them — for the words typed and for what they mean, in the index " +
			"of their text. Returns passages, best first: the text around each hit, the file it was " +
			"read from, where it sits (a page, a chapter, a time) and its address. A file answers with " +
			"at most three passages, so a long book does not take the answer. " +
			"note_search finds only notes, by their title, path and text; use library_search for the " +
			"text of a book, a PDF or the transcript of a recording. " +
			"A passage is a window cut to a size and it ends where it was cut, which is mid-sentence " +
			"as often as not: read on with library_read before concluding that a source says nothing " +
			"about something. When the person asks about a book, pass kinds [\"file\"]: a vault holds " +
			"far more notes than books, and a search told to look everywhere answers with notes.",
		Effect:  tools.ReadOnly,
		Step:    tools.Search,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).searchLibrary))
}

func (t *Tools) searchLibrary(ctx context.Context, in LibrarySearchInput) (LibrarySearchOutput, error) {
	out := LibrarySearchOutput{Query: in.Query, Matches: []LibraryMatch{}}
	if strings.TrimSpace(in.Query) == "" {
		return out, nil
	}
	kinds, err := parseKinds(in.Kinds)
	if err != nil {
		return out, err
	}
	if t.search == nil {
		return out, errNoLibrary
	}
	limit := searchedLimitDefault
	if in.Limit > 0 {
		limit = min(in.Limit, searchedLimitMax)
	}
	found, err := t.search.SearchPassages(ctx, host.PassageQuery{
		Query: in.Query, Limit: limit, Kinds: kinds, PathPrefix: in.PathPrefix,
	})
	if err != nil {
		return out, fmt.Errorf("search the library: %w", err)
	}
	titles := t.getTitles(ctx, found)
	for _, passage := range found {
		out.Matches = append(out.Matches, LibraryMatch{
			Source:   passage.Path,
			Title:    titles[passage.Path],
			Location: passage.Location,
			Text:     passage.Text,
			Start:    passage.Start,
			Length:   passage.Length,
			Line:     passage.Line,
			At:       formatAddress(passage.Path, passage.Start, passage.Length),
		})
	}
	return out, nil
}

// parseKinds is the kinds of source a question names: note for a markdown note, file for any other.
func parseKinds(named []string) ([]domain.SourceKind, error) {
	kinds := make([]domain.SourceKind, 0, len(named))
	for _, one := range named {
		kind := domain.SourceKind(one)
		if kind != domain.KindNote && kind != domain.KindFile {
			return nil, fmt.Errorf("%q is not a sort of file a search reaches: try %q or %q", one, domain.KindNote, domain.KindFile)
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

// getTitles is the title of each source: a note's own, else the name of the file without its extension.
func (t *Tools) getTitles(ctx context.Context, found []host.Passage) map[string]string {
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
		return titles
	}
	for notePath, ref := range refs {
		if ref.Title != "" {
			titles[notePath] = ref.Title
		}
	}
	return titles
}

// formatAddress is where a passage is reached, as the window opens one.
func formatAddress(source string, start, length int) string {
	return fmt.Sprintf("numen:%s?start=%d&length=%d", url.PathEscape(source), start, length)
}

func (t *Tools) addLibraryRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "library_read",
		Title: "Read the text of a file",
		Description: "Read the text of one file of the vault from its text version: the pages of a PDF, " +
			"the chapters of a book, the transcript of a recording, the recognised text of a scan. " +
			"Give a page, or a start and a length in bytes; the answer names the places the text " +
			"covers, how long the whole text is, and where to start to read on. " +
			"A text file is read as it is. A file with no text version returns an error that names " +
			"what makes one. file_read reads the bytes of a text file and refuses a binary; " +
			"library_search finds the passage to read.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).readLibrary))
}

func (t *Tools) readLibrary(ctx context.Context, in LibraryReadInput) (LibraryReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return LibraryReadOutput{}, errors.New("path required")
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
	text, err := host.ReadWholeText(ctx, t.texts, live)
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	segments := getPlaces(text.GetSegments())
	from, to, err := pickRange(in, text.Text, segments)
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	out := LibraryReadOutput{
		Path: in.Path, Kind: string(live.Kind), Version: string(live.GetID()),
		Text: text.Text[from:to], Start: from, Length: to - from, Size: len(text.Text),
		Locations: getLocations(segments, from, to),
	}
	if to < len(text.Text) {
		out.Next = &to
	}
	return out, nil
}

// readPlainText reads a file that has no text version: a text file is its own text, and any other
// file has no text to read.
func (t *Tools) readPlainText(ctx context.Context, in LibraryReadInput) (LibraryReadOutput, error) {
	data, _, err := t.vault.Read(ctx, in.Path)
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read %s: %w", in.Path, err)
	}
	if !isText(data) {
		return LibraryReadOutput{}, fmt.Errorf("%s has no text version yet. %s", in.Path, noTextVersionHelp)
	}
	if in.Page > 0 {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: a text file has no pages; read it by start and length", in.Path)
	}
	text := string(data)
	from, to, err := pickRange(in, text, nil)
	if err != nil {
		return LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	out := LibraryReadOutput{Path: in.Path, Text: text[from:to], Start: from, Length: to - from, Size: len(text)}
	if to < len(text) {
		out.Next = &to
	}
	return out, nil
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

// getPlaces is the segments with the byte range each holds.
func getPlaces(segments []domain.TextSegment) []place {
	places := make([]place, 0, len(segments))
	offset := 0
	for _, segment := range segments {
		var s selector
		if segment.Selector != "" {
			if err := json.Unmarshal([]byte(segment.Selector), &s); err != nil {
				s = selector{}
			}
		}
		places = append(places, place{start: offset, end: offset + len(segment.Text), selector: s})
		offset += len(segment.Text)
	}
	return places
}

// pickRange is the byte range of the text a call asks for, on character boundaries.
func pickRange(in LibraryReadInput, text string, places []place) (from, to int, err error) {
	length := readLengthDefault
	if in.Length > 0 {
		length = min(in.Length, readLengthMax)
	}
	from = max(in.Start, 0)
	end := len(text)
	if in.Page > 0 {
		from, end, err = findPage(in.Page, places)
		if err != nil {
			return 0, 0, err
		}
	}
	if from > len(text) || (from == len(text) && len(text) > 0) {
		return 0, 0, fmt.Errorf("start %d is past the end of the text, which is %d bytes long", from, len(text))
	}
	for from < len(text) && !utf8.RuneStart(text[from]) {
		from++
	}
	to = min(end, from+length)
	for to < len(text) && to > from && !utf8.RuneStart(text[to]) {
		to--
	}
	if to == from && from < len(text) {
		_, width := utf8.DecodeRuneInString(text[from:])
		to = from + width
	}
	return from, to, nil
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
		location := LibraryLocation{Start: start, Length: end - start, Page: s.Page, CFI: s.CFI}
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

// isText reports whether the start of a file reads as text: valid UTF-8 without a zero byte.
func isText(data []byte) bool {
	sample := data[:min(len(data), sampleBytes)]
	if bytes.IndexByte(sample, 0) >= 0 {
		return false
	}
	if len(data) > len(sample) {
		for i := 0; i < utf8.UTFMax && len(sample) > 0 && !utf8.Valid(sample); i++ {
			sample = sample[:len(sample)-1]
		}
	}
	return utf8.Valid(sample)
}
