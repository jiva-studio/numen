package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func addNoteWriteTools(server *sdk.Server, core Core) {
	addImportTool(server, core)
	addNoteCreate(server, core)
	addNoteRewrite(server, core)
	addNoteEdit(server, core)
	addNoteRename(server, core)
	addNoteMove(server, core)
	addNoteRemove(server, core)
}

// addNoteCreate adds the tool that creates one note per call.
func addNoteCreate(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
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
	}, newHandler(func(ctx context.Context, in NewNote) (CreateOutcome, error) {
		return createNote(ctx, core, in)
	}))
}

func createNote(ctx context.Context, core Core, in NewNote) (CreateOutcome, error) {
	size := len(in.Body)
	for _, l := range in.Links {
		size += countLinkBytes(l)
	}
	if size > maxBytes {
		return CreateOutcome{}, fmt.Errorf(
			"a call writing %d bytes is more than this carries at once, which is %d", size, maxBytes)
	}

	created, err := core.Notes.Create.Execute(ctx, core.getCurrentVault().Vault, note.NewNote{
		Title: in.Title, Body: in.Body, Path: in.Folder,
		Links: newLinks(in.Links),
	})
	// A path returned with an error means the file was written and a later step failed.
	outcome := CreateOutcome{CreateResult: created}
	if err != nil {
		outcome.CreateResult.Title = in.Title
		outcome.Error = sayError(err)
	}
	return outcome, nil
}

type noteRewriteInput struct {
	Path        string `json:"path" jsonschema:"the note to write"`
	Body        string `json:"body" jsonschema:"the markdown to put in it"`
	Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses a write over somebody else's edit"`
}

type noteRewriteOutput struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

func addNoteRewrite(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_rewrite",
		Title: "Rewrite a note",
		Description: "Replace the whole prose of a note. The frontmatter is left alone — " +
			"use the link tools to change what a note is joined to. The fingerprint from " +
			"`note_read` is required, and a write lands only on the note that fingerprint " +
			"names. This " +
			"answers with the fingerprint it produced: pass that one to write the same " +
			"note again without reading it back. To change part of a note, `note_edit` " +
			"replaces one span and leaves the rest untouched; this is for a note being " +
			"rewritten, or one short enough that rewriting it is the plainer thing — under " +
			"about 800 characters it usually is.",
	}, newHandler(func(ctx context.Context, in noteRewriteInput) (noteRewriteOutput, error) {
		return rewriteNote(ctx, core, in)
	}))
}

func rewriteNote(ctx context.Context, core Core, in noteRewriteInput) (noteRewriteOutput, error) {
	if len(in.Body) > maxBytes {
		return noteRewriteOutput{}, fmt.Errorf("a body of %d bytes is larger than the %d this writes",
			len(in.Body), maxBytes)
	}
	ref, err := parseFingerprint(in.Fingerprint)
	if err != nil {
		return noteRewriteOutput{}, err
	}
	// The fingerprint after the write; the next write to this note presents it.
	content, err := core.Notes.Write.Execute(ctx, core.getCurrentVault().Vault, in.Path, in.Body, ref)
	if err != nil {
		return noteRewriteOutput{}, err
	}
	return noteRewriteOutput{Path: in.Path, Fingerprint: fingerprintOf(content)}, nil
}

type noteEditInput struct {
	Path        string `json:"path" jsonschema:"the note to edit"`
	Match       string `json:"match" jsonschema:"the text to replace, as the note has it"`
	Text        string `json:"text" jsonschema:"what to put in its place; empty takes the text out"`
	Fingerprint string `json:"fingerprint" jsonschema:"what note_read said the note was, which refuses an edit over somebody else's edit"`
}

type noteEditOutput struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
	Match       string `json:"match" jsonschema:"the text that was replaced, as the note had it"`
	IsLoose     bool   `json:"loose,omitempty" jsonschema:"the span was found only once punctuation and spacing were flattened, so what the note held is not what you asked for"`
}

func addNoteEdit(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_edit",
		Title: "Edit a note",
		Description: "Replace one span of a note's prose with another and leave the " +
			"rest of it the bytes it was. `match` is that span as `note_read` gave it " +
			"to you, and it must stand in exactly one place: where it stands twice, take " +
			"in enough of what surrounds one of them to tell it from the others. Quotes, " +
			"dashes and spacing may differ from what the note has and the span is " +
			"still found; the answer says so, and says what the note held. Reach for this " +
			"before `note_rewrite` for anything short of rewriting a note — it costs you the " +
			"span instead of the whole note, and it cannot change a word you did not " +
			"name. The fingerprint from `note_read` is required, and an edit lands only on " +
			"the note that fingerprint names. It answers with the fingerprint it produced.",
	}, newHandler(func(ctx context.Context, in noteEditInput) (noteEditOutput, error) {
		return editNote(ctx, core, in)
	}))
}

func editNote(ctx context.Context, core Core, in noteEditInput) (noteEditOutput, error) {
	baseline, err := parseFingerprint(in.Fingerprint)
	if err != nil {
		return noteEditOutput{}, err
	}
	isDone, err := core.Notes.Replace.Execute(ctx, core.getCurrentVault().Vault, in.Path, in.Match, in.Text, baseline)
	if err != nil {
		return noteEditOutput{}, err
	}
	// The text found at the span is returned as stored; a span matched loosely
	// is not the requested text.
	return noteEditOutput{
		Path:        in.Path,
		Fingerprint: fingerprintOf(isDone.Fingerprint),
		Match:       isDone.Matched,
		IsLoose:     isDone.IsPlain,
	}, nil
}
