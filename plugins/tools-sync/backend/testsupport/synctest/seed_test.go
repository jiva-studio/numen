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
