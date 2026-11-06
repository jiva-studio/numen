package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"golang.org/x/text/unicode/norm"
)

const (
	defaultNoteListLimit = 100
	maxNoteListLimit     = 500
	maxBatch             = 50
)

// NoteEntry is the path and title of a note.
type NoteEntry struct {
	Path  string `json:"path" jsonschema:"path of the note relative to the vault folder"`
	Title string `json:"title" jsonschema:"title of the note"`
}

// NoteTitle is the path, title and identifier of a note.
type NoteTitle struct {
	Path  string `json:"path" jsonschema:"path of the note relative to the vault folder"`
	Title string `json:"title" jsonschema:"title of the note"`
	ID    string `json:"id,omitempty" jsonschema:"identifier written in the note, which [[note://<identifier>]] names; absent for a note without one"`
}

// NoteListInput is the input of the note_list tool.
type NoteListInput struct {
	Folder string `json:"folder,omitempty" jsonschema:"list only the notes in this folder and its subfolders, relative to the vault folder; the whole vault when left out"`
	Limit  int    `json:"limit,omitempty" jsonschema:"how many notes to return (default: 100, max: 500)"`
	Offset int    `json:"offset,omitempty" jsonschema:"how many notes to skip; the nextOffset of the previous result continues the list"`
}

// NoteListOutput is the output of the note_list tool.
type NoteListOutput struct {
	Notes      []NoteEntry `json:"notes" jsonschema:"the notes of this page, ordered by path"`
	Total      int         `json:"total" jsonschema:"number of notes that match, before limit and offset"`
	NextOffset int         `json:"nextOffset,omitempty" jsonschema:"the offset of the next page; absent on the last page"`
}

// NoteTitleListInput is the input of the note_title_list tool.
type NoteTitleListInput struct {
	Paths []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
}

// NoteTitleListOutput is the output of the note_title_list tool.
type NoteTitleListOutput struct {
	Notes        []NoteTitle `json:"notes" jsonschema:"the notes found, with their titles"`
	MissingPaths []string    `json:"missingPaths" jsonschema:"the paths that name no note, in the order given"`
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
	return addTool(registry, tools.Spec{
		Step:         tools.ByEffect,
		Name:         "note_list",
		Title:        "List Notes",
		Description:  "List the paths and titles of the notes in the vault or in one folder, ordered by path. The result is paged with limit and offset and gives the total count.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{noteListSentence},
	}, inVault(t, (*Tools).listNotes))
}

func (t *Tools) addNoteTitleList(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:         tools.ByEffect,
		Name:         "note_title_list",
		Title:        "Get Note Titles",
		Description:  "Get the titles and identifiers of up to 50 notes by their paths. Paths that name no note are returned in missingPaths.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{noteTitleListSentence},
	}, inVault(t, (*Tools).listNoteTitles))
}

func (t *Tools) addNoteResolve(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:         tools.ByEffect,
		Name:         "note_resolve",
		Title:        "Resolve Note Name",
		Description:  "Get the paths of every note with the given file name, which is what a wikilink names. More than one path means the name is ambiguous.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{noteResolveSentence},
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
		return NoteListOutput{}, err
	}
	if prefix := getFolderPrefix(in.Folder); prefix != "" {
		paths = slices.DeleteFunc(paths, func(p string) bool { return !strings.HasPrefix(p, prefix) })
	}
	slices.Sort(paths)

	total := len(paths)
	start := min(in.Offset, total)
	end := min(start+limit, total)
	page := paths[start:end]
	refs, err := t.vault.GetNotes(ctx, page)
	if err != nil {
		return NoteListOutput{}, err
	}
	notes := make([]NoteEntry, 0, len(page))
	for _, p := range page {
		notes = append(notes, NoteEntry{Path: p, Title: refs[p].Title})
	}
	next := 0
	if end < total {
		next = end
	}
	return NoteListOutput{Notes: notes, Total: total, NextOffset: next}, nil
}

func (t *Tools) listNoteTitles(ctx context.Context, in NoteTitleListInput) (NoteTitleListOutput, error) {
	if len(in.Paths) > maxBatch {
		return NoteTitleListOutput{}, fmt.Errorf("%d paths given, at most %d are allowed", len(in.Paths), maxBatch)
	}
	refs, err := t.vault.GetNotes(ctx, in.Paths)
	if err != nil {
		return NoteTitleListOutput{}, err
	}
	out := NoteTitleListOutput{Notes: make([]NoteTitle, 0, len(refs)), MissingPaths: []string{}}
	for _, p := range in.Paths {
		ref, ok := refs[p]
		if !ok {
			out.MissingPaths = append(out.MissingPaths, p)
			continue
		}
		out.Notes = append(out.Notes, NoteTitle{Path: p, Title: ref.Title, ID: ref.ID})
	}
	return out, nil
}

func (t *Tools) resolveNoteName(ctx context.Context, in NoteResolveInput) (NoteResolveOutput, error) {
	if strings.TrimSpace(in.Name) == "" {
		return NoteResolveOutput{}, errors.New("name is required")
	}
	paths, err := t.vault.GetNamedPaths(ctx, in.Name)
	if err != nil {
		return NoteResolveOutput{}, fmt.Errorf("%s cannot be resolved: %w", in.Name, err)
	}
	if paths == nil {
		paths = []string{}
	}
	return NoteResolveOutput{Paths: paths}, nil
}

// getFolderPrefix returns the folder in the form vault paths start with: without a leading "./" or
// "/" and a trailing "/", composed (NFC), and followed by a slash, so "library" matches
// "library/a.pdf" and not "library2/a.pdf". The vault root is empty.
func getFolderPrefix(folder string) string {
	for {
		next := strings.TrimPrefix(strings.TrimLeft(folder, "/"), "./")
		if next == folder {
			break
		}
		folder = next
	}
	folder = strings.TrimRight(folder, "/")
	if folder == "" || folder == "." {
		return ""
	}
	return norm.NFC.String(folder) + "/"
}
