package session_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/accountdata"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

type deletingPair struct {
	server *synctest.Server
	first  *synctest.Computer
	second *synctest.Computer
	keep   *session.Keep
}

func newDeletingPair(t *testing.T) *deletingPair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock)
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	keep := session.NewKeep(session.Deps{Identity: first.Account, Server: first.Sessions, Health: first.NewHealth(), Clock: first.Clock})
	if err := keep.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	deps := accountdata.Deps{Identity: first.Account, Challenges: first.Sessions, Account: first.Cloud, Deletion: keep}
	if err := accountdata.NewDeleteCloudData(deps).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	return &deletingPair{server: server, first: first, second: second, keep: keep}
}
