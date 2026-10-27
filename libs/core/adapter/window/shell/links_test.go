package shell

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// linkIndex resolves the names it holds, in the vault it is queried for.
type linkIndex struct {
	notes   map[string]string
	queried []domain.VaultID
}

func (*linkIndex) GetLinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, nil
}

func (*linkIndex) GetBacklinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, nil
}

func (l *linkIndex) ResolveLinks(
	_ context.Context, vaultID domain.VaultID, _ string, content []string,
) (map[string]domain.ResolvedLink, error) {
	l.queried = append(l.queried, vaultID)
	out := map[string]domain.ResolvedLink{}
	for _, one := range content {
		if path, isFound := l.notes[one]; isFound {
			out[one] = domain.ResolvedLink{To: path, ToVault: vaultID}
		}
	}
	return out, nil
}

func newNoteClient(t *testing.T, index *linkIndex) numenv1connect.NoteServiceClient {
	t.Helper()
	api := newAPI(t)
	api.ListVaults = func() ([]domain.Vault, error) {
		return []domain.Vault{{ID: "garden", Name: "Garden", Path: "/vaults/garden"}}, nil
	}
	api.ResolveLinks = note.NewResolveLinks(index)
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewNoteServiceClient(server.Client(), server.URL)
}

func TestResolveAddressesAnswersInTheNamedVault(t *testing.T) {
	index := &linkIndex{notes: map[string]string{"name://Compost": "Compost.md"}}
	client := newNoteClient(t, index)

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		Vault: "garden",
		Text:  "Made like [[Compost|compost]], unlike [[Peat]].",
	}))
	if err != nil {
		t.Fatal(err)
	}

	var raws []string
	for _, one := range answer.Msg.GetFound() {
		raws = append(raws, one.GetRaw())
	}
	if len(raws) != 2 || raws[0] != "[[Compost|compost]]" || raws[1] != "[[Peat]]" {
		t.Errorf("found %q", raws)
	}
	resolved := answer.Msg.GetResolved()
	if len(resolved) != 1 || resolved[0].GetWritten() != "name://Compost" || resolved[0].GetPath() != "Compost.md" {
		t.Errorf("resolved %v", resolved)
	}
	if len(index.queried) != 1 || index.queried[0] != "garden" {
		t.Errorf("the index was queried for %v", index.queried)
	}
}

func TestResolveAddressesRejectsARequestNamingNoVault(t *testing.T) {
	client := newNoteClient(t, &linkIndex{})

	_, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{Text: "[[A]]"}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("got %v, want invalid argument", err)
	}
}

func TestResolveAddressesRejectsAnUnknownVault(t *testing.T) {
	client := newNoteClient(t, &linkIndex{})

	_, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		Vault: "elsewhere", Text: "[[A]]",
	}))

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("got %v, want not found", err)
	}
}
