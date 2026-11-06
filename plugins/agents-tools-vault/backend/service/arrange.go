package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

const (
	defaultNoteListLimit = 100
	maxNoteListLimit     = 500
	maxNoteTitlesPaths   = 50
)

// ListedNote is the path and title of a note.
type ListedNote struct {
	Path  string `json:"path" jsonschema:"path of the note relative to the vault folder"`
	Title string `json:"title" jsonschema:"title of the note"`
}

// NoteListInput is the input of the note_list tool.
type NoteListInput struct {
	Folder string `json:"folder,omitempty" jsonschema:"list only the notes in this folder and its subfolders, relative to the vault folder; the whole vault when left out"`
	Limit  int    `json:"limit,omitempty" jsonschema:"how many notes to return (default: 100, max: 500)"`
	Offset int    `json:"offset,omitempty" jsonschema:"how many notes to skip; the nextOffset of the previous result continues the list"`
}

// NoteListOutput is the output of the note_list tool.
type NoteListOutput struct {
	Notes      []ListedNote `json:"notes" jsonschema:"the notes of this page, ordered by path"`
	Total      int          `json:"total" jsonschema:"number of notes that match, before limit and offset"`
	NextOffset int          `json:"nextOffset,omitempty" jsonschema:"the offset of the next page; absent on the last page"`
}

// NoteTitlesInput is the input of the note_titles tool.
type NoteTitlesInput struct {
	Paths []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
}

// NoteTitlesOutput is the output of the note_titles tool.
type NoteTitlesOutput struct {
	Notes   []ListedNote `json:"notes" jsonschema:"the notes found, with their titles"`
	Missing []string     `json:"missing" jsonschema:"the paths that name no note, in the order given"`
}

// NoteResolveInput is the input of the note_resolve tool.
type NoteResolveInput struct {
	Name string `json:"name" jsonschema:"file name of a note without its extension, as a wikilink writes it"`
}

// NoteResolveOutput is the output of the note_resolve tool.
type NoteResolveOutput struct {
	Paths []string `json:"paths" jsonschema:"paths of every note with that name; empty when none has it"`
}

func (t *Tools) addNoteList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "note_list",
		Title:       "List Notes",
		Description: "List the paths and titles of the notes in the vault or in one folder, ordered by path. The result is paged with limit and offset and gives the total count.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listNotes))
}

func (t *Tools) addNoteTitles(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "note_titles",
		Title:       "Get Note Titles",
		Description: "Get the titles of up to 50 notes by their paths. Paths that name no note are returned in missing.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).getNoteTitles))
}

func (t *Tools) addNoteResolve(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "note_resolve",
		Title:       "Resolve Note Name",
		Description: "Get the paths of every note with the given file name, which is what a wikilink names. More than one path means the name is ambiguous.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).resolveNoteName))
}

func (t *Tools) listNotes(ctx context.Context, in NoteListInput) (NoteListOutput, error) {
	limit := in.Limit
	switch {
	case limit < 0:
		return NoteListOutput{}, errors.New("limit must not be negative")
	case limit == 0:
		limit = defaultNoteListLimit
	case limit > maxNoteListLimit:
		return NoteListOutput{}, fmt.Errorf("limit %d is above the maximum of %d", limit, maxNoteListLimit)
	}
	if in.Offset < 0 {
		return NoteListOutput{}, errors.New("offset must not be negative")
	}

	paths, err := t.vault.GetPathsOfType(ctx, domain.TypeNote)
	if err != nil {
		return NoteListOutput{}, fmt.Errorf("list notes: %w", err)
	}
	if folder := strings.Trim(in.Folder, "/"); folder != "" {
		prefix := folder + "/"
		paths = slices.DeleteFunc(paths, func(p string) bool { return !strings.HasPrefix(p, prefix) })
	}
	slices.Sort(paths)

	total := len(paths)
	start := min(in.Offset, total)
	end := min(start+limit, total)
	page := paths[start:end]
	refs, err := t.vault.GetNotes(ctx, page)
	if err != nil {
		return NoteListOutput{}, fmt.Errorf("get note titles: %w", err)
	}
	notes := make([]ListedNote, 0, len(page))
	for _, p := range page {
		notes = append(notes, ListedNote{Path: p, Title: refs[p].Title})
	}
	next := 0
	if end < total {
		next = end
	}
	return NoteListOutput{Notes: notes, Total: total, NextOffset: next}, nil
}

func (t *Tools) getNoteTitles(ctx context.Context, in NoteTitlesInput) (NoteTitlesOutput, error) {
	if len(in.Paths) > maxNoteTitlesPaths {
		return NoteTitlesOutput{}, fmt.Errorf("%d paths given, at most %d are allowed", len(in.Paths), maxNoteTitlesPaths)
	}
	refs, err := t.vault.GetNotes(ctx, in.Paths)
	if err != nil {
		return NoteTitlesOutput{}, fmt.Errorf("get note titles: %w", err)
	}
	out := NoteTitlesOutput{Notes: make([]ListedNote, 0, len(refs)), Missing: []string{}}
	for _, p := range in.Paths {
		ref, ok := refs[p]
		if !ok {
			out.Missing = append(out.Missing, p)
			continue
		}
		out.Notes = append(out.Notes, ListedNote{Path: p, Title: ref.Title})
	}
	return out, nil
}

func (t *Tools) resolveNoteName(ctx context.Context, in NoteResolveInput) (NoteResolveOutput, error) {
	if strings.TrimSpace(in.Name) == "" {
		return NoteResolveOutput{}, errors.New("name required")
	}
	paths, err := t.vault.GetNamedPaths(ctx, in.Name)
	if err != nil {
		return NoteResolveOutput{}, fmt.Errorf("resolve note name %s: %w", in.Name, err)
	}
	if paths == nil {
		paths = []string{}
	}
	return NoteResolveOutput{Paths: paths}, nil
}
