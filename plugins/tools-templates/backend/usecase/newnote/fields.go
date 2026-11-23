package newnote

import (
	"context"
	"fmt"
	pathpkg "path"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/mask"
)

// FieldsRequest names a template and the folder a note from it would be made in.
type FieldsRequest struct {
	Template string
	Folder   string
}

// FieldsResult is the fields of a template with the etag of the text they were read from, and the name a note
// from the template is suggested to be given in the folder.
type FieldsResult struct {
	Fields []engine.Field
	Etag   string
	Name   string
}

// ListFields lists what a template asks of the person before it is made into a note.
type ListFields struct {
	vault    Vault
	clock    Clock
	getMasks func() mask.Set
	renderer engine.Renderer
}

// NewListFields returns the use case.
func NewListFields(vault Vault, clock Clock, getMasks func() mask.Set) ListFields {
	return ListFields{vault: vault, clock: clock, getMasks: getMasks, renderer: engine.New(nil)}
}

// Execute lists the fields of the template, in the order they are asked, and suggests the name of the note: the
// name of the template, then with 2, 3 and on up to 100 where a note of the folder has it, and with every number
// taken nothing.
func (u ListFields) Execute(ctx context.Context, in FieldsRequest) (FieldsResult, error) {
	if !isFolder(in.Folder) {
		return FieldsResult{}, fmt.Errorf("%w: %q", ErrInvalidFolder, in.Folder)
	}
	text, err := readTemplate(ctx, u.vault, u.getMasks(), in.Template)
	if err != nil {
		return FieldsResult{}, err
	}
	name, err := suggestName(ctx, u.vault, in.Folder, in.Template)
	if err != nil {
		return FieldsResult{}, err
	}
	data := engine.Data{Title: name, Folder: in.Folder, ID: "", Now: u.clock.Now(), Values: nil}
	fields, err := u.renderer.ListFields(ctx, pathpkg.Base(in.Template), text, data)
	if err != nil {
		return FieldsResult{}, fmt.Errorf("%w: %w", ErrInvalidNote, err)
	}
	return FieldsResult{Fields: fields, Etag: ComputeEtag(text), Name: name}, nil
}
