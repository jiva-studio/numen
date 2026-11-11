package media_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"

	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// window shows one vault, which may be none, and gives every file the address of the loopback socket
// it names.
type window struct{ vault domain.Vault }

func (w window) GetCurrentVault() domain.Vault { return w.vault }

func (window) GetMediaAddress(v domain.Vault, stat domain.FileStat) string {
	return "http://127.0.0.1:1/" + string(v.ID) + "/" + stat.Path
}

func (window) GetPluginFileAddress(_ domain.Vault, address file.Address, _ int64) string {
	return "http://127.0.0.1:1/plugin/" + address.PluginID + "/" + address.Path
}

func newHandler(open window) *mediawindow.Handler {
	resolve := file.NewResolveAddress(file.NewStatFile(vaultreader.Readers{}), nil, nil)
	return mediawindow.NewHandler(open, open, resolve)
}

func TestAFileOfTheShownVaultIsPlayedFromTheSocket(t *testing.T) {
	open := window{vault: vaultfixture.NewVault(t, map[string]string{"clips/tide.mp3": "sound"})}

	answer, err := newHandler(open).ResolveAddress(
		t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "clips/tide.mp3"}))
	if err != nil {
		t.Fatal(err)
	}

	if want := "http://127.0.0.1:1/" + string(open.vault.ID) + "/clips/tide.mp3"; answer.Msg.GetSrc() != want {
		t.Errorf("the file plays from %q, want %q", answer.Msg.GetSrc(), want)
	}
}

func TestAWindowShowingNoVaultResolvesNothing(t *testing.T) {
	_, err := newHandler(window{}).ResolveAddress(
		t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "clips/tide.mp3"}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("the refusal is %v, want a failed precondition", err)
	}
}

func TestAnAddressOutsideTheVaultIsRefused(t *testing.T) {
	open := window{vault: vaultfixture.NewVault(t, nil)}

	_, err := newHandler(open).ResolveAddress(
		t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "../secret.mp3"}))

	if err == nil {
		t.Error("an address outside the vault was resolved")
	}
}
