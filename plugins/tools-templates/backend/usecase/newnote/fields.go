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

// Execute lists the fields of the template, in the order they are asked.
func (u ListFields) Execute(ctx context.Context, in FieldsRequest) ([]engine.Field, error) {
	text, err := readTemplate(ctx, u.vault, u.getMasks(), in.Template)
	if err != nil {
		return nil, err
	}
	data := engine.Data{Title: getName(in.Template), Folder: in.Folder, ID: "", Now: u.clock.Now(), Values: nil}
	fields, err := u.renderer.Fields(pathpkg.Base(in.Template), text, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidNote, err)
	}
	return fields, nil
}
