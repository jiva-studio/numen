package hosttext_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

type openFailure struct{}

func (openFailure) Open(domain.Vault) (vaultport.VaultReader, error) {
	return nil, errors.New("vault is gone")
}

func showNone() (domain.Vault, bool) { return domain.Vault{}, false }

func TestEveryCallWithNoVaultShownIsNoText(t *testing.T) {
	texts := hosttext.NewTexts(nil, openFailure{}, showNone)
	ctx := t.Context()
	_, putErr := texts.PutVersion(ctx, "engines.transcription", pluginport.PluginTextPut{Path: "talk.mp3", Key: "k"})
	removeErr := texts.RemoveVersion(ctx, "engines.transcription", "talk.mp3", "k")
	_, listErr := texts.ListVersions(ctx, "talk.mp3")
	_, readErr := texts.ReadText(ctx, "talk.mp3", "engines.transcription/k", "")
	for name, err := range map[string]error{"put": putErr, "remove": removeErr, "list": listErr, "read": readErr} {
		if !errors.Is(err, textport.ErrNoText) {
			t.Errorf("%s: got %v, want no text", name, err)
		}
	}
}

func TestListVersionsReturnsTheReaderFailure(t *testing.T) {
	getVault := func() (domain.Vault, bool) { return domain.Vault{ID: "v1"}, true }
	texts := hosttext.NewTexts(nil, openFailure{}, getVault)
	if _, err := texts.ListVersions(t.Context(), "talk.mp3"); err == nil || errors.Is(err, textport.ErrNoText) {
		t.Fatalf("got %v, want the reader's failure", err)
	}
}
