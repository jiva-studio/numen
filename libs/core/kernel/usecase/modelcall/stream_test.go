package modelcall_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall/modelcalltest"
)

func (s *setup) stream(t *testing.T) (answer, error) {
	t.Helper()
	return s.callAs(t.Context(), consumerID, modelcalltest.ChatOperation)
}

func TestTheFramesOfACallAreStreamedFromTheFirstRowThatAnswers(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, nil, "Let me ", "look.", `{"tool":"list_notes"}`)

	got, err := s.stream(t)

	if err != nil || !slices.Equal(got.Frames, []string{"Let me ", "look.", `{"tool":"list_notes"}`}) || got.Text != "answer from "+addressOne {
		t.Fatalf("answer = %+v, %v", got, err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the second row was called though the first answered")
	}
}

func TestAFailureBeforeTheFirstFrameGoesToTheNextRow(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 0))

	got, err := s.stream(t)

	if err != nil || got.Text != "answer from "+addressTwo {
		t.Fatalf("answer = %+v, %v", got, err)
	}
	if n := len(s.world.Provider.CallsTo(addressOne)); n != 1 {
		t.Errorf("a rate limit was tried %d times, want once", n)
	}
}

func TestAFailureAfterTheFirstFrameIsPassedOnAndNothingIsRepeated(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, tryNext(connection.FailureReasonUnavailable, 0), "partial", "more")

	got, err := s.stream(t)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) || !slices.Equal(failed.Failures, []connection.Failure{{ConnectionName: "First", Reason: "unavailable"}}) {
		t.Fatalf("err = %v, want a failure naming First and its reason", err)
	}
	if !slices.Equal(got.Frames, []string{"partial", "more"}) {
		t.Errorf("frames = %v, want the frames before the failure kept", got.Frames)
	}
	if len(s.world.Provider.CallsTo(addressOne)) != 1 || len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Errorf("calls to One = %d, to Two = %d, want one and none", len(s.world.Provider.CallsTo(addressOne)), len(s.world.Provider.CallsTo(addressTwo)))
	}
}

func TestAPlainErrorAfterTheFirstFrameCountsAsUnavailable(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, errors.New("rpc error: connection reset by Bearer "+keyOne), "partial")

	_, err := s.stream(t)

	if err == nil || err.Error() != "First: unavailable" {
		t.Errorf("err = %v", err)
	}
}

func TestAStopAfterTheFirstFrameNamesTheConnection(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, stop(connection.FailureReasonInvalidRequest), "partial")

	_, err := s.stream(t)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) || failed.Failures[0].Reason != "invalid request" {
		t.Errorf("err = %v", err)
	}
}

func TestAFailureAfterTheFirstFrameSetsTheConnectionAside(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, tryNext(connection.FailureReasonRateLimited, 0), "partial")
	if _, err := s.stream(t); err == nil {
		t.Fatal("the first call did not fail")
	}

	got, err := s.stream(t)

	if err != nil || got.Text != "answer from "+addressTwo {
		t.Errorf("answer = %+v, %v, want the second row after the first was set aside", got, err)
	}
}

func TestAnErrorOfTheConsumerEndsTheCallAndIsReturnedAsItIs(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, nil, "partial")
	consumerFailure := errors.New("the consumer went away")

	_, err := s.calls.Call(t.Context(), consumerID, modelcalldomain.Request{Operation: modelcalltest.ChatOperation},
		func([]byte) error { return consumerFailure })

	if !errors.Is(err, consumerFailure) {
		t.Errorf("err = %v, want the error of the consumer", err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the next row was called after the consumer went away")
	}
}

func TestACanceledStreamReturnsTheErrorOfTheContextAndFreesTheLease(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Block(addressOne)
	ctx, cancel := context.WithCancel(t.Context())
	finish := make(chan error, 1)
	go func() {
		_, err := s.callAs(ctx, consumerID, modelcalltest.ChatOperation)
		finish <- err
	}()
	<-s.world.Provider.Entered()

	cancel()

	select {
	case err := <-finish:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a canceled call did not return")
	}
	if opened, released := len(s.world.Source.Opened()), s.world.Source.Released(); opened != released {
		t.Errorf("opened %d providers and released %d", opened, released)
	}
}

func TestNothingAStreamingConsumerReceivesContainsAKey(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, errors.New("HTTP 401: Authorization: Bearer "+keyOne), "partial")

	got, err := s.stream(t)

	received := strings.Join(got.Frames, "") + got.Text
	if err != nil {
		received += err.Error()
	}
	if strings.Contains(received, "sk-") {
		t.Errorf("the consumer received a key: %q", received)
	}
}

func TestTheSlotOfAConnectionIsHeldForTheLengthOfTheStream(t *testing.T) {
	s := newQueueSetupFor(t, modelcalltest.ChatOperation, 1)
	s.world.Provider.Script(addressLimited, nil, "partial")
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		_, err := s.calls.Call(t.Context(), consumerID, modelcalldomain.Request{Operation: modelcalltest.ChatOperation},
			func([]byte) error {
				<-release
				return nil
			})
		first <- err
	}()
	s.awaitEntered(t)
	second := make(chan error, 1)
	go func() {
		_, err := s.calls.Call(t.Context(), otherConsumerID, modelcalldomain.Request{Operation: modelcalltest.ChatOperation},
			func([]byte) error { return nil })
		second <- err
	}()

	select {
	case <-s.world.Provider.Entered():
		t.Fatal("a second call reached the provider while the first stream was open on a connection of one slot")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)

	if err := awaitDone(t, first); err != nil {
		t.Errorf("first call: %v", err)
	}
	if err := awaitDone(t, second); err != nil {
		t.Errorf("second call: %v", err)
	}
}

func newQueueSetupFor(t *testing.T, operation string, parallelRequests int) *queueSetup {
	t.Helper()
	w := modelcalltest.New()
	entry := w.AddLimited(t, "Limited", addressLimited, parallelRequests)
	for _, plugin := range []string{consumerID, otherConsumerID} {
		w.Rows.Set(plugin, operation, modelcalltest.Row(entry, "m"))
	}
	return &queueSetup{
		world: w, entry: entry,
		calls: modelcall.NewCalls(modelcall.Connections{Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows.ListRows}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside),
	}
}
