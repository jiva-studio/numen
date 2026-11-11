package file_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// clipStores is the store of each plugin, holding the files it is given by name.
type clipStores map[string]map[string]string

func (s clipStores) Open(_ domain.Vault, pluginID string) (pluginport.PluginStoreReader, error) {
	return clipStore{files: s[pluginID]}, nil
}

func (clipStores) DeleteFolder(context.Context, domain.Vault, string) error { return nil }

type clipStore struct {
	pluginport.PluginStoreReader
	files map[string]string
}

type clipBody struct{ *strings.Reader }

func (clipBody) Close() error { return nil }

func (s clipStore) Open(_ context.Context, name string) (io.ReadSeekCloser, int64, error) {
	content, ok := s.files[name]
	if !ok {
		return nil, 0, fs.ErrNotExist
	}
	return clipBody{strings.NewReader(content)}, int64(len(content)), nil
}

func newResolveAddress(t *testing.T) (*file.ResolveAddress, domain.Vault) {
	t.Helper()
	v := vaultfixture.NewVault(t, map[string]string{"talks/river.mp3": "sound"})
	stores := clipStores{"org.example.links": {"river.mp4": "a clip"}}
	return file.NewResolveAddress(file.NewStatFile(vaultreader.Readers{}), file.NewOpenPluginFile(stores), nil), v
}

func TestResolveAddressFindsAFileOfTheVault(t *testing.T) {
	resolve, v := newResolveAddress(t)

	got, err := resolve.Execute(t.Context(), v, "talks/river.mp3")

	if err != nil || got.Address.IsPluginFile() || got.Stat.Path != "talks/river.mp3" || got.Size != 5 {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestResolveAddressFindsAFileOfAPluginStore(t *testing.T) {
	resolve, v := newResolveAddress(t)

	got, err := resolve.Execute(t.Context(), v, "plugin://org.example.links/river.mp4")

	if err != nil || got.Address != (file.Address{PluginID: "org.example.links", Path: "river.mp4"}) || got.Size != 6 {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestResolveAddressRejectsAMalformedAddressAndAMissingFile(t *testing.T) {
	resolve, v := newResolveAddress(t)

	if _, err := resolve.Execute(t.Context(), v, "plugin://org.example.links"); !errors.Is(err, file.ErrInvalidAddress) {
		t.Errorf("a malformed address: %v, want ErrInvalidAddress", err)
	}
	if _, err := resolve.Execute(t.Context(), v, "plugin://org.example.links/gone.mp4"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing plugin file: %v, want fs.ErrNotExist", err)
	}
}
