package library

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/textplace"
)

const (
	readLengthDefault = 20000
	readLengthMax     = 50000
)

// ReadRequest names the page or the byte range of the text of a file to read.
type ReadRequest struct {
	Path   string
	Page   int
	Start  int
	Length int
}

// Range is a range of the text and the place in the source where it starts.
type Range struct {
	Start  int
	Length int
	Place  *Place
}

// Text is the text read of a file. Kind and Version are empty for a text file.
type Text struct {
	Path      string
	Kind      string
	Version   string
	Text      string
	Start     int
	Size      int
	NextStart *int
	Ranges    []Range
}

// Read returns a page or a byte range of the text of a file: its live text version, else the file itself
// when it is text.
func (u *UseCase) Read(ctx context.Context, in ReadRequest) (Text, error) {
	if strings.TrimSpace(in.Path) == "" {
		return Text{}, errors.New("path is required")
	}
	if err := textplace.CheckRead(in.Page, in.Start, in.Length); err != nil {
		return Text{}, err
	}
	if u.texts == nil {
		return Text{}, errNoLibrary
	}
	listing, err := u.versions.ListVersions(ctx, in.Path)
	if err != nil {
		return Text{}, fmt.Errorf("list the text versions of %s: %w", in.Path, err)
	}
	if listing.Live == "" {
		return u.readPlainText(ctx, in)
	}
	var live domain.TextVersion
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			live = version
		}
	}
	out, err := u.readVersion(ctx, in, live)
	if err != nil {
		return Text{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	return out, nil
}

func getReadLength(in ReadRequest) int {
	if in.Length > 0 {
		return min(in.Length, readLengthMax)
	}
	return readLengthDefault
}

func (u *UseCase) readPlainText(ctx context.Context, in ReadRequest) (Text, error) {
	read, err := u.files.ReadRange(ctx, in.Path, in.Start, getReadLength(in))
	if errors.Is(err, host.ErrNotText) {
		return Text{}, fmt.Errorf("%s has no text version; a recognition or a transcription creates one", in.Path)
	}
	if err != nil {
		return Text{}, fmt.Errorf("read %s: %w", in.Path, err)
	}
	if in.Page > 0 {
		return Text{}, fmt.Errorf("read the text of %s: a text file has no pages; read it by start and length", in.Path)
	}
	size := int(read.Stat.Size)
	if err := textplace.CheckStart(in.Start, size); err != nil {
		return Text{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	end := read.Start + len(read.Text)
	return Text{
		Path: in.Path, Kind: "", Version: "", Text: read.Text, Start: read.Start, Size: size,
		NextStart: textplace.GetNextStart(end, size), Ranges: nil,
	}, nil
}

func (u *UseCase) readVersion(ctx context.Context, in ReadRequest, live domain.TextVersion) (Text, error) {
	start, length := in.Start, getReadLength(in)
	if in.Page > 0 {
		pageStart, pageEnd, err := u.findPage(ctx, in.Path, live.GetID(), in.Page)
		if err != nil {
			return Text{}, err
		}
		if pageEnd == pageStart {
			return u.readEmptyPage(ctx, in.Path, live, pageStart)
		}
		start, length = pageStart, min(pageEnd-pageStart, length)
	}
	read, err := u.texts.ReadTextRange(ctx, in.Path, live.GetID(), start, length)
	if err != nil {
		return Text{}, err
	}
	if err := textplace.CheckStart(start, read.Size); err != nil {
		return Text{}, err
	}
	spans, err := textplace.GetSpans(read.Segments, read.Start)
	if err != nil {
		return Text{}, err
	}
	var text strings.Builder
	for _, segment := range read.Segments {
		text.WriteString(segment.Text)
	}
	end := read.Start + text.Len()
	return Text{
		Path: in.Path, Kind: string(live.Kind), Version: string(live.GetID()),
		Text: text.String(), Start: read.Start, Size: read.Size,
		NextStart: textplace.GetNextStart(end, read.Size), Ranges: newRanges(textplace.GetRanges(spans, read.Start, end)),
	}, nil
}

func (u *UseCase) findPage(ctx context.Context, path string, id domain.TextVersionID, page int) (start, end int, err error) {
	var spans []textplace.Span
	cursor, offset := "", 0
	for {
		read, err := u.texts.ReadText(ctx, path, id, cursor)
		if err != nil {
			return 0, 0, err
		}
		more, err := textplace.GetSpans(read.Segments, offset)
		if err != nil {
			return 0, 0, err
		}
		spans = append(spans, more...)
		if len(spans) > 0 {
			offset = spans[len(spans)-1].End
		}
		if read.NextCursor == "" || textplace.IsPastPage(spans, page) {
			return textplace.FindPage(page, spans)
		}
		if read.NextCursor == cursor {
			return 0, 0, fmt.Errorf("the host returned the page at %q twice", cursor)
		}
		cursor = read.NextCursor
	}
}

func (u *UseCase) readEmptyPage(ctx context.Context, path string, live domain.TextVersion, start int) (Text, error) {
	read, err := u.texts.ReadTextRange(ctx, path, live.GetID(), start, 1)
	if err != nil {
		return Text{}, err
	}
	return Text{
		Path: path, Kind: string(live.Kind), Version: string(live.GetID()), Text: "", Start: start, Size: read.Size,
		NextStart: textplace.GetNextStart(start, read.Size), Ranges: nil,
	}, nil
}

func newRanges(ranges []textplace.Range) []Range {
	var out []Range
	for _, one := range ranges {
		out = append(out, Range{Start: one.Start, Length: one.Length, Place: newPlace(one.Selector)})
	}
	return out
}
