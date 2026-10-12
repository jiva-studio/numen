package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

type dialog struct {
	chose string
	did   bool
	err   error
}

func (d dialog) ChooseFolder(context.Context, string, string) (string, bool, error) {
	return d.chose, d.did, d.err
}

var _ port.FolderDialog = dialog{}

func TestTheFolderThePersonChoosesIsReturned(t *testing.T) {
	t.Parallel()

	if got, err := vaults.NewPickFolder(dialog{chose: "/picked", did: true}).Execute(t.Context()); got.Path != "/picked" || !got.IsChosen || err != nil {
		t.Errorf("a picked folder came back as %+v, %v", got, err)
	}
	if got, err := vaults.NewPickFolder(dialog{}).Execute(t.Context()); got.IsChosen || err != nil {
		t.Errorf("a closed dialog came back as %+v, %v", got, err)
	}
	failed := errors.New("no screen")
	if _, err := vaults.NewPickFolder(dialog{err: failed}).Execute(t.Context()); !errors.Is(err, failed) {
		t.Errorf("a dialog that failed answered %v", err)
	}
}
