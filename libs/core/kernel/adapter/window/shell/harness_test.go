package shell

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// carrying is the index as this window uses it: it reports which vaults it
// holds.
type carrying struct {
	targetVault map[domain.VaultID]bool
}

func (c carrying) Has(_ context.Context, id domain.VaultID) (bool, error) {
	return c.targetVault[id], nil
}

// testWindow is the name the window under test answers to.
const testWindow = "test"

func newAPI(tb testing.TB, target ...domain.Vault) *API {
	tb.Helper()
	index := carrying{targetVault: map[domain.VaultID]bool{}}
	for _, v := range target {
		index.targetVault[v.ID] = true
	}
	api := &API{Window: NewWindow(testWindow, run.NewTasks())}
	api.Ensure = indexing.NewEnsureIndexed(&api.Runs, query.NewIsVaultIndexed(index), api.Window, api)
	return api
}

func newWindowClient(t *testing.T, api *API) numenv1connect.WindowServiceClient {
	t.Helper()
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewWindowServiceClient(server.Client(), server.URL)
}
