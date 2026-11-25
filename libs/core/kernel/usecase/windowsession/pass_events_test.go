package windowsession

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestUnwatchedVaultReportsFailureAndLifecycle(t *testing.T) {
	problem := errors.New("watch unavailable")
	events := make(chan string, 3)
	tasks := &taskList{}
	diag := NewDiagnostics(io.Discard, tasks)
	runner := NewPassRunner(PassRunnerConfig{
		Failures: &Runtime{}, Diag: diag,
		NewSession: func(domain.Vault) IndexSession {
			return IndexSession{Run: func(context.Context, bool, ReadNotes) {}}
		},
		Reporter: NewChangeReporter(&changeLog{}, nil, nil, diag),
		Passes: VaultPassesFunc(func(context.Context, domain.Vault, bool, Reporters) StartedPasses {
			return StartedPasses{Unwatched: problem, Run: func(context.Context) {}}
		}),
		Report: func(_ domain.Vault, event string, err error) {
			if event == "Vault watch unavailable" && !errors.Is(err, problem) {
				t.Errorf("watch error = %v", err)
			}
			events <- event
		},
	})
	run := runner.build(t.Context(), garden(), false, runner.newSession(garden()))
	run.cancel()
	run.wait()
	for _, want := range []string{"Vault watch unavailable", "Vault opened", "Vault closed"} {
		if got := <-events; got != want {
			t.Errorf("event = %q, want %q", got, want)
		}
	}
}
