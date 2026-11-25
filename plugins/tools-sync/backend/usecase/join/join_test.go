package join_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/join"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func newWorld(t *testing.T) (*synctest.Server, *synctest.Clock) {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	return synctest.NewServer(t, clock, fakeserver.Config{}), clock
}

func parseChain(t *testing.T, raw []byte) identity.Chain {
	t.Helper()
	chain, err := identity.ParseChain(raw)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func TestJoin_AddsThisDeviceToTheAccountOfTheRecoveryWords(t *testing.T) {
	server, clock := newWorld(t)
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewPendingComputer(t, server, clock, "Copy", words)

	isJoined, err := join.NewJoin(second.JoinDeps()).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !isJoined {
		t.Fatal("Execute reported no join, want the restore completed")
	}
	if second.Fake.Public().AccountFingerprint != first.Fake.Public().AccountFingerprint {
		t.Error("the second computer joined another account than the first one made")
	}
	joined, original := parseChain(t, second.Fake.Public().Chain), parseChain(t, first.Fake.Public().Chain)
	if joined.Len() != original.Len()+1 || !joined.IsExtensionOf(original) {
		t.Errorf("chain of the second computer has %d entries, want the chain of the first with one device added (%d)", joined.Len(), original.Len())
	}
}

func TestJoin_OpensWhatTheFirstComputerSealed(t *testing.T) {
	server, clock := newWorld(t)
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewPendingComputer(t, server, clock, "Copy", words)
	sealed, err := first.Client.SealData(t.Context(), []byte("context"), []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := join.NewJoin(second.JoinDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	opened, err := second.Client.OpenData(t.Context(), []byte("context"), sealed)
	if err != nil || !bytes.Equal(opened, []byte("secret")) {
		t.Errorf("OpenData = %q, %v; want the secret the first computer sealed", opened, err)
	}
}

func TestJoin_DoesNothingWhenNoRestoreIsPending(t *testing.T) {
	server, clock := newWorld(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", synctest.NewWords(t))
	before := server.Fake.CountCalls(syncv1connect.SignInServiceGetChallengeProcedure)

	isJoined, err := join.NewJoin(first.JoinDeps()).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if isJoined {
		t.Error("Execute reported a join with no restore pending")
	}
	if after := server.Fake.CountCalls(syncv1connect.SignInServiceGetChallengeProcedure); after != before {
		t.Errorf("challenges asked for grew from %d to %d, want no call to the server", before, after)
	}
}

func TestJoin_ReadsTheChainOnceWithAChallengeOfTheRecoveryKey(t *testing.T) {
	server, clock := newWorld(t)
	words := synctest.NewWords(t)
	synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewPendingComputer(t, server, clock, "Copy", words)

	if _, err := join.NewJoin(second.JoinDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	if got := server.Fake.CountCalls(syncv1connect.AccountServiceGetAccountChainByRootProcedure); got != 1 {
		t.Errorf("reads of the chain by root = %d, want 1", got)
	}
}

func TestJoin_MakesANewAccountWhenTheServerHasNone(t *testing.T) {
	server, clock := newWorld(t)
	second := synctest.NewPendingComputer(t, server, clock, "Copy", synctest.NewWords(t))

	isJoined, err := join.NewJoin(second.JoinDeps()).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	public := second.Fake.Public()
	if !isJoined || public.AccountFingerprint == "" || parseChain(t, public.Chain).Len() != 2 {
		t.Errorf("joined = %v, identity = %+v; want a new identity of a genesis entry and this device", isJoined, public)
	}
}

func TestJoin_KeepsTheRestorePendingWhenTheServerIsDown(t *testing.T) {
	server, clock := newWorld(t)
	second := synctest.NewPendingComputer(t, server, clock, "Copy", synctest.NewWords(t))
	server.SetDown(true)

	isJoined, err := join.NewJoin(second.JoinDeps()).Execute(t.Context())

	if !errors.Is(err, session.ErrUnreachable) || isJoined {
		t.Errorf("joined = %v, error = %v; want no join and ErrUnreachable", isJoined, err)
	}
	pending, pendingErr := second.Client.GetPendingRestore(t.Context())
	if pendingErr != nil || !pending.IsPending {
		t.Errorf("pending = %+v, %v; want the restore still pending", pending, pendingErr)
	}
}
