package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func addNoteReadTools(server *sdk.Server, core Core) {
	addNoteSearch(server, core)
	addNoteTitles(server, core)
	addNoteRead(server, core)
	addNoteNeighbourhood(server, core)
}

type noteTitlesInput struct {
	Paths []string `json:"paths" jsonschema:"the paths to look up"`
}

type noteTitlesOutput struct {
	Notes   []Note   `json:"notes"`
	Missing []string `json:"missing,omitempty"`
}

func addNoteTitles(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_titles",
		Title: "Look up note titles",
		Description: "What notes at these paths are called, and the identifier each " +
			"carries. Nothing of their prose comes back — `note_read` gives that. Use " +
			"this to name a note in an answer, or to see whether the vault still holds " +
			"one. Paths that name nothing come back under `missing` rather than as an " +
			"error: a note may have been removed since you last saw it.",
	}, newHandler(func(ctx context.Context, in noteTitlesInput) (noteTitlesOutput, error) {
		return getNoteTitles(ctx, core, in)
	}))
}

func getNoteTitles(ctx context.Context, core Core, in noteTitlesInput) (noteTitlesOutput, error) {
	if len(in.Paths) > maxRefs {
		return noteTitlesOutput{}, fmt.Errorf("ask about at most %d notes at a time", maxRefs)
	}
	found, err := core.Notes.GetTitles.Execute(ctx, core.getCurrentVault().Vault, in.Paths)
	if err != nil {
		return noteTitlesOutput{}, err
	}
	res := noteTitlesOutput{Notes: make([]Note, 0, len(found.Notes)), Missing: found.Missing}
	for _, ref := range found.Notes {
		res.Notes = append(res.Notes, noteOf(ref))
	}
	return res, nil
}

type noteReadInput struct {
	Paths []string `json:"paths" jsonschema:"the paths to read"`
}

type noteReadOutput struct {
	Notes   []Contents    `json:"notes"`
	Missing []string      `json:"missing,omitempty"`
	Errors  []ReadFailure `json:"errors,omitempty"`
}

func addNoteRead(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_read",
		Title: "Read notes",
		Description: "Read the prose of notes — the text below the frontmatter, which " +
			"is exactly what `note_rewrite` takes back. What a note is called is not in " +
			"here; `note_titles` answers that, and for less. What a note is joined to is " +
			"not in here either; `link_list` answers that. The fingerprint that comes back is " +
			"what `note_rewrite` wants: hand it back and the write fails if the " +
			"person changed the note in the meantime. A path that names nothing comes " +
			"back under `missing` and one that could not be read under `errors` saying " +
			"why, and the rest of the batch still comes back.",
	}, newHandler(func(ctx context.Context, in noteReadInput) (noteReadOutput, error) {
		return readNotes(ctx, core, in)
	}))
}

func readNotes(ctx context.Context, core Core, in noteReadInput) (noteReadOutput, error) {
	if len(in.Paths) > maxBodies {
		return noteReadOutput{}, fmt.Errorf("read at most %d notes at a time", maxBodies)
	}
	read := core.Notes.Read
	res := noteReadOutput{Notes: make([]Contents, 0, len(in.Paths))}
	for _, path := range in.Paths {
		if err := ctx.Err(); err != nil {
			return noteReadOutput{}, err
		}
		contents, err := read.Execute(ctx, core.getCurrentVault().Vault, path)
		if err != nil {
			return noteReadOutput{}, err
		}
		addNoteContents(&res, path, contents)
	}
	return res, nil
}

// addNoteContents adds the contents read from the note at path to the result.
func addNoteContents(res *noteReadOutput, path string, contents note.Contents) {
	switch contents.Outcome {
	case note.Ok:
		res.Notes = append(res.Notes, Contents{
			Path: path,
			Body: contents.Body,
			// The fingerprint is taken before the bytes are read; a write in between
			// makes it stale and the next write fails.
			Fingerprint: fingerprintOf(contents.Fingerprint),
		})
	case note.Missing:
		res.Missing = append(res.Missing, path)
	default:
		res.Errors = append(res.Errors, ReadFailure{Path: path, Why: why(contents)})
	}
}

type noteNeighbourhoodInput struct {
	Path string `json:"path" jsonschema:"the note to look out from"`
}

type noteNeighbourhoodOutput struct {
	Focus   Note        `json:"focus"`
	Related []Neighbour `json:"related"`
}

func addNoteNeighbourhood(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_neighbourhood",
		Title: "Show a note's neighbourhood",
		Description: "One note and everything joined to it — its parents, children, " +
			"siblings and jumps. This is the picture the person is looking at.",
	}, newHandler(func(ctx context.Context, in noteNeighbourhoodInput) (noteNeighbourhoodOutput, error) {
		found, err := core.Notes.Neighbourhood.Execute(ctx, core.getCurrentVault().Vault, in.Path)
		if err != nil {
			return noteNeighbourhoodOutput{}, err
		}
		res := noteNeighbourhoodOutput{Focus: noteOf(found.Focus)}
		for _, related := range found.Related {
			res.Related = append(res.Related, Neighbour{
				Note:     noteOf(related.NoteRef),
				Relation: string(related.Relation),
				Label:    related.Label,
				Through:  related.Parent,
				IsMutual: related.IsMutual,
			})
		}
		return res, nil
	}))
}
