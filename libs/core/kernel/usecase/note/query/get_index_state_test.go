package query_test

import (
	"context"
	"errors"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type vaultsHeld struct {
	inIndex map[domain.VaultID]bool
	err     error
}

func (h vaultsHeld) Has(_ context.Context, id domain.VaultID) (bool, error) {
	return h.inIndex[id], h.err
}

func TestGetIndexStateReturnsIndexedForAVaultInTheIndexAndIndexingForAnyOther(t *testing.T) {
	state := query.NewGetIndexState(vaultsHeld{inIndex: map[domain.VaultID]bool{"garden": true}})

	got, err := state.Execute(t.Context(), domain.Vault{ID: "garden"})
	if err != nil || got.Status != plugindomain.Indexed {
		t.Errorf("a held vault is %+v, %v", got, err)
	}
	got, err = state.Execute(t.Context(), domain.Vault{ID: "orchard"})
	if err != nil || got.Status != plugindomain.Indexing {
		t.Errorf("a vault not held is %+v, %v", got, err)
	}
}

func TestGetIndexStateReturnsTheErrorOfTheIndex(t *testing.T) {
	failure := errors.New("index closed")

	_, err := query.NewGetIndexState(vaultsHeld{err: failure}).Execute(t.Context(), domain.Vault{ID: "garden"})

	if !errors.Is(err, failure) {
		t.Errorf("error = %v, want the index error", err)
	}
}
