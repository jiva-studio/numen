package identity_test

import "testing"

func TestFindFork_ReportsTheEntriesThatTheRecoveryKeyVoids(t *testing.T) {
	t.Parallel()
	x := newTakeover(t)

	fork, ok := x.recovery.FindFork(x.stolen)

	if !ok || fork.Voided != x.stolen.Len()-x.team.chain.Len() {
		t.Errorf("fork = %+v, %v, want the entries of the stolen branch after the shared prefix voided (prefix %d, branch %d)", fork, ok, x.team.chain.Len(), x.stolen.Len())
	}
	if _, ok := x.stolen.FindFork(x.recovery); ok {
		t.Error("a chain signed by a device replaces the chain of the recovery key")
	}
}

func TestFindFork_FindsNothingInAnExtension(t *testing.T) {
	t.Parallel()
	x := newTakeover(t)

	if fork, ok := x.stolen.FindFork(x.team.chain); ok {
		t.Errorf("fork = %+v, want none for a chain that extends the other", fork)
	}
}
