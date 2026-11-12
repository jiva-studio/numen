package modelcall_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	connectionuse "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
)

func TestThePersonIsNotifiedWhenARowIsSetAsideAndTheNextOneAnswers(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	want := []modelcalltest.Notice{{ConnectionID: s.one.ID, ConnectionName: "First", Reason: connection.FailureReasonPaymentRequired}}
	if got := s.world.Notices.List(); !slices.Equal(got, want) {
		t.Errorf("notices = %v, want %v", got, want)
	}
}

func TestThePersonIsNotifiedOncePerCooldownOfAConnection(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonRateLimited, 0))

	for range 2 {
		if _, err := s.complete(t); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(s.world.Notices.List()); got != 1 {
		t.Fatalf("notices = %v, want one for the cooldown", s.world.Notices.List())
	}

	s.world.Clock.Advance(2 * time.Minute)
	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	if got := len(s.world.Notices.List()); got != 2 {
		t.Errorf("notices = %v, want a new one after the cooldown ended and the connection failed again", s.world.Notices.List())
	}
}

func TestAnotherReasonOfTheSameConnectionIsNotifiedAgain(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 0))
	s.attempt(t)
	s.world.Clock.Advance(2 * time.Minute)
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.attempt(t)

	got := s.world.Notices.List()

	if len(got) != 2 || got[0].Reason != connection.FailureReasonRateLimited || got[1].Reason != connection.FailureReasonPaymentRequired {
		t.Errorf("notices = %v", got)
	}
}

func TestAnEditedConnectionIsNotifiedAgain(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonKeyRejected, 0))
	s.attempt(t)
	s.world.Edit(t, s.one, "First, new key")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonKeyRejected, 0))

	s.attempt(t)

	if got := len(s.world.Notices.List()); got != 2 {
		t.Errorf("notices = %v, want one for each revision", s.world.Notices.List())
	}
}

func TestThePersonIsNotNotifiedWhenEveryRowFails(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonPaymentRequired, 0))

	s.attempt(t)

	if got := s.world.Notices.List(); len(got) != 0 {
		t.Errorf("notices = %v, want none: the plugin got the error", got)
	}
}

func TestThePersonIsNotNotifiedAboutARowThatWasOnlySkipped(t *testing.T) {
	s := newSetup(t, "complete")
	delete(s.world.Keys.Values, connectionuse.GetEntryName(s.one.ID, "apiKey"))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressTwo {
		t.Fatalf("Complete = %v, %v", result, err)
	}

	if got := s.world.Notices.List(); len(got) != 0 {
		t.Errorf("notices = %v, want none for a row whose key is missing", got)
	}
}

func TestThePersonIsNotNotifiedAboutAStop(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, stop(connection.FailureReasonBadRequest))

	s.attempt(t)

	if got := s.world.Notices.List(); len(got) != 0 {
		t.Errorf("notices = %v", got)
	}
}
