package identity_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func (m team) firstRecord(tb testing.TB) identity.Record {
	tb.Helper()
	wrapped, err := m.keys.WrapForDevice(m.first.Public())
	if err != nil {
		tb.Fatal(err)
	}
	return identity.Record{Device: m.first, Wrapped: wrapped}
}

func (m team) secondRecord(tb testing.TB) identity.Record {
	tb.Helper()
	wrapped, err := m.keys.WrapForDevice(m.second.Public())
	if err != nil {
		tb.Fatal(err)
	}
	return identity.Record{Device: m.second, Wrapped: wrapped}
}

func TestNewState_JoinsTheRecordToItsChain(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	state, err := identity.NewState(m.firstRecord(t), m.chain)
	if err != nil {
		t.Fatalf("NewState() = %v", err)
	}
	if state.Status != identity.StatusActive || state.Chain.Len() != m.chain.Len() {
		t.Errorf("state = %+v, want an active state over the chain", state)
	}
}

func TestNewState_RefusesADeviceThatIsNotInTheChain(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	stranger := newDeviceOf(t)
	wrapped, err := m.keys.WrapForDevice(stranger.Public())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identity.NewState(identity.Record{Device: stranger, Wrapped: wrapped}, m.chain); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestNewState_ReportsARevokedDeviceAsRevoked(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	state, err := identity.NewState(m.secondRecord(t), chain)
	if err != nil || state.Status != identity.StatusRevoked {
		t.Errorf("status = %v, error = %v, want revoked", state.Status, err)
	}
}

func TestAdopt_AdoptsNewKeysOfLongerChain(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	state, err := identity.NewState(m.firstRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	chain, keys := m.revokeSecond(t)
	next, err := state.Adopt(chain)
	if err != nil {
		t.Fatalf("Adopt() = %v", err)
	}
	opened, err := next.Record.Device.UnwrapAccountKeys(next.Record.Wrapped)
	if err != nil || opened.BoxPublicKey() != keys.BoxPublicKey() {
		t.Errorf("the record does not hold the keys of the new epoch: %v", err)
	}
	if next.Status != identity.StatusActive || next.Chain.Epoch() != 1 {
		t.Errorf("state = %+v, want active at epoch 1", next)
	}
}

func TestAdopt_KeepsTheKeysWhenTheEpochDidNotGrow(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	state, err := identity.NewState(m.firstRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	longer, err := appended(m.chain, m.first, newDeviceOf(t).Public(), "Tablet", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	next, err := state.Adopt(longer)
	if err != nil || string(next.Record.Wrapped.Bytes()) != string(state.Record.Wrapped.Bytes()) || next.Chain.Len() != longer.Len() {
		t.Errorf("Adopt() = %+v, %v, want the same wrapped keys over the longer chain", next, err)
	}
}

func TestAdopt_MarksTheDeviceRevokedWhenTheChainRevokesIt(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	state, err := identity.NewState(m.secondRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	chain, _ := m.revokeSecond(t)
	next, err := state.Adopt(chain)
	if err != nil || next.Status != identity.StatusRevoked {
		t.Errorf("status = %v, error = %v, want revoked", next.Status, err)
	}
	if string(next.Record.Wrapped.Bytes()) != string(state.Record.Wrapped.Bytes()) {
		t.Error("a revoked device got other keys")
	}
}

func TestAdopt_RefusesAChainThatForksFromTheOneOfTheState(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	state, err := identity.NewState(m.firstRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	fork, err := appended(m.chain, m.first, newDeviceOf(t).Public(), "Tablet", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	state, err = state.Adopt(fork)
	if err != nil {
		t.Fatal(err)
	}
	other, err := appended(m.chain, m.first, newDeviceOf(t).Public(), "Desktop", epoch().Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.Adopt(other); !errors.Is(err, identity.ErrNotExtension) {
		t.Errorf("error = %v, want ErrNotExtension", err)
	}
	if _, err := state.Adopt(m.chain); !errors.Is(err, identity.ErrNotExtension) {
		t.Errorf("a shorter chain: error = %v, want ErrNotExtension", err)
	}
}

func TestAdopt_IsIdempotent(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	state, err := identity.NewState(m.firstRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	chain, _, err := m.chain.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, identity.Lost, rand.Reader, epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	once, err := state.Adopt(chain)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := once.Adopt(chain)
	if err != nil || string(twice.Record.Bytes()) != string(once.Record.Bytes()) {
		t.Errorf("adopting the same chain again: %v", err)
	}
}
