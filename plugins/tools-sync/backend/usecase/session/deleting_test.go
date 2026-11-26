package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/accountdata"
)

type deletingPair struct {
	server *synctest.Server
	first  *synctest.Computer
	second *synctest.Computer
}

func newDeletingPair(t *testing.T) *deletingPair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock)
	words := synctest.NewWords(t)
	first := synctest.NewFirstComputer(t, server, clock, "Test", words)
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	deps := accountdata.Deps{Identity: first.Account, Challenges: first.Sessions, Account: first.Cloud, Deletion: &recordedDeletion{}}
	if _, err := accountdata.NewDeleteCloudData(deps).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	return &deletingPair{server: server, first: first, second: second}
}

type recordedDeletion struct{}

func (*recordedDeletion) GetDeletion() (deleteAt time.Time, isDeleting bool) {
	return time.Time{}, false
}
func (*recordedDeletion) SetDeletion(time.Time)         {}
func (*recordedDeletion) Restore(context.Context) error { return nil }
