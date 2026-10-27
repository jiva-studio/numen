package settings

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

type shownVault domain.Vault

func (v shownVault) GetCurrentVault() domain.Vault { return domain.Vault(v) }

// folders is the size of each plugin's folder, and the plugins whose data was deleted.
type folders struct {
	sizes   map[string]textversion.PluginDataSize
	deleted []string
}

func (f *folders) GetPluginData(_ context.Context, _ domain.Vault, pluginID string) (textversion.PluginDataSize, error) {
	return f.sizes[pluginID], nil
}

func (f *folders) DeletePluginData(_ context.Context, _ domain.Vault, pluginID string) error {
	f.deleted = append(f.deleted, pluginID)
	delete(f.sizes, pluginID)
	return nil
}

func TestDeletePluginDataEmptiesTheFolderOfTheShownVault(t *testing.T) {
	data := &folders{sizes: map[string]textversion.PluginDataSize{"org.example.transcription": {FileCount: 2, ByteCount: 40}}}
	handler := &Handler{PluginData: data, Vaults: shownVault{ID: "v1"}}
	ask := func() *v1.GetPluginDataResponse {
		t.Helper()
		answer, err := handler.GetPluginData(t.Context(), connect.NewRequest(&v1.GetPluginDataRequest{PluginId: "org.example.transcription"}))
		if err != nil {
			t.Fatal(err)
		}
		return answer.Msg
	}
	if got := ask(); got.GetFileCount() != 2 || got.GetByteCount() != 40 {
		t.Fatalf("the folder holds %v, want two files of 40 bytes", got)
	}

	if _, err := handler.DeletePluginData(t.Context(), connect.NewRequest(&v1.DeletePluginDataRequest{PluginId: "org.example.transcription"})); err != nil {
		t.Fatal(err)
	}
	if got := ask(); got.GetFileCount() != 0 {
		t.Fatalf("after the delete the folder holds %v", got)
	}
}

func TestDeletePluginDataWithNoVaultShownIsRefused(t *testing.T) {
	data := &folders{}
	handler := &Handler{PluginData: data, Vaults: shownVault{}}
	_, err := handler.DeletePluginData(t.Context(), connect.NewRequest(&v1.DeletePluginDataRequest{PluginId: "org.example.transcription"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || len(data.deleted) != 0 {
		t.Fatalf("answered %v and deleted %v, want a refusal", err, data.deleted)
	}
}
