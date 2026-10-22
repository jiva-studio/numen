package flashcards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

// carrying is the index as far as this window asks of it: which vaults it holds.
type carrying struct {
	noteport.NoteQueries
	targetVault map[domain.VaultID]bool
}

func (c carrying) Has(_ context.Context, id domain.VaultID) (bool, error) {
	return c.targetVault[id], nil
}

// newAPI is the window over an index that holds the vaults named.
func newAPI(t testing.TB, target ...domain.Vault) *API {
	t.Helper()
	index := carrying{targetVault: map[domain.VaultID]bool{}}
	for _, v := range target {
		index.targetVault[v.ID] = true
	}
	return &API{Window: NewWindow(run.NewTasks()), IsVaultIndexed: note.NewIsVaultIndexed(index)}
}

// newWindowClient is the window itself, over the handler it answers with.
func newWindowClient(t *testing.T, api *API) numenv1connect.WindowServiceClient {
	t.Helper()
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewWindowServiceClient(server.Client(), server.URL)
}
