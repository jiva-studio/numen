package shell

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// IndexVault is a vault walked into the index.
type IndexVault func(ctx context.Context, v domain.Vault) error

// indexRuns is which vaults this window has indexed: the ones indexed
// since it opened, the ones being indexed now, and why the last indexing of one
// failed.
type indexRuns struct {
	mu sync.Mutex

	indexVault IndexVault
	closing    <-chan struct{}

	indexed  map[domain.VaultID]bool
	running  map[domain.VaultID]bool
	failures map[domain.VaultID]string
}

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

func (r *indexRuns) clear(id domain.VaultID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.failures, id)
	delete(r.indexed, id)
}

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

// ClearIndexFailure releases the reason why a vault could not be indexed. A vault that moved
// underneath the window is indexed again.
func (a *API) ClearIndexFailure(vaultID domain.VaultID) {
	a.indexRuns.clear(vaultID)
}

// EnsureIndexed brings a vault into the index when it is missing and reports its state, which
// is the state a plugin counting the vault receives. A window that reads no vault reports that the
// index does not carry the vaults it does not read.
func (a *API) EnsureIndexed(ctx context.Context, v domain.Vault) (plugin.IndexState, error) {
	underway, reason := a.startIndexing(ctx, v)
	switch {
	case underway:
		return plugin.IndexState{Status: plugin.Indexing}, nil
	case reason != "":
		return plugin.IndexState{Status: plugin.IndexFailed, Reason: reason}, nil
	}
	if !a.IsVaultIndexed.Execute(ctx, v) {
		return plugin.IndexState{
			Status: plugin.IndexFailed, Reason: "the index does not contain this vault",
		}, nil
	}
	return plugin.IndexState{Status: plugin.Indexed}, nil
}
