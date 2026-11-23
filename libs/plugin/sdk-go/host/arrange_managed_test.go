package host_test

import (
	"testing"
)

func (a *fakeArrangeHost) setManagedBy(managedBy string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.managedBy = managedBy
}

func (a *fakeArrangeHost) getManagedBy() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.managedBy
}

func TestTheClientAddsAManagedVaultAndSendsTheManager(t *testing.T) {
	a := &fakeArrangeHost{}

	added, err := newFakeArrangeClient(t, a).AddManagedVault(t.Context(), "/shared/Food", "Food", "md.numen.tools.sync")

	if err != nil || added.ID != "v3" || added.Name != "Food" {
		t.Fatalf("the added vault came back as %+v (%v)", added, err)
	}
	if got := a.getManagedBy(); got != "md.numen.tools.sync" {
		t.Errorf("the host received managed_by %q, want the id of the plugin", got)
	}
	if got := a.getRequests(); len(got) != 1 || got[0] != "add /shared/Food Food" {
		t.Errorf("the host received %v, want one add", got)
	}
}

func TestTheClientSendsNoManagerForAnOrdinaryVault(t *testing.T) {
	a := &fakeArrangeHost{}

	if _, err := newFakeArrangeClient(t, a).AddVault(t.Context(), "/vaults/Seas", "Seas"); err != nil {
		t.Fatal(err)
	}

	if got := a.getManagedBy(); got != "" {
		t.Errorf("the host received managed_by %q for an ordinary vault, want none", got)
	}
}
