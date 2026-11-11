// Package cardcommands is the commands that make a deck, a card model or a preset, which the
// manifest declares.
package cardcommands

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

// The commands the manifest declares, and the argument each takes.
const (
	CreateDeck      = "deck.create"
	CreateCardModel = "cardModel.create"
	CreatePreset    = "preset.create"
	ArgumentTitle   = "title"
)

// firstCardModelField is the field a card model created by its command declares, which the user renames.
const firstCardModelField = "Field 1"

// CreateNote makes a note in the current vault through build, and reports how it ended the way
// apierror.ClassifyCreate does.
type CreateNote func(
	ctx context.Context, build func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, api.ErrorCode, bool, error)

// kind is one note a command makes: the scenario that makes it, and the fields it starts with.
type kind struct {
	create func(cards.Create, context.Context, domain.Vault, cards.New) (cards.CreateNoteResult, error)
	fields []string
}

// New returns the commands. A command run on a row of the file list makes its note in the folder
// of that row.
func New(current func() cards.Create, files cards.FileStatter, createNote CreateNote) (*command.Registry, error) {
	registry := command.New()
	for id, one := range map[string]kind{
		CreateDeck:      {create: cards.Create.Deck, fields: nil},
		CreateCardModel: {create: cards.Create.CardModel, fields: []string{firstCardModelField}},
		CreatePreset:    {create: cards.Create.Preset, fields: nil},
	} {
		// gomutants:disable-next-line BRANCH_IF reason="Static command identifiers are non-empty and unique."
		if err := registry.Add(id, newRun(one, current, files, createNote)); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func newRun(one kind, current func() cards.Create, files cards.FileStatter, createNote CreateNote) command.Run {
	return func(ctx context.Context, request command.Request) (command.Result, error) {
		folder, err := cards.GetRowFolder(ctx, files, request.Path)
		if err != nil {
			return command.Result{}, err
		}
		made, code, _, err := createNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			in.Fields = one.fields
			return one.create(current(), ctx, showing, in)
		}, request.Arguments.GetString(ArgumentTitle), folder)
		if err != nil {
			return command.Result{}, err
		}
		if code != api.ErrorCodeUnspecified {
			return command.Result{}, errors.New(apierror.DescribeCreate(code))
		}
		return command.Result{Message: "", OpenedPath: made.Path}, nil
	}
}
