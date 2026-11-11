package settings

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/textfake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

type shownVault domain.Vault

func (v shownVault) GetCurrentVault() domain.Vault { return domain.Vault(v) }

const transcription = "org.example.transcription"

func newFolders(t *testing.T, current shownVault) (*Handler, *textfake.TextVersions) {
	t.Helper()
	files := textfake.Readers{"talk.mp3": []byte("audio")}
	folders := &textfake.Folders{}
	versions, err := textfake.NewTextVersions(
		&textfake.Versions{}, folders, textfake.Plugins{transcription}, files, textfake.NewSteppingClock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	segments := []domain.TextSegment{{Text: "Hello.\n"}}
	if _, err := folders.WriteTextFile(t.Context(), transcription, "unrelated.jsonl", segments); err != nil {
		t.Fatal(err)
	}
	if _, err := folders.WriteTextFile(t.Context(), transcription, "k.jsonl", segments); err != nil {
		t.Fatal(err)
	}
	report := plugin.TextPut{
		Path: "talk.mp3", Key: "k", Kind: "transcript", File: "k.jsonl", InputHash: domain.HashContent(files["talk.mp3"]),
	}
	if _, err := versions.PutVersion(t.Context(), domain.Vault(current), transcription, report); err != nil {
		t.Fatal(err)
	}
	data, err := textversion.NewPluginData(versions.Versions, textfake.PluginFolders{Files: folders})
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(Values{}, NewPluginFolders(nil, data, current), &SecretItems{}, &SecretGroups{}), versions
}

func TestDeletePluginDataEmptiesTheFolderOfTheShownVault(t *testing.T) {
	current := shownVault{ID: "v1"}
	handler, versions := newFolders(t, current)
	ask := func() *v1.GetPluginDataResponse {
		t.Helper()
		answer, err := handler.GetPluginData(t.Context(), connect.NewRequest(&v1.GetPluginDataRequest{PluginId: transcription}))
		if err != nil {
			t.Fatal(err)
		}
		return answer.Msg
	}
	if got := ask(); got.GetFileCount() != 2 || got.GetByteCount() == 0 {
		t.Fatalf("the folder holds %v, want two files", got)
	}

	if _, err := handler.DeletePluginData(t.Context(), connect.NewRequest(&v1.DeletePluginDataRequest{PluginId: transcription})); err != nil {
		t.Fatal(err)
	}

	if got := ask(); got.GetFileCount() != 0 {
		t.Fatalf("after the delete the folder holds %v", got)
	}
	listing, err := versions.Texts.ListVersions(t.Context(), domain.Vault(current), "talk.mp3", domain.HashContent([]byte("audio")))
	if err != nil || len(listing.Versions) != 0 {
		t.Fatalf("after the delete the recording has %+v (%v), want no version", listing, err)
	}
}

func TestDeletePluginDataWithNoVaultShownIsRefused(t *testing.T) {
	handler, versions := newFolders(t, shownVault{ID: "v1"})
	handler.vaults = shownVault{}

	_, err := handler.DeletePluginData(t.Context(), connect.NewRequest(&v1.DeletePluginDataRequest{PluginId: transcription}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("answered %v, want a refusal", err)
	}
	listing, err := versions.Texts.ListVersions(t.Context(), domain.Vault{ID: "v1"}, "talk.mp3", domain.HashContent([]byte("audio")))
	if err != nil || len(listing.Versions) != 1 {
		t.Fatalf("the refused delete left %+v (%v), want the version", listing, err)
	}
}
