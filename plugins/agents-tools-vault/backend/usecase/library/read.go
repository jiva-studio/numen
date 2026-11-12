package library

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/textplace"
)

const (
	readLengthDefault = 20000
	readLengthMax     = 50000
)

// Read returns a page or a byte range of the text of a file: its live text version, else the file itself
// when it is text.
func (u *UseCase) Read(ctx context.Context, in api.LibraryReadInput) (api.LibraryReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.LibraryReadOutput{}, errors.New("path is required")
	}
	if err := textplace.CheckRead(in.Page, in.Start, in.Length); err != nil {
		return api.LibraryReadOutput{}, err
	}
	if u.texts == nil {
		return api.LibraryReadOutput{}, errNoLibrary
	}
	listing, err := u.versions.ListVersions(ctx, in.Path)
	if err != nil {
		return api.LibraryReadOutput{}, fmt.Errorf("list the text versions of %s: %w", in.Path, err)
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
		return api.LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	return out, nil
}

func getReadLength(in api.LibraryReadInput) int {
	if in.Length > 0 {
		return min(in.Length, readLengthMax)
	}
	return readLengthDefault
}

func (u *UseCase) readPlainText(ctx context.Context, in api.LibraryReadInput) (api.LibraryReadOutput, error) {
	read, err := u.files.ReadRange(ctx, in.Path, in.Start, getReadLength(in))
	if errors.Is(err, host.ErrNotText) {
		return api.LibraryReadOutput{}, fmt.Errorf("%s has no text version; a recognition or a transcription creates one", in.Path)
	}
	if err != nil {
		return api.LibraryReadOutput{}, fmt.Errorf("read %s: %w", in.Path, err)
	}
	if in.Page > 0 {
		return api.LibraryReadOutput{}, fmt.Errorf("read the text of %s: a text file has no pages; read it by start and length", in.Path)
	}
	size := int(read.Stat.Size)
	if err := textplace.CheckStart(in.Start, size); err != nil {
		return api.LibraryReadOutput{}, fmt.Errorf("read the text of %s: %w", in.Path, err)
	}
	end := read.Start + len(read.Text)
	return api.LibraryReadOutput{
		Path: in.Path, Kind: "", Version: "", Text: read.Text, Start: read.Start, Length: len(read.Text), Size: size,
		NextStart: textplace.GetNextStart(end, size), Ranges: nil,
	}, nil
}

func (u *UseCase) readVersion(ctx context.Context, in api.LibraryReadInput, live domain.TextVersion) (api.LibraryReadOutput, error) {
	start, length := in.Start, getReadLength(in)
	if in.Page > 0 {
		pageStart, pageEnd, err := u.findPage(ctx, in.Path, live.GetID(), in.Page)
		if err != nil {
			return api.LibraryReadOutput{}, err
		}
		if pageEnd == pageStart {
			return u.readEmptyPage(ctx, in.Path, live, pageStart)
		}
		start, length = pageStart, min(pageEnd-pageStart, length)
	}
	read, err := u.texts.ReadTextRange(ctx, in.Path, live.GetID(), start, length)
	if err != nil {
		return api.LibraryReadOutput{}, err
	}
	if err := textplace.CheckStart(start, read.Size); err != nil {
		return api.LibraryReadOutput{}, err
	}
	spans, err := textplace.GetSpans(read.Segments, read.Start)
	if err != nil {
		return api.LibraryReadOutput{}, err
	}
	var text strings.Builder
	for _, segment := range read.Segments {
		text.WriteString(segment.Text)
	}
	end := read.Start + text.Len()
	return api.LibraryReadOutput{
		Path: in.Path, Kind: string(live.Kind), Version: string(live.GetID()),
		Text: text.String(), Start: read.Start, Length: text.Len(), Size: read.Size,
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

func (u *UseCase) readEmptyPage(ctx context.Context, path string, live domain.TextVersion, start int) (api.LibraryReadOutput, error) {
	read, err := u.texts.ReadTextRange(ctx, path, live.GetID(), start, 1)
	if err != nil {
		return api.LibraryReadOutput{}, err
	}
	return api.LibraryReadOutput{
		Path: path, Kind: string(live.Kind), Version: string(live.GetID()), Text: "", Start: start, Length: 0, Size: read.Size,
		NextStart: textplace.GetNextStart(start, read.Size), Ranges: nil,
	}, nil
}

func newRanges(ranges []textplace.Range) []api.LibraryRange {
	var out []api.LibraryRange
	for _, one := range ranges {
		out = append(out, api.LibraryRange{Start: one.Start, Length: one.Length, Place: newPlace(one.Selector)})
	}
	return out
}
