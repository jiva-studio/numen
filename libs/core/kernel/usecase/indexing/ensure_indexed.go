package indexing

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// noIndexReason is why a vault is failed where nothing indexes it.
const noIndexReason = "the index does not contain this vault"

type IndexVault func(ctx context.Context, v domain.Vault) error

// VaultIndexed reports whether the index already contains a vault.
type VaultIndexed interface {
	Execute(ctx context.Context, v domain.Vault) bool
}

// Tasks is the list of what the window is doing.
type Tasks interface {
	AddTask(task work.Task)
	RemoveTask(id string)
}

// Changes tells the views drawing the vaults that data they render from has changed.
type Changes interface {
	ReportChange()
}

// RunState is which vaults a window has indexed: the ones indexed since it opened, the ones being indexed
// now, and why the last indexing of one failed.
type RunState interface {
	// Claim reserves the indexing of the vault for the caller, which starts it only when read is not nil.
	// Otherwise underway says an indexing of the vault runs now, and reason why the last one failed.
	Claim(v domain.Vault) (read IndexVault, closing <-chan struct{}, underway bool, reason string)
	// Record ends the indexing Claim reserved, with the error it ended with.
	Record(v domain.Vault, err error)
	// Clear forgets how the vault was indexed, so the next claim starts again.
	Clear(id domain.VaultID)
}

// EnsureIndexed brings a vault into the index when it is missing and reports its state, which is the
// state a plugin counting the vault receives. The indexing runs in the background and is shown as work.
type EnsureIndexed struct {
	runs    RunState
	indexed VaultIndexed
	tasks   Tasks
	changes Changes
}

// NewEnsureIndexed returns the use case over the state of the indexing runs, the question whether the
// index holds a vault, the list of work and the notification of changes.
func NewEnsureIndexed(runs RunState, indexed VaultIndexed, tasks Tasks, changes Changes) *EnsureIndexed {
	return &EnsureIndexed{runs: runs, indexed: indexed, tasks: tasks, changes: changes}
}

// Execute starts the indexing of the vault when the index lacks it and no run has failed or is underway,
// and reports the state: indexing while it runs, failed with the reason of the failed run or when the index
// does not contain the vault, indexed otherwise. A window that reads no vault reports that the index does
// not carry the vaults it does not read.
func (u *EnsureIndexed) Execute(ctx context.Context, v domain.Vault) (plugin.IndexState, error) {
	if u == nil {
		return plugin.IndexState{Status: plugin.IndexFailed, Reason: noIndexReason}, nil
	}
	underway, reason := u.start(ctx, v)
	switch {
	case underway:
		return plugin.IndexState{Status: plugin.Indexing}, nil
	case reason != "":
		return plugin.IndexState{Status: plugin.IndexFailed, Reason: reason}, nil
	}
	if !u.indexed.Execute(ctx, v) {
		return plugin.IndexState{
			Status: plugin.IndexFailed, Reason: noIndexReason,
		}, nil
	}
	return plugin.IndexState{Status: plugin.Indexed}, nil
}

func (u *EnsureIndexed) start(ctx context.Context, v domain.Vault) (underway bool, reason string) {
	read, closing, running, reason := u.runs.Claim(v)
	if read == nil {
		return running, reason
	}
	isIndexed := u.indexed.Execute(ctx, v)
	runCtx, cancel := context.WithCancel(plugin.WithBackground(context.WithoutCancel(ctx), plugin.WhoIndexing))
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
		u.run(runCtx, read, v, isIndexed)
	}()
	return true, ""
}

func (u *EnsureIndexed) run(ctx context.Context, read IndexVault, v domain.Vault, target bool) {
	pos := work.Task{
		ID:            "indexing\t" + string(v.ID),
		Title:         "Indexing the vault",
		About:         v.Name,
		IsUserStarted: !target,
	}
	u.tasks.AddTask(pos)

	err := read(ctx, v)

	u.runs.Record(v, err)

	if err != nil {
		pos.Error = err.Error()
		u.tasks.AddTask(pos)
		return
	}
	u.tasks.RemoveTask(pos.ID)
	u.changes.ReportChange()
}

// Runs is the RunState of one window, in memory.
type Runs struct {
	mu sync.Mutex

	indexVault IndexVault
	closing    <-chan struct{}

	indexed  map[domain.VaultID]bool
	running  map[domain.VaultID]bool
	failures map[domain.VaultID]string
}

// SetIndexer sets how a vault is brought up to date in the index, and the lifecycle of those runs: they
// end when ctx does. Without an indexer no vault is claimed.
func (r *Runs) SetIndexer(ctx context.Context, read IndexVault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx != nil {
		r.closing = ctx.Done()
	} else {
		r.closing = nil
	}
	r.indexVault = read
}

// Clear implements RunState.
func (r *Runs) Clear(id domain.VaultID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.failures, id)
	delete(r.indexed, id)
}

// Claim implements RunState.
func (r *Runs) Claim(v domain.Vault) (read IndexVault, closing <-chan struct{}, underway bool, reason string) {
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

// Record implements RunState.
func (r *Runs) Record(v domain.Vault, err error) {
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
