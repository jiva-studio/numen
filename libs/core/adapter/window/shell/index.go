package flashcards

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// IndexVault is a vault walked into the index.
type IndexVault func(ctx context.Context, v domain.Vault) error

// indexRuns is which vaults this window has indexed: the ones indexed
// since it opened, the ones being indexed now, and why the last indexing of one
// failed.
type indexRuns struct {
	mu sync.Mutex

	// indexVault is how a vault is brought up to date in the index, and ctx is the
	// life those runs have.
	indexVault IndexVault
	ctx        context.Context

	indexed  map[domain.VaultID]bool
	running  map[domain.VaultID]bool
	failures map[domain.VaultID]string
}

// setIndexer is how a vault is indexed from now on, and the life those runs have.
func (r *indexRuns) setIndexer(ctx context.Context, read IndexVault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ctx, r.indexVault = ctx, read
}

// clear lets go of why a vault could not be indexed, and that it was indexed at all.
func (r *indexRuns) clear(id domain.VaultID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.failures, id)
	delete(r.indexed, id)
}

// tryStart claims a vault for indexing. It returns the indexer and the life to
// run it under where this call is the one that begins it, and no indexer where
// the vault has been indexed, is being indexed, could not be indexed, or where this
// window indexes nothing.
func (r *indexRuns) tryStart(
	v domain.Vault,
) (read IndexVault, limit context.Context, underway bool, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.indexed[v.ID] {
		return nil, nil, false, ""
	}
	if why, hasFailure := r.failures[v.ID]; hasFailure {
		return nil, nil, false, why
	}
	if r.running[v.ID] {
		return nil, nil, true, ""
	}
	if r.indexVault == nil {
		return nil, nil, false, ""
	}
	if r.running == nil {
		r.running = make(map[domain.VaultID]bool)
	}
	r.running[v.ID] = true
	return r.indexVault, r.ctx, true, ""
}

// record is what one indexing came to: the vault is indexed, or it is why it is not.
func (r *indexRuns) record(v domain.Vault, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, v.ID)
	if err != nil {
		if r.failures == nil {
			r.failures = make(map[domain.VaultID]string)
		}
		r.failures[v.ID] = err.Error()
		return
	}
	if r.indexed == nil {
		r.indexed = make(map[domain.VaultID]bool)
	}
	r.indexed[v.ID] = true
}

// SetVaultIndexer is how a vault is brought up to date in the index, and the life those
// runs have. A window naming none counts a vault from the index as it
// stands.
func (a *API) SetVaultIndexer(ctx context.Context, read IndexVault) {
	a.indexRuns.setIndexer(ctx, read)
}

// startIndexing brings a vault up to date in the index. It says whether an indexing of
// that vault is running now, and why the last one failed, when one did.
//
// A vault is indexed once for the life of the window, and the watcher carries it
// from there. One vault is indexed once at a time however many counts ask for it,
// and an indexing that failed is not begun again until the vault moves.
func (a *API) startIndexing(ctx context.Context, v domain.Vault) (underway bool, reason string) {
	read, item, running, reason := a.indexRuns.tryStart(v)
	if read == nil {
		return running, reason
	}
	//nolint:contextcheck // a vault is indexed for the life of the window, under behind, not under the count that asked
	go a.runIndexing(item, read, v, a.IsVaultIndexed.Execute(ctx, v))
	return true, ""
}

// runIndexing is one vault indexed, reported as work for as long as it runs. An indexing
// that finished wakes the counts, which is what puts the vault's numbers on the
// page.
//
// held says the index already carries the vault. A vault it does not carry has
// nothing to show until this is done and the person is waiting on it, so that
// work is drawn the moment it begins; a vault it carries is work drawn once it
// has lasted.
func (a *API) runIndexing(ctx context.Context, read IndexVault, v domain.Vault, target bool) {
	pos := run.Task{
		ID:            "indexing\t" + string(v.ID),
		Title:         "Indexing the vault",
		About:         v.Name,
		IsUserStarted: !target,
	}
	a.reportTask(pos)

	err := read(ctx, v)

	a.indexRuns.record(v, err)

	if err != nil {
		pos.Error = err.Error()
		a.reportTask(pos)
		return
	}
	a.finishTask(pos.ID)
	a.ReportChange()
}

// ClearIndexFailure lets go of why a vault could not be indexed. A vault that moved
// underneath the window is indexed again.
func (a *API) ClearIndexFailure(vaultID domain.VaultID) {
	a.indexRuns.clear(vaultID)
}

// EnsureIndexed brings a vault into the index when it is not there and says where that stands, which
// is what a plugin counting the vault is told. A window that reads no vault says that the index does
// not carry the ones it does not.
func (a *API) EnsureIndexed(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error) {
	underway, reason := a.startIndexing(ctx, v)
	switch {
	case underway:
		return pluginport.PluginIndexState{Status: pluginport.PluginIndexing}, nil
	case reason != "":
		return pluginport.PluginIndexState{Status: pluginport.PluginIndexFailed, Reason: reason}, nil
	}
	if !a.IsVaultIndexed.Execute(ctx, v) {
		return pluginport.PluginIndexState{
			Status: pluginport.PluginIndexFailed, Reason: "the index does not carry this vault",
		}, nil
	}
	return pluginport.PluginIndexState{Status: pluginport.PluginIndexed}, nil
}
