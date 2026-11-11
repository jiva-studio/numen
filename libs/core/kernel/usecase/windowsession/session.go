package windowsession

import (
	"context"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// vaultSession starts and stops the passes behind the vault of a window.
type vaultSession interface {
	startVault(ctx context.Context, v domain.Vault, rebuild bool)
	stopVault()
	resetStatus()
}

// vaultRunner builds the passes behind one vault.
type vaultRunner interface {
	build(ctx context.Context, v domain.Vault, rebuild bool) *runningVault
}

// Session starts and stops the passes behind the vault of a window. Every scenario that changes what
// runs behind the vault goes through it.
type Session struct {
	runtime       *Runtime
	running       atomic.Pointer[runningVault]
	recorded      OpenRecord
	diag          *Diagnostics
	embedderError error
	runner        vaultRunner
}

// NewSession makes the session of a window over its runtime. embedderError is why the window has no
// embedder, or nil.
func NewSession(
	runtime *Runtime, recorded OpenRecord, diag *Diagnostics, embedderError error, runner *PassRunner,
) *Session {
	return &Session{runtime: runtime, recorded: recorded, diag: diag, embedderError: embedderError, runner: runner}
}

func (s *Session) startVault(ctx context.Context, v domain.Vault, rebuild bool) {
	s.runtime.SetVault(v)
	if v.ID == "" {
		// Nothing is being read, so nothing is waited for.
		s.runtime.SetReady(true)
		return
	}
	// Recorded before the vault is built, so the next window opens on it. A failure
	// to write the list is reported and does not stop the vault from opening.
	if err := s.recorded.Execute(v.ID); err != nil {
		s.diag.say("not recording %s as the vault opened: %v\n", v.Name, err)
	}
	// Published last, so a request that reads a run reads the one belonging to the vault in front of it.
	s.running.Store(s.runner.build(ctx, v, rebuild))
}

func (s *Session) stopVault() {
	on := s.running.Swap(nil)
	if on == nil {
		return
	}
	if on.retire != nil {
		on.retire()
	}
	on.cancel()
	on.wait()
}

func (s *Session) resetStatus() {
	s.runtime.SetReady(false)
	s.runtime.SetError("")
	s.runtime.SetWatchError("")

	for _, pass := range []string{work.WalkingNotes, work.IndexingTexts, work.MakingVectors, work.WordsAlone} {
		s.diag.tasks.RemoveTask(pass)
	}
	if s.embedderError != nil {
		s.diag.tasks.AddTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: s.embedderError.Error()})
	}
}
