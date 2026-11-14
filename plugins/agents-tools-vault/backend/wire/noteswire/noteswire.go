// Package noteswire calls the notes use case with the messages of the note tools.
package noteswire

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/notes"
)

// ReadNote answers note_read.
func ReadNote(ctx context.Context, u *notes.UseCase, in api.NoteReadInput) (api.NoteReadOutput, error) {
	note, err := u.Read(ctx, in.Path)
	if err != nil {
		return api.NoteReadOutput{}, err
	}
	return api.NoteReadOutput{
		Path:        note.Path,
		Title:       note.Title,
		Text:        note.Text,
		Body:        note.Body,
		Fingerprint: note.Fingerprint,
		Frontmatter: note.Frontmatter,
	}, nil
}

// WriteNote answers note_write.
func WriteNote(ctx context.Context, u *notes.UseCase, in api.NoteWriteInput) (api.NoteWriteOutput, error) {
	written, err := u.Write(ctx, in.Path, in.Text, in.Fingerprint)
	if err != nil {
		return api.NoteWriteOutput{}, err
	}
	return api.NoteWriteOutput{Path: written.Path, Fingerprint: written.Fingerprint, Bytes: written.Bytes}, nil
}

// ListHeadings answers note_heading_list.
func ListHeadings(ctx context.Context, u *notes.UseCase, in api.NoteHeadingListInput) (api.NoteHeadingListOutput, error) {
	path, headings, err := u.ListHeadings(ctx, in.Path)
	if err != nil {
		return api.NoteHeadingListOutput{}, err
	}
	items := make([]api.HeadingItem, 0, len(headings))
	for _, heading := range headings {
		items = append(items, api.HeadingItem{Level: heading.Level, Text: heading.Text, Line: heading.Line, Start: heading.Start})
	}
	return api.NoteHeadingListOutput{Path: path, Headings: items}, nil
}

// SearchNotes answers note_search.
func SearchNotes(ctx context.Context, u *notes.UseCase, in api.NoteSearchInput) (api.NoteSearchOutput, error) {
	matches, total, err := u.Search(ctx, in.Query, in.Limit)
	if err != nil {
		return api.NoteSearchOutput{}, err
	}
	var out []api.SearchMatch
	if matches != nil {
		out = make([]api.SearchMatch, 0, len(matches))
	}
	for _, match := range matches {
		out = append(out, api.SearchMatch{Path: match.Path, Title: match.Title, Snippet: match.Snippet, Line: match.Line, Score: match.Score})
	}
	return api.NoteSearchOutput{Query: in.Query, Matches: out, Total: total}, nil
}

// ListNotes answers note_list.
func ListNotes(ctx context.Context, u *notes.UseCase, in api.NoteListInput) (api.NoteListOutput, error) {
	page, err := u.List(ctx, in.Folder, in.Limit, in.Offset)
	if err != nil {
		return api.NoteListOutput{}, err
	}
	entries := make([]api.NoteEntry, 0, len(page.Notes))
	for _, note := range page.Notes {
		entries = append(entries, api.NoteEntry{Path: note.Path, Title: note.Title})
	}
	return api.NoteListOutput{Notes: entries, Total: page.Total, NextOffset: page.NextOffset}, nil
}

// ListTitles answers note_title_list.
func ListTitles(ctx context.Context, u *notes.UseCase, in api.NoteTitleListInput) (api.NoteTitleListOutput, error) {
	found, missing, err := u.ListTitles(ctx, in.Paths)
	if err != nil {
		return api.NoteTitleListOutput{}, err
	}
	out := api.NoteTitleListOutput{Notes: make([]api.NoteTitle, 0, len(found)), MissingPaths: missing}
	for _, note := range found {
		out.Notes = append(out.Notes, api.NoteTitle{Path: note.Path, Title: note.Title, ID: note.ID})
	}
	return out, nil
}

// ResolveName answers note_resolve.
func ResolveName(ctx context.Context, u *notes.UseCase, in api.NoteResolveInput) (api.NoteResolveOutput, error) {
	paths, err := u.ResolveName(ctx, in.Name)
	if err != nil {
		return api.NoteResolveOutput{}, err
	}
	return api.NoteResolveOutput{Paths: paths}, nil
}
