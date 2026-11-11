package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/apierror"
)

// vaultSession is what the backend and its handlers share about the vault a call works on: how to reach
// a vault by identity, the vault the window shows, and the title of a note in it.
type vaultSession struct {
	vaults func(domain.VaultID) ports.HostVault
	notes  hostports.Notes
}

func (s *vaultSession) getCurrentVault(ctx context.Context) (domain.Vault, error) {
	v, err := s.vaults("").Selector.GetVault(ctx)
	if err != nil {
		return domain.Vault{}, apierror.NewCurrentVaultError(err)
	}
	return v, nil
}

func (s *vaultSession) getVault(ctx context.Context, id string) (domain.Vault, error) {
	v, err := s.vaults(domain.VaultID(id)).Selector.GetVault(ctx)
	if err != nil {
		return domain.Vault{}, apierror.NewVaultError(err)
	}
	return v, nil
}

func (s *vaultSession) getTitle(ctx context.Context, v domain.Vault, path string) string {
	if path == "" {
		return ""
	}
	found, err := s.notes.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}

func (s *vaultSession) newTitler(ctx context.Context, v domain.Vault) func(path string) string {
	return func(path string) string { return s.getTitle(ctx, v, path) }
}

func (s *vaultSession) createCardNote(
	ctx context.Context, create func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, api.ErrorCode, bool, error) {
	showing, err := s.getCurrentVault(ctx)
	if err != nil {
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, err
	}
	return apierror.ClassifyCreate(create(showing, cards.New{Title: title, Body: "", Path: folder, Fields: nil, Templates: nil}))
}
