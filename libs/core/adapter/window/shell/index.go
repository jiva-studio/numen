package shell

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// IndexVault is a vault walked into the index.
type IndexVault func(ctx context.Context, v domain.Vault) error

// indexRuns is which vaults this window has indexed: the ones indexed
// since it opened, the ones being indexed now, and why the last indexing of one
// failed.
type indexRuns struct {
	mu sync.Mutex

	// indexVault is how a vault is brought up to date in the index, and closing
	// signals when those runs end.
	indexVault IndexVault
	closing    <-chan struct{}

	indexed  map[domain.VaultID]bool
	running  map[domain.VaultID]bool
	failures map[domain.VaultID]string
}

// setIndexer is how a vault is indexed from now on, and the life those runs have.
func (r *indexRuns) setIndexer(ctx context.Context, read IndexVault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx != nil {
		r.closing = ctx.Done()
	} else {
		r.closing = nil
	}
	r.indexVault = read
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
) (read IndexVault, closing <-chan struct{}, underway bool, reason string) {
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
	return r.indexVault, r.closing, true, ""
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

// SetVaultIndexer sets how a vault is brought up to date in the index, and the lifecycle of
// those runs. A window without an indexer counts a vault from the index as it
// currently is.
func (a *API) SetVaultIndexer(ctx context.Context, read IndexVault) {
	a.indexRuns.setIndexer(ctx, read)
}

// startIndexing brings a vault up to date in the index. It reports whether an indexing of
// that vault is running, and why the last one failed, if it did.
//
// A vault is indexed once for the lifetime of the window, and the watcher keeps it current
// afterwards. One vault is indexed at most once at a time however many counts request it,
// and a failed indexing is not restarted until the vault changes.
func (a *API) startIndexing(ctx context.Context, v domain.Vault) (underway bool, reason string) {
	read, closing, running, reason := a.indexRuns.tryStart(v)
	if read == nil {
		return running, reason
	}
	isIndexed := a.IsVaultIndexed.Execute(ctx, v)
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	go func() {
		defer cancel()
		if closing != nil {
			go func() {
				select {
				case <-closing:
					cancel()
				case <-runCtx.Done():
				}
			}()
		}
		a.runIndexing(runCtx, read, v, isIndexed)
	}()
	return true, ""
}

// runIndexing indexes one vault and reports it as work for as long as it runs. A finished
// indexing wakes the counts, which puts the vault's numbers on the page.
//
// target is true when the index already carries the vault. A vault the index does not
// carry has nothing to show until indexing completes, so its work is displayed
// immediately; for a carried vault the work is displayed only once it has lasted.
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

// EnsureIndexed brings a vault into the index when it is missing and reports its state, which
// is what a plugin counting the vault is told. A window that reads no vault reports that the
// index does not carry the vaults it does not read.
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
