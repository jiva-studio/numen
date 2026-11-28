package synctest_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func TestNewWordsFor_ReturnsTheSameWordsForTheSameLabel(t *testing.T) {
	same := slices.Equal(synctest.NewWordsFor(t, "one"), synctest.NewWordsFor(t, "one"))
	other := slices.Equal(synctest.NewWordsFor(t, "one"), synctest.NewWordsFor(t, "two"))

	if !same || other {
		t.Errorf("same label gives same words = %v, other label gives same words = %v; want true and false", same, other)
	}
}

func TestSeedAccount_JoinsTheSecondAndTheThirdComputerToTheAccountOfTheFirst(t *testing.T) {
	stack := synctest.GetStack(t)
	server := synctest.NewRealServer(t, stack)

	account := synctest.SeedAccount(t, server, synctest.NewClock(synctest.GetNow()), synctest.NewLabel(t))

	first := account.First.Fake.Public()
	for name, computer := range map[string]*synctest.Computer{"second": account.Second, "third": account.Third} {
		if got := computer.Fake.Public().AccountFingerprint; got != first.AccountFingerprint {
			t.Errorf("the %s computer is of account %s, want %s", name, got, first.AccountFingerprint)
		}
	}
	if server.CountCalls(syncv1connect.SignInServiceSignInProcedure) < 3 {
		t.Errorf("sign-ins counted by the proxy = %d, want one for each computer", server.CountCalls(syncv1connect.SignInServiceSignInProcedure))
	}
}

func TestSubscribe_LetsTheAccountPushAFileThatTheOtherComputerReceives(t *testing.T) {
	stack := synctest.GetStack(t)
	server := synctest.NewRealServer(t, stack)
	clock := synctest.NewClock(synctest.GetNow())
	words := synctest.NewWords(t)
	first := synctest.NewUnsubscribedComputer(t, server, clock, string(synctest.FirstVault), words)
	second := synctest.NewSecondComputer(t, server, clock, string(synctest.SecondVault), words)
	first.Subscribe(t, stack, "prod_basic")
	first.Connect(t, synctest.FirstVault)
	first.WriteNote(synctest.FirstVault, "Papers/Report.md", "the text of a report")
	second.ConnectExisting(t, synctest.SecondVault, first.Link(t, synctest.FirstVault).RemoteVaultID)

	if err := first.Sync(t.Context(), synctest.FirstVault); err != nil {
		t.Fatalf("push by the subscribed computer: %v", err)
	}
	if err := second.Sync(t.Context(), synctest.SecondVault); err != nil {
		t.Fatalf("pull by the other computer: %v", err)
	}

	got, ok := second.ReadNote(synctest.SecondVault, "Papers/Report.md")
	if !ok || got != "the text of a report" {
		t.Errorf("file on the other computer = %q (present %v), want the text pushed by the first", got, ok)
	}
}
