package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// How much one call may ask for. A tool with no ceiling is a way to put a whole
// vault in a context window by accident, and a limit that truncates in silence
// reads as "that is all there is" — so going over fails and says so.
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
	addNoteReadTools(server, core)
	addNoteResolve(server, core)
	addNoteWriteTools(server, core)
}

type noteResolveInput struct {
	Name string `json:"name" jsonschema:"a note's filename without its extension, which is what a link writes"`
}

type noteResolveOutput struct {
	Paths []string `json:"paths"`
}

// addNoteResolve is the notes a link's name reaches. It is what the window a
// person writes in asks; the surfaces that only read are served the lookups by
// path and no more.
func addNoteResolve(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_resolve",
		Title: "Resolve a name",
		Description: "Every note filed under one name, which is what a link written by " +
			"that name resolves to. More than one path back means the link is ambiguous " +
			"and reaches the nearest of them, which can change when either note is " +
			"moved. Nothing back means no note answers to the name.",
	}, newHandler(func(ctx context.Context, in noteResolveInput) (noteResolveOutput, error) {
		paths, err := core.Notes.GetNamedPaths.Execute(ctx, core.getCurrentVault().Vault, in.Name)
		if err != nil {
			return noteResolveOutput{}, err
		}
		return noteResolveOutput{Paths: paths}, nil
	}))
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

// ReadFailure is one path that came back with no prose behind it, and what
// stopped it. A batch of ten notes with one export among them comes back with
// nine.
type ReadFailure struct {
	Path string `json:"path"`
	Why  string `json:"why" jsonschema:"why this one was not read"`
}

// Neighbour is a note in the picture around another one.
type Neighbour struct {
	Note     Note   `json:"note"`
	Relation string `json:"relation" jsonschema:"parent, child, sibling or jump"`
	Label    string `json:"label,omitempty" jsonschema:"what the person calls this relationship"`
	Through  string `json:"through,omitempty" jsonschema:"the note they share, when they are siblings"`
	IsMutual bool   `json:"mutual,omitempty" jsonschema:"set when both notes name this relationship, the label being then the word the note in focus wrote"`
}

// The filesystem offers no transaction over many files: the twenty-ninth can
// fail on its own. A batch says what happened to each.

// MoveOutcome is what happened to one note in a batch. Refused is empty when it
// moved.
type MoveOutcome struct {
	note.MoveResult
	Error string `json:"error,omitempty" jsonschema:"why this one did not move, empty when it did"`
}

// NewNote is one note a caller wants made, and what it should be joined to.
type NewNote struct {
	Title  string `json:"title" jsonschema:"what the note is called"`
	Body   string `json:"body,omitempty" jsonschema:"the markdown to start it with"`
	Folder string `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
	Links  []Link `json:"links,omitempty" jsonschema:"the relationships to write into it, so it arrives already joined"`
}

// CreateOutcome is what happened to one note in a batch.
type CreateOutcome struct {
	note.CreateResult
	Error string `json:"error,omitempty" jsonschema:"why this one was not made, empty when it was"`
}

// RemoveOutcome is the same for removing.
type RemoveOutcome struct {
	note.RemoveResult
	Error string `json:"error,omitempty" jsonschema:"why this one was not removed, empty when it was"`
}
