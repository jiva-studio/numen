package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
)

// The commands the manifest declares, and the argument each takes.
const (
	commandCreateDeck    = "create-deck"
	commandCreateStencil = "create-stencil"
	commandCreatePreset  = "create-preset"
	argumentTitle        = "title"
)

// firstStencilField is the field a stencil made by its command declares, which the person renames.
const firstStencilField = "Field 1"

// Commands runs the commands the manifest declares.
func (b *Backend) Commands() (*command.Registry, error) {
	registry := command.New()
	for id, create := range map[string]func(context.Context, domain.Vault, cards.New) (cards.CreateNoteResult, error){
		commandCreateDeck: func(ctx context.Context, v domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			return b.scenarios().cards.Create.Deck(ctx, v, in)
		},
		commandCreateStencil: func(ctx context.Context, v domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			in.Fields = []string{firstStencilField}
			return b.scenarios().cards.Create.Stencil(ctx, v, in)
		},
		commandCreatePreset: func(ctx context.Context, v domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			return b.scenarios().cards.Create.Preset(ctx, v, in)
		},
	} {
		if err := registry.Add(id, b.newCreateRun(create)); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// newCreateRun makes the file under the title at the root of the vault the window shows, and answers
// it as the file to open.
func (b *Backend) newCreateRun(
	create func(context.Context, domain.Vault, cards.New) (cards.CreateNoteResult, error),
) command.Run {
	return func(ctx context.Context, request command.Request) (command.Result, error) {
		made, code, _, err := b.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			return create(ctx, showing, in)
		}, request.Arguments.GetString(argumentTitle), "")
		if err != nil {
			return command.Result{}, err
		}
		if code != nil {
			return command.Result{}, errors.New(describeCreateError(*code))
		}
		return command.Result{OpenedPath: made.Path}, nil
	}
}

// describeCreateError is why a file was not made, in words a person reads.
func describeCreateError(code v1.ErrorCode) string {
	switch code {
	case v1.ErrorCode_ERROR_CODE_OCCUPIED:
		return "something of that name is filed there already, so nothing was written"
	case v1.ErrorCode_ERROR_CODE_UNNAMEABLE:
		return "a note cannot be called that"
	default:
		return "the file could not be made: " + code.String()
	}
}
