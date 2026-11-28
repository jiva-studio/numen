package synctest_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

const (
	source domain.VaultID = "Source"
	target domain.VaultID = "Target"
)

func TestPush_ANoteOfTheFirstComputerReachesTheSecondWithoutTheIntervalPassing(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock)
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	b.StartPush(t, time.Hour)
	synctest.WaitUntil(t, "the open stream", func() bool { return server.CountCalls(syncv1connect.WatchServiceWatchChangesProcedure) == 1 })
	a.WriteNote(source, "Shopping list.md", "milk")

	if err := a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}

	synctest.WaitUntil(t, "the note on the second computer", func() bool {
		text, isThere := b.ReadNote(target, "Shopping list.md")
		return isThere && text == "milk"
	})
	clock.Advance(time.Second)
}
