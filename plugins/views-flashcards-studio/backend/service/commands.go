package service

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
)

// The commands the manifest declares, and the argument each takes.
const (
	commandCreateDeck      = "deck.create"
	commandCreateCardModel = "cardModel.create"
	commandCreatePreset    = "preset.create"
	argumentTitle          = "title"
)

// firstCardModelField is the field a card model created by its command declares, which the user renames.
const firstCardModelField = "Field 1"

// Commands runs the commands the manifest declares.
func (b *Backend) Commands() (*command.Registry, error) {
	registry := command.New()
	for id, create := range map[string]func(context.Context, domain.Vault, cards.New) (cards.CreateNoteResult, error){
		commandCreateDeck: func(ctx context.Context, v domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			return b.scenarios().cards.Create.Deck(ctx, v, in)
		},
		commandCreateCardModel: func(ctx context.Context, v domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			in.Fields = []string{firstCardModelField}
			return b.scenarios().cards.Create.CardModel(ctx, v, in)
		},
		commandCreatePreset: func(ctx context.Context, v domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			return b.scenarios().cards.Create.Preset(ctx, v, in)
		},
	} {
		// gomutants:disable-next-line BRANCH_IF reason="Static command identifiers are non-empty and unique."
		if err := registry.Add(id, b.newCreateRun(create)); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// errRowGone is a row the vault holds no file or folder at.
var errRowGone = errors.New("no such file or folder")

func (b *Backend) getRowFolder(ctx context.Context, row string) (string, error) {
	if row == "" {
		return "", nil
	}
	_, err := b.vault.StatFile(ctx, row)
	switch {
	case errors.Is(err, host.ErrNotExist):
		return "", fmt.Errorf("%s: %w", row, errRowGone)
	case errors.Is(err, host.ErrNotANote):
		return row, nil
	case err != nil:
		return "", err
	}
	if folder := path.Dir(row); folder != "." {
		return folder, nil
	}
	return "", nil
}

func (b *Backend) newCreateRun(
	create func(context.Context, domain.Vault, cards.New) (cards.CreateNoteResult, error),
) command.Run {
	return func(ctx context.Context, request command.Request) (command.Result, error) {
		folder, err := b.getRowFolder(ctx, request.Path)
		if err != nil {
			return command.Result{}, err
		}
		made, code, _, err := b.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
			return create(ctx, showing, in)
		}, request.Arguments.GetString(argumentTitle), folder)
		if err != nil {
			return command.Result{}, err
		}
		if code != api.ErrorCodeUnspecified {
			return command.Result{}, errors.New(describeCreateError(code))
		}
		return command.Result{Message: "", OpenedPath: made.Path}, nil
	}
}

func describeCreateError(code api.ErrorCode) string {
	switch code {
	case api.ErrorCodeAlreadyExists:
		return "something of that name is filed there already, so nothing was written"
	case api.ErrorCodeInvalidName:
		return "a note cannot be called that"
	case api.ErrorCodeUnspecified, api.ErrorCodeNotFound, api.ErrorCodeNotANote, api.ErrorCodeNotText,
		api.ErrorCodeTooLarge, api.ErrorCodeBodyUnwritable, api.ErrorCodeUnreadable,
		api.ErrorCodeNotACardModel, api.ErrorCodeNotADeck, api.ErrorCodeDeckTooLarge,
		api.ErrorCodeNotAPreset, api.ErrorCodeConflict:
		// gomutants:disable-next-line BRANCH_CASE reason="Other error codes share the fallback message."
		return "the file could not be made: " + string(code)
	}
	return "the file could not be made: " + string(code)
}
