package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// How much one call may ask for. A tool with no ceiling is a way to put a whole
// vault in a context window by accident, and a limit that truncates in silence
// reads as "that is all there is" — so going over is refused and says so.
const (
	maxRefs   = 50
	maxBodies = 10
)

// maxBytes is the most one call will carry in either direction, whether that is
// one note or a batch of them. It is the ceiling a read holds a note to.
//
// What a link says is measured with the prose. It lands in the same file, and
// the bound is on what a call writes.
const maxBytes = note.MaxBytes

// Note is a note as every tool reports it: the address it is asked for by, what
// it is called, and the identifier if it carries one.
type Note struct {
	Path  string `json:"path" jsonschema:"the note's path relative to the vault folder"`
	Title string `json:"title" jsonschema:"what the note is called"`
	ID    string `json:"id,omitempty" jsonschema:"the note's stable identifier, absent for a note written outside the application"`
}

func noteOf(ref domain.NoteRef) Note {
	return Note{Path: ref.Path, Title: ref.Title, ID: ref.ID}
}

func addNoteTools(server *sdk.Server, core Core) {
	addNoteReadingTools(server, core)
	addNoteResolve(server, core)
	addNoteWritingTools(server, core)
}

// addNoteResolve is the notes a link's name reaches. It is what the window a
// person writes in asks; the surfaces that only read are served the lookups by
// path and no more.
func addNoteResolve(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_resolve",
		Title: "Resolve a name",
		Description: "Every note filed under one name, which is what a link written by " +
			"that name resolves to. More than one path back means the link is ambiguous " +
			"and reaches the nearest of them, which can change when either note is " +
			"moved. Nothing back means no note answers to the name.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Name string `json:"name" jsonschema:"a note's filename without its extension, which is what a link writes"`
	}) (*sdk.CallToolResult, struct {
		Paths []string `json:"paths"`
	}, error) {
		type out = struct {
			Paths []string `json:"paths"`
		}
		paths, err := core.Notes.Queries.Named(ctx, core.shown().Vault.ID, domain.LinkName(in.Name))
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Paths: paths}, nil
	})
}

func addNoteReadingTools(server *sdk.Server, core Core) {
	addNoteSearch(server, core)

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_titles",
		Title: "Look up note titles",
		Description: "What notes at these paths are called, and the identifier each " +
			"carries. Nothing of their prose comes back — `note_read` gives that. Use " +
			"this to name a note in an answer, or to see whether the vault still holds " +
			"one. Paths that name nothing come back under `missing` rather than as an " +
			"error: a note may have been removed since you last saw it.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Paths []string `json:"paths" jsonschema:"the paths to look up"`
	}) (*sdk.CallToolResult, struct {
		Notes   []Note   `json:"notes"`
		Missing []string `json:"missing,omitempty"`
	}, error) {
		type out = struct {
			Notes   []Note   `json:"notes"`
			Missing []string `json:"missing,omitempty"`
		}
		if len(in.Paths) > maxRefs {
			return nil, out{}, fmt.Errorf("ask about at most %d notes at a time", maxRefs)
		}
		found, err := core.Notes.Queries.Notes(ctx, core.shown().Vault.ID, in.Paths)
		if err != nil {
			return nil, out{}, err
		}
		res := out{Notes: make([]Note, 0, len(found))}
		for _, path := range in.Paths {
			if ref, ok := found[path]; ok {
				res.Notes = append(res.Notes, noteOf(ref))
				continue
			}
			res.Missing = append(res.Missing, path)
		}
		return nil, res, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_read",
		Title: "Read notes",
		Description: "Read the prose of notes — the text below the frontmatter, which " +
			"is exactly what `note_rewrite` takes back. What a note is called is not in " +
			"here; `note_titles` answers that, and for less. What a note is joined to is " +
			"not in here either; `link_list` answers that. The fingerprint that comes back is " +
			"what `note_rewrite` wants: hand it back and the write is refused if the " +
			"person changed the note in the meantime. A path that could not be read " +
			"comes back under `refused` saying why, and the rest of the batch still " +
			"comes back.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Paths []string `json:"paths" jsonschema:"the paths to read"`
	}) (*sdk.CallToolResult, struct {
		Notes   []Contents `json:"notes"`
		Missing []string   `json:"missing,omitempty"`
		Refused []Refusal  `json:"refused,omitempty"`
	}, error) {
		type out = struct {
			Notes   []Contents `json:"notes"`
			Missing []string   `json:"missing,omitempty"`
			Refused []Refusal  `json:"refused,omitempty"`
		}
		if len(in.Paths) > maxBodies {
			return nil, out{}, fmt.Errorf("read at most %d notes at a time", maxBodies)
		}
		read := note.NewRead(core.Readers)
		res := out{Notes: make([]Contents, 0, len(in.Paths))}
		for _, path := range in.Paths {
			if err := ctx.Err(); err != nil {
				return nil, out{}, err
			}
			contents, err := read.Execute(ctx, core.shown().Vault, path)
			if err != nil {
				return nil, out{}, err
			}
			switch contents.Outcome {
			case note.Ok:
				res.Notes = append(res.Notes, Contents{
					Path: path,
					Body: contents.Body,
					// What the file was when it was asked about, which is
					// before its bytes were read. A write landing in between
					// makes this stale, and the next write is refused.
					Fingerprint: fingerprintOf(contents.Fingerprint),
				})
			case note.Missing:
				res.Missing = append(res.Missing, path)
			default:
				res.Refused = append(res.Refused, Refusal{Path: path, Why: why(contents)})
			}
		}
		return nil, res, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_neighbourhood",
		Title: "Show a note's neighbourhood",
		Description: "One note and everything joined to it — its parents, children, " +
			"siblings and jumps. This is the picture the person is looking at.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the note to look out from"`
	}) (*sdk.CallToolResult, struct {
		Focus   Note        `json:"focus"`
		Related []Neighbour `json:"related"`
	}, error) {
		type out = struct {
			Focus   Note        `json:"focus"`
			Related []Neighbour `json:"related"`
		}
		found, err := core.Notes.Neighbourhood.Execute(ctx, core.shown().Vault, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		res := out{Focus: noteOf(found.Focus)}
		for _, related := range found.Related {
			res.Related = append(res.Related, Neighbour{
				Note:    noteOf(related.NoteRef),
				Seat:    string(related.Seat),
				Label:   related.Label,
				Through: related.Parent,
				Mutual:  related.Mutual,
			})
		}
		return nil, res, nil
	})
}

func addNoteWritingTools(server *sdk.Server, core Core) {
	addImportTool(server, core)

	// One note per call.
	//
	// A call is written out in full before it is made, and this one carries the
	// text of a note: one at a time, each is filed as it is finished, and stopping
	// halfway keeps what was made. The other calls that take a list carry names,
	// which are written in a moment.
	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_create",
		Title: "Create a note",
		Description: "Make one note. It is named after its title, so choose a title that " +
			"reads as a name. For several notes call this once for each, in the order " +
			"they should appear: every note is filed as it is finished, and the person " +
			"watching sees each one arrive. Give the note its `links` here rather than " +
			"adding them afterwards: it is one write, and the person sees it arrive " +
			"already joined instead of appearing loose and then jumping into place. If " +
			"other notes already answer to a name they come back under `shares`, and " +
			"links written by that name will be ambiguous.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in NewNote) (*sdk.CallToolResult, CreateOutcome, error) {
		size := len(in.Body)
		for _, l := range in.Links {
			size += carried(l)
		}
		if size > maxBytes {
			return nil, CreateOutcome{}, fmt.Errorf(
				"a call writing %d bytes is more than this carries at once, which is %d", size, maxBytes)
		}

		created, err := core.Notes.Create.Execute(ctx, core.shown().Vault, note.NewNote{
			Title: in.Title, Body: in.Body, Folder: in.Folder,
			Links: written(in.Links),
		})
		// A path alongside a refusal means the file was written and something
		// after it was not; the note is there under that name.
		outcome := CreateOutcome{CreateResult: created}
		if err != nil {
			outcome.CreateResult.Title = in.Title
			outcome.Refused = refusing(err)
		}
		return nil, outcome, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_rewrite",
		Title: "Rewrite a note",
		Description: "Replace the whole prose of a note. The frontmatter is left alone — " +
			"use the link tools to change what a note is joined to. The fingerprint from " +
			"`note_read` is required, and a write lands only on the note that fingerprint " +
			"names. This " +
			"answers with the fingerprint it produced: pass that one to write the same " +
			"note again without reading it back. To change part of a note, `note_edit` " +
			"replaces one stretch and leaves the rest untouched; this is for a note being " +
			"rewritten, or one short enough that rewriting it is the plainer thing — under " +
			"about 800 characters it usually is.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the note to write"`
		Body        string `json:"body" jsonschema:"the markdown to put in it"`
		Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, struct {
		Path        string `json:"path"`
		Fingerprint string `json:"fingerprint"`
	}, error) {
		type out = struct {
			Path        string `json:"path"`
			Fingerprint string `json:"fingerprint"`
		}
		if len(in.Body) > maxBytes {
			return nil, out{}, fmt.Errorf("a body of %d bytes is larger than the %d this writes",
				len(in.Body), maxBytes)
		}
		ref, err := parseFingerprint(in.Fingerprint)
		if err != nil {
			return nil, out{}, err
		}
		// What the file became. A caller writing this note again presents it, and
		// the one it read is behind by its own write.
		written, err := core.Notes.Write.Execute(ctx, core.shown().Vault, in.Path, in.Body, ref)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Path: in.Path, Fingerprint: fingerprintOf(written)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_edit",
		Title: "Edit a note",
		Description: "Replace one stretch of a note's prose with another and leave the " +
			"rest of it the bytes it was. `match` is that stretch as `note_read` gave it " +
			"to you, and it must stand in exactly one place: where it stands twice, take " +
			"in enough of what surrounds one of them to tell it from the others. Quotes, " +
			"dashes and spacing may differ from what the note has and the stretch is " +
			"still found; the answer says so, and says what the note held. Reach for this " +
			"before `note_rewrite` for anything short of rewriting a note — it costs you the " +
			"stretch instead of the whole note, and it cannot change a word you did not " +
			"name. The fingerprint from `note_read` is required, and an edit lands only on " +
			"the note that fingerprint names. It answers with the fingerprint it produced.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the note to edit"`
		Match       string `json:"match" jsonschema:"the text to replace, as the note has it"`
		Text        string `json:"text" jsonschema:"what to put in its place; empty takes the text out"`
		Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses an edit over somebody else's edit"`
	}) (*sdk.CallToolResult, struct {
		Path        string `json:"path"`
		Fingerprint string `json:"fingerprint"`
		Match       string `json:"match" jsonschema:"the text that was replaced, as the note had it"`
		Loose       bool   `json:"loose,omitempty" jsonschema:"the stretch was found only once punctuation and spacing were flattened, so what the note held is not what you asked for"`
	}, error) {
		type out = struct {
			Path        string `json:"path"`
			Fingerprint string `json:"fingerprint"`
			Match       string `json:"match" jsonschema:"the text that was replaced, as the note had it"`
			Loose       bool   `json:"loose,omitempty" jsonschema:"the stretch was found only once punctuation and spacing were flattened, so what the note held is not what you asked for"`
		}
		seen, err := parseFingerprint(in.Fingerprint)
		if err != nil {
			return nil, out{}, err
		}
		done, err := core.Notes.Replace.Execute(ctx, core.shown().Vault, in.Path, in.Match, in.Text, seen)
		if err != nil {
			return nil, out{}, err
		}
		// What stood there is answered because a stretch found by a looser
		// reading is not the text that was asked for.
		return nil, out{
			Path:        in.Path,
			Fingerprint: fingerprintOf(done.Fingerprint),
			Match:       done.Matched,
			Loose:       done.Plainly,
		}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_rename",
		Title: "Rename a note",
		Description: "Give a note a different name. " + namingOrder + " Whichever of the " +
			"three names it is brought into line, and the file is renamed with it. A note " +
			"its filename names is moved and not written. The answer says which of them " +
			"named it, and what the file did. Links written by the old name are repaired " +
			"only where they stopped resolving.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path  string `json:"path" jsonschema:"the note to rename"`
		Title string `json:"title" jsonschema:"what it is called from now on"`
	}) (*sdk.CallToolResult, note.RenameResult, error) {
		renamed, err := core.Notes.Rename.Execute(ctx, core.shown().Vault, in.Path, in.Title)
		return nil, renamed, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_move",
		Title: "Move a note",
		Description: "File notes under a different folder, keeping their names. Folders " +
			"are for how the files are arranged on disk and change nothing about the " +
			"graph. Links written by name follow the note.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Paths  []string `json:"paths" jsonschema:"the notes to move"`
		Folder string   `json:"folder" jsonschema:"where they go, relative to the vault folder; empty is the root"`
	}) (*sdk.CallToolResult, struct {
		Moved []MoveOutcome `json:"moved"`
	}, error) {
		type out = struct {
			Moved []MoveOutcome `json:"moved"`
		}
		if len(in.Paths) > maxRefs {
			return nil, out{}, fmt.Errorf("move at most %d notes at a time", maxRefs)
		}
		res := out{Moved: make([]MoveOutcome, 0, len(in.Paths))}
		for _, path := range in.Paths {
			if err := ctx.Err(); err != nil {
				return nil, out{}, err
			}
			moved, err := core.Notes.Move.Execute(ctx, core.shown().Vault, path, note.Into(in.Folder, path))
			outcome := MoveOutcome{MoveResult: moved}
			if err != nil {
				outcome.MoveResult = note.MoveResult{From: path}
				outcome.Refused = refusing(err)
			}
			res.Moved = append(res.Moved, outcome)
		}
		return nil, res, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_remove",
		Title: "Remove a note",
		Description: "Take notes out of the vault. They go to the vault's trash folder " +
			"and can be put back — unless the call sets `destroy`, which takes the file " +
			"off the disk and leaves nothing to put back. " +
			"Links that pointed at them are left as they are and " +
			"come back under `dangling`: a link is not wrong because its note is gone. " +
			"This removes notes: a path naming a folder is refused, and the notes under " +
			"one are removed by naming each of them.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Paths   []string `json:"paths" jsonschema:"the notes to remove"`
		Destroy bool     `json:"destroy,omitempty" jsonschema:"delete outright instead of moving to the trash; nothing brings these back"`
	}) (*sdk.CallToolResult, struct {
		Removed []RemoveOutcome `json:"removed"`
	}, error) {
		type out = struct {
			Removed []RemoveOutcome `json:"removed"`
		}
		if len(in.Paths) > maxRefs {
			return nil, out{}, fmt.Errorf("remove at most %d notes at a time", maxRefs)
		}
		reader, err := core.Readers.Open(core.shown().Vault)
		if err != nil {
			return nil, out{}, err
		}
		res := out{Removed: make([]RemoveOutcome, 0, len(in.Paths))}
		for _, path := range in.Paths {
			if err := ctx.Err(); err != nil {
				return nil, out{}, err
			}
			if isFolder(ctx, reader, path) {
				res.Removed = append(res.Removed, RemoveOutcome{
					RemoveResult: note.RemoveResult{Path: path},
					Refused:      "this is a folder, and this removes notes: name the notes to remove",
				})
				continue
			}
			var removed note.RemoveResult
			var err error
			if in.Destroy {
				removed, err = core.Notes.Remove.Destroy(ctx, core.shown().Vault, path)
			} else {
				removed, err = core.Notes.Remove.Execute(ctx, core.shown().Vault, path)
			}
			outcome := RemoveOutcome{RemoveResult: removed}
			if err != nil {
				outcome.RemoveResult = note.RemoveResult{Path: path}
				outcome.Refused = refusing(err)
			}
			res.Removed = append(res.Removed, outcome)
		}
		return nil, res, nil
	})
}

// isFolder reports whether the vault holds a folder at the path. A folder is
// listed; a path holding a file is not.
func isFolder(ctx context.Context, reader port.VaultReader, path string) bool {
	_, err := reader.List(ctx, path)
	return err == nil
}

// Contents is a note as note_rewrite takes it back: the prose, without the
// frontmatter. Handing back the whole file would invite an agent to edit what
// it was given and write that in, putting a second frontmatter block inside the
// body.
type Contents struct {
	Path        string `json:"path"`
	Body        string `json:"body" jsonschema:"the prose below the frontmatter, which is what note_rewrite takes"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to note_rewrite to refuse a write over an edit you did not see"`
}

// Refusal is one path that came back with no prose behind it, and what stopped
// it. A batch of ten notes with one export among them comes back with nine.
type Refusal struct {
	Path string `json:"path"`
	Why  string `json:"why" jsonschema:"why this one was not read"`
}

// Neighbour is a note in the picture around another one.
type Neighbour struct {
	Note    Note   `json:"note"`
	Seat    string `json:"seat" jsonschema:"parent, child, sibling or jump"`
	Label   string `json:"label,omitempty" jsonschema:"what the person calls this relationship"`
	Through string `json:"through,omitempty" jsonschema:"the note they share, when they are siblings"`
	Mutual  bool   `json:"mutual,omitempty" jsonschema:"set when both notes name this relationship, the label being then the word the note in focus wrote"`
}

// The filesystem offers no transaction over many files: the twenty-ninth can
// fail on its own. A batch says what happened to each.

// MoveOutcome is what happened to one note in a batch. Refused is empty when it
// moved.
type MoveOutcome struct {
	note.MoveResult
	Refused string `json:"refused,omitempty" jsonschema:"why this one did not move, empty when it did"`
}

// NewNote is one note a caller wants made, and what it should be joined to.
type NewNote struct {
	Title  string    `json:"title" jsonschema:"what the note is called"`
	Body   string    `json:"body,omitempty" jsonschema:"the markdown to start it with"`
	Folder string    `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
	Links  []NewLink `json:"links,omitempty" jsonschema:"the relationships to write into it, so it arrives already joined"`
}

// CreateOutcome is what happened to one note in a batch.
type CreateOutcome struct {
	note.CreateResult
	Refused string `json:"refused,omitempty" jsonschema:"why this one was not made, empty when it was"`
}

// RemoveOutcome is the same for removing.
type RemoveOutcome struct {
	note.RemoveResult
	Refused string `json:"refused,omitempty" jsonschema:"why this one was not removed, empty when it was"`
}
