package modelcall_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

func TestTheTimeoutsAreTheDecidedOnes(t *testing.T) {
	if modelcall.DefaultCooldown != 60*time.Second || modelcall.MaxCooldown != 10*time.Minute {
		t.Errorf("DefaultCooldown = %v, MaxCooldown = %v, want 60s and 10m", modelcall.DefaultCooldown, modelcall.MaxCooldown)
	}
}

func callsToFirst(s *setup) int { return len(s.world.Provider.CallsTo(addressOne)) }

func TestARowSetAsideIsSkippedFor60SecondsWhenTheServiceGaveNoWait(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 0))
	s.attempt(t)

	s.world.Clock.Advance(59 * time.Second)
	s.attempt(t)
	if got := callsToFirst(s); got != 1 {
		t.Fatalf("the first row was called %d times within 59 s, want 1", got)
	}

	s.world.Clock.Advance(2 * time.Second)
	s.attempt(t)
	if got := callsToFirst(s); got != 2 {
		t.Errorf("the first row was called %d times after 61 s, want 2", got)
	}
}

func TestTheWaitTheServiceAskedForReplacesTheDefault(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 5*time.Second))
	s.attempt(t)

	s.world.Clock.Advance(6 * time.Second)
	s.attempt(t)

	if got := callsToFirst(s); got != 2 {
		t.Errorf("the first row was called %d times after the 5 s asked for, want 2", got)
	}
}

func TestAWaitOfMoreThanTenMinutesIsLimitedToTenMinutes(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 3*time.Hour))
	s.attempt(t)

	s.world.Clock.Advance(9 * time.Minute)
	s.attempt(t)
	if got := callsToFirst(s); got != 1 {
		t.Fatalf("the first row was called %d times within 9 minutes, want 1", got)
	}

	s.world.Clock.Advance(2 * time.Minute)
	s.attempt(t)
	if got := callsToFirst(s); got != 2 {
		t.Errorf("the first row was called %d times after 11 minutes, want 2", got)
	}
}

func TestARejectedKeyOrAPaymentRequiredLastsUntilTheConnectionIsEdited(t *testing.T) {
	for _, reason := range []connection.FailureReason{connection.FailureReasonRejectedKey, connection.FailureReasonPaymentRequired} {
		s := newSetup(t, "complete")
		s.world.Provider.Fail(addressOne, tryNext(reason, 0))
		s.attempt(t)

		s.world.Clock.Advance(30 * 24 * time.Hour)
		s.attempt(t)
		if got := callsToFirst(s); got != 1 {
			t.Fatalf("%v: the first row was called %d times a month later, want 1", reason, got)
		}

		s.world.Edit(t, s.one, "First, new key")
		result, err := s.complete(t)
		if err != nil || result.Text != "answer from "+addressOne {
			t.Errorf("%v: after the edit: %v, %v", reason, result, err)
		}
	}
}

func TestACooldownIsTrackedPerConnection(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 0))
	s.attempt(t)

	s.world.Provider.Fail(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))
	_, err := s.complete(t)

	if err == nil || err.Error() != "First: rate limited; Second: rate limited" {
		t.Errorf("err = %v", err)
	}
	if got := len(s.world.Provider.CallsTo(addressTwo)); got != 2 {
		t.Errorf("the second row was called %d times, want 2", got)
	}
}

func TestACooldownIsSharedByEveryPluginThatUsesTheConnection(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Rows.Set("md.example.other-consumer", "complete", modelcalltestRow(s))
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.attempt(t)

	_, err := s.calls.Complete(t.Context(), "md.example.other-consumer", modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)

	if err == nil || err.Error() != "First: payment required" {
		t.Errorf("err = %v", err)
	}
	if got := callsToFirst(s); got != 1 {
		t.Errorf("the first row was called %d times, want 1", got)
	}
}

func modelcalltestRow(s *setup) connection.Row {
	return connection.Row{ConnectionID: s.one.ID, Model: "m"}
}

func TestATimeoutIsTriedOnceMoreBeforeTheRowIsSetAside(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonUnavailable, 0))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressOne || callsToFirst(s) != 2 {
		t.Errorf("Complete = %v, %v after %d calls, want the retry to answer", result, err, callsToFirst(s))
	}
	if got := s.world.Notices.List(); len(got) != 0 {
		t.Errorf("notices = %v after a retry that answered", got)
	}
}

func TestAnUnreachableServiceIsTriedOnceMoreBeforeTheRowIsSetAside(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonUnavailable, 0), tryNext(connection.FailureReasonUnavailable, 0))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressTwo || callsToFirst(s) != 2 {
		t.Errorf("Complete = %v, %v after %d calls, want two tries and then the next row", result, err, callsToFirst(s))
	}
}

func TestOnlyOneRetryIsMade(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonUnavailable, 0))

	s.attempt(t)

	if got := callsToFirst(s); got != 2 {
		t.Errorf("the first row was called %d times, want 2", got)
	}
}

func TestAReasonOtherThanATimeoutOrAnUnreachableServiceIsNotRetried(t *testing.T) {
	for _, reason := range []connection.FailureReason{
		connection.FailureReasonRejectedKey, connection.FailureReasonPaymentRequired,
		connection.FailureReasonRateLimited, connection.FailureReasonUnavailable,
	} {
		s := newSetup(t, "complete")
		s.world.Provider.FailAlways(addressOne, tryNext(reason, 0))

		s.attempt(t)

		if got := callsToFirst(s); got != 1 {
			t.Errorf("%v: the first row was called %d times, want 1", reason, got)
		}
	}
}

func TestARowSetAsideAfterARetryStartsItsCooldown(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonUnavailable, 0))
	s.attempt(t)

	s.attempt(t)

	if got := callsToFirst(s); got != 2 {
		t.Errorf("the first row was called %d times, want 2 (one call and its retry)", got)
	}
}
