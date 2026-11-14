// Package librarywire calls the library use case with the messages of the library tools.
package librarywire

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/library"
)

// SearchLibrary answers library_search.
func SearchLibrary(ctx context.Context, u *library.UseCase, in api.LibrarySearchInput) (api.LibrarySearchOutput, error) {
	matches, err := u.Search(ctx, library.Query{Query: in.Query, Limit: in.Limit, Kinds: in.Kinds, Folder: in.Folder})
	if err != nil {
		return api.LibrarySearchOutput{}, err
	}
	out := api.LibrarySearchOutput{Query: in.Query, Matches: make([]api.LibraryMatch, 0, len(matches))}
	for _, match := range matches {
		out.Matches = append(out.Matches, api.LibraryMatch{
			Path:   match.Path,
			Title:  match.Title,
			Place:  newPlace(match.Place),
			Text:   match.Text,
			Start:  match.Start,
			Length: match.Length,
			Line:   match.Line,
			URI:    match.URI,
		})
	}
	return out, nil
}

// ReadLibrary answers library_read.
func ReadLibrary(ctx context.Context, u *library.UseCase, in api.LibraryReadInput) (api.LibraryReadOutput, error) {
	read, err := u.Read(ctx, library.ReadRequest{Path: in.Path, Page: in.Page, Start: in.Start, Length: in.Length})
	if err != nil {
		return api.LibraryReadOutput{}, err
	}
	var ranges []api.LibraryRange
	for _, one := range read.Ranges {
		ranges = append(ranges, api.LibraryRange{Start: one.Start, Length: one.Length, Place: newPlace(one.Place)})
	}
	return api.LibraryReadOutput{
		Path: read.Path, Kind: read.Kind, Version: read.Version, Text: read.Text, Start: read.Start,
		Length: len(read.Text), Size: read.Size, NextStart: read.NextStart, Ranges: ranges,
	}, nil
}

func newPlace(place *library.Place) *api.Place {
	if place == nil {
		return nil
	}
	return &api.Place{
		Page: place.Page, PageCount: 0, PositionSeconds: place.PositionSeconds, EndSeconds: place.EndSeconds,
		CFI: place.CFI, Chapter: place.Chapter,
	}
}
