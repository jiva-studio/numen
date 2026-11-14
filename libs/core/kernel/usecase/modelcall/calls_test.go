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

const (
	consumerID = "md.example.consumer"
	addressOne = "https://one.test/v1"
	addressTwo = "https://two.test/v1"
	keyOne     = "sk-one-0123456789"
	keyTwo     = "sk-two-0123456789"
)

type setup struct {
	world *modelcalltest.World
	calls *modelcall.Calls
	one   modelcalltest.Entry
	two   modelcalltest.Entry
}

func newSetup(t *testing.T, operation string) *setup {
	t.Helper()
	w := modelcalltest.New()
	one := w.AddServer(t, "First", addressOne, keyOne)
	two := w.AddServer(t, "Second", addressTwo, keyTwo)
	w.Rows.Set(consumerID, operation, modelcalltest.Row(one, "model-one"), modelcalltest.Row(two, "model-two"))
	return &setup{
		world: w, one: one, two: two,
		calls: modelcall.NewCalls(modelcall.Connections{Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows.ListRows}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside),
	}
}

// answer is what a consumer received from a call: the data frames and the done payload.
type answer struct {
	Frames []string
	Text   string
}

func (s *setup) complete(t *testing.T) (answer, error) {
	t.Helper()
	return s.callAs(t.Context(), consumerID, modelcalltest.CompleteOperation)
}

func (s *setup) callAs(ctx context.Context, pluginID, operation string) (answer, error) {
	var received answer
	done, err := s.calls.Call(ctx, pluginID, modelcalldomain.Request{
		Operation: operation, Payload: []byte("fix this"), Priority: modelcalldomain.PriorityBackground,
	}, func(data []byte) error {
		received.Frames = append(received.Frames, string(data))
		return nil
	})
	received.Text = string(done)
	return received, err
}

func tryNext(reason connection.FailureReason, retryAfter time.Duration) error {
	return &connection.TryNextError{Reason: reason, RetryAfter: retryAfter}
}

func stop(reason connection.FailureReason) error {
	return &connection.StopError{Reason: reason}
}

func TestTheFirstRowThatAnswersServesTheCall(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressOne {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the second row was called though the first answered")
	}
}

func TestTheCallCarriesTheSnapshotOfTheRow(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, connection.Row{ConnectionID: s.one.ID, Options: map[string]string{"model": "model-one", "reasoning": "low"}})

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	got := s.world.Provider.CallsTo(addressOne)[0].Call
	if got.Type != "server" || got.ConnectionID != s.one.ID || got.Options["model"] != "model-one" || got.Options["reasoning"] != "low" ||
		got.Fields["baseUrl"] != addressOne || got.Secrets["apiKey"] != keyOne {
		t.Errorf("call = %+v", got)
	}
}

func TestAFailureToTryTheNextGoesToTheNextRow(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 0))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressTwo {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	if got := len(s.world.Provider.CallsTo(addressOne)); got != 1 {
		t.Errorf("a rate limit was tried %d times, want once", got)
	}
}

func TestAStopEndsTheCallAndNamesTheConnection(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.Fail(addressOne, stop(connection.FailureReasonInvalidRequest))

	_, err := s.complete(t)

	var rejected *connection.StoppedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "First" || rejected.Reason != "invalid request" {
		t.Fatalf("err = %v, want a rejection by First for an invalid request", err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the call went on to the next row after a stop")
	}
}

func TestAStopSetsNoCooldown(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.Fail(addressOne, stop(connection.FailureReasonInvalidRequest))
	s.attempt(t)

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	if got := len(s.world.Provider.CallsTo(addressOne)); got != 2 {
		t.Errorf("the first row was called %d times, want 2", got)
	}
}

func TestEveryReasonOfAFailureIsNamedForThePerson(t *testing.T) {
	for reason, want := range map[connection.FailureReason]string{
		connection.FailureReasonRejectedKey:     "key rejected",
		connection.FailureReasonPaymentRequired: "payment required",
		connection.FailureReasonRateLimited:     "rate limited",
		connection.FailureReasonUnavailable:     "unavailable",
		connection.FailureReasonInvalidRequest:  "invalid request",
	} {
		s := newSetup(t, modelcalltest.CompleteOperation)
		s.world.Provider.FailAlways(addressOne, tryNext(reason, 0))
		s.world.Provider.FailAlways(addressTwo, stop(reason))

		_, err := s.complete(t)

		var rejected *connection.StoppedError
		if !errors.As(err, &rejected) || rejected.Reason != want {
			t.Errorf("reason %v: err = %v, want a rejection for %q", reason, err, want)
		}
	}
}

func TestWhenEveryRowFailsTheErrorNamesEachConnectionAndReason(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))

	_, err := s.complete(t)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v, want *AllFailedError", err)
	}
	want := []connection.Failure{
		{ConnectionName: "First", Reason: "payment required", Code: connection.FailureReasonPaymentRequired},
		{ConnectionName: "Second", Reason: "rate limited", Code: connection.FailureReasonRateLimited},
	}
	if !slices.Equal(failed.Failures, want) {
		t.Errorf("failures = %v, want %v", failed.Failures, want)
	}
	if err.Error() != "First: payment required; Second: rate limited" {
		t.Errorf("text = %q", err.Error())
	}
}

func TestARowSetAsideEarlierStillAppearsInTheErrorWithItsReason(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))
	s.attempt(t)

	_, err := s.complete(t)

	if err == nil || err.Error() != "First: payment required; Second: rate limited" {
		t.Errorf("err = %v", err)
	}
	if got := len(s.world.Provider.CallsTo(addressOne)); got != 1 {
		t.Errorf("a connection set aside was called %d times, want once", got)
	}
}

func TestAPluginWithNoRowGetsNoConnection(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)

	_, err := s.callAs(t.Context(), "md.example.stranger", modelcalltest.CompleteOperation)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
	if len(s.world.Source.Opened()) != 0 {
		t.Error("a provider was started for a plugin with no row")
	}
}

func TestTheRowsOfAnotherOperationAreNotUsed(t *testing.T) {
	s := newSetup(t, modelcalltest.EmbedOperation)

	_, err := s.complete(t)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}

func TestAPluginReachesOnlyTheConnectionsOfItsOwnRows(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Rows.Set("md.example.first", modelcalltest.CompleteOperation, modelcalltest.Row(s.one, "m"))
	s.world.Rows.Set("md.example.second", modelcalltest.CompleteOperation, modelcalltest.Row(s.two, "m"))

	first, err := s.callAs(t.Context(), "md.example.first", modelcalltest.CompleteOperation)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.callAs(t.Context(), "md.example.second", modelcalltest.CompleteOperation)
	if err != nil {
		t.Fatal(err)
	}

	if first.Text != "answer from "+addressOne || second.Text != "answer from "+addressTwo {
		t.Errorf("first = %q, second = %q", first.Text, second.Text)
	}
	if _, err := s.callAs(t.Context(), "md.example.third", modelcalltest.CompleteOperation); !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("a plugin with no row reached a connection: %v", err)
	}
}

func TestTheKeysGoOnlyToThePluginThatProvidesTheType(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	other := s.world.Add(t, modelcalltest.OtherServerTypeID, "Other", "https://other.test/v1", "sk-other")
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(other, "m"), modelcalltest.Row(s.one, "m"))
	s.world.Provider.FailAlways("https://other.test/v1", tryNext(connection.FailureReasonUnavailable, 0))

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	want := []string{modelcalltest.OtherPluginID, modelcalltest.OtherPluginID, modelcalltest.ProviderPluginID}
	if got := s.world.Source.Opened(); !slices.Equal(got, want) {
		t.Errorf("providers opened = %v, want %v", got, want)
	}
}

func TestNothingTheConsumerReceivesContainsAKey(t *testing.T) {
	for name, arrange := range map[string]func(*setup){
		"an answer": func(*setup) {},
		"every row failed": func(s *setup) {
			s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonRejectedKey, 0))
			s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonPaymentRequired, 0))
		},
		"a rejection": func(s *setup) { s.world.Provider.FailAlways(addressOne, stop(connection.FailureReasonInvalidRequest)) },
		"a plain error of the provider": func(s *setup) {
			s.world.Provider.FailAlways(addressOne, errors.New("HTTP 401: Authorization: Bearer "+keyOne))
			s.world.Provider.FailAlways(addressTwo, errors.New("HTTP 401: Authorization: Bearer "+keyTwo))
		},
	} {
		s := newSetup(t, modelcalltest.CompleteOperation)
		arrange(s)

		result, err := s.complete(t)

		received := result.Text
		if err != nil {
			received += err.Error()
		}
		if strings.Contains(received, "sk-") {
			t.Errorf("%s: the consumer received a key: %q", name, received)
		}
	}
}

func TestAnErrorOfTheProviderThatIsNoFailureKindCountsAsUnavailable(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.FailAlways(addressOne, errors.New("rpc error: connection refused"))
	s.world.Provider.FailAlways(addressTwo, errors.New("rpc error: connection refused"))

	_, err := s.complete(t)

	if err == nil || err.Error() != "First: unavailable; Second: unavailable" {
		t.Errorf("err = %v", err)
	}
}

func TestAProviderThatCannotStartSetsTheRowAsideForTheCallOnly(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Source.FailOpen(modelcalltest.ProviderPluginID, errors.New("the program is missing"))

	_, err := s.complete(t)
	if err == nil || err.Error() != "First: unavailable; Second: unavailable" {
		t.Fatalf("err = %v", err)
	}

	s.world.Source.FailOpen(modelcalltest.ProviderPluginID, nil)
	if _, err := s.complete(t); err != nil {
		t.Errorf("the next call did not start the provider again: %v", err)
	}
}

func TestAProviderRestartedInTheMiddleOfACallIsStartedAgainForTheRetry(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.Fail(addressOne, errors.New("rpc error: connection reset"))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressOne {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	if got := s.world.Source.Opened(); len(got) != 2 {
		t.Errorf("providers opened = %v, want the retry to open it again", got)
	}
}

func TestEveryProviderOpenedIsReleased(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonUnavailable, 0))

	s.attempt(t)
	s.world.Provider.FailAlways(addressTwo, stop(connection.FailureReasonInvalidRequest))
	s.attempt(t)

	if opened, released := len(s.world.Source.Opened()), s.world.Source.Released(); opened != released {
		t.Errorf("opened %d providers and released %d", opened, released)
	}
}

func TestACallCanceledBeforeItStartsReachesNoProvider(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.callAs(ctx, consumerID, modelcalltest.CompleteOperation)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(s.world.Provider.Calls()) != 0 {
		t.Error("a provider was called for a canceled call")
	}
}

func TestACallCanceledWhileTheProviderWorksReturnsTheCancellationAndSetsNothingAside(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.Block(addressOne)
	ctx, cancel := context.WithCancel(t.Context())
	failed := make(chan error, 1)
	go func() {
		_, err := s.callAs(ctx, consumerID, modelcalltest.CompleteOperation)
		failed <- err
	}()
	<-s.world.Provider.Entered()

	cancel()

	if err := <-failed; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the call went on to the next row after it was canceled")
	}
	s.world.Provider.Unblock(addressOne)
	result, err := s.complete(t)
	if err != nil || result.Text != "answer from "+addressOne {
		t.Errorf("the canceled connection was set aside: %v, %v", result, err)
	}
	if got := s.world.Notices.List(); len(got) != 0 {
		t.Errorf("notices = %v after a canceled call", got)
	}
}

func TestARowWhoseKeyIsMissingIsSkippedAndTheNextRowServes(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	clear(s.world.Keys.Values)
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(s.one, "m"))

	_, err := s.complete(t)
	if err == nil || err.Error() != "First: key missing" {
		t.Errorf("err = %v", err)
	}
}

func TestARowWhoseTypeIsNotProvidedIsSkipped(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	other := s.world.Add(t, modelcalltest.OtherServerTypeID, "Other", "https://other.test/v1", "sk-other")
	s.world.SetAvailable(modelcalltest.ProviderPluginID, false)
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(s.one, "m"), modelcalltest.Row(other, "m"))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from https://other.test/v1" {
		t.Errorf("Complete = %v, %v", result, err)
	}
}

func TestARowWhoseTypeDoesNotHaveTheOperationIsSkippedWithTheReason(t *testing.T) {
	s := newSetup(t, modelcalltest.EmbedOperation)
	completer := s.world.Add(t, modelcalltest.CompleterTypeID, "Completer", "https://completer.test/v1", "")
	s.world.Rows.Set(consumerID, modelcalltest.EmbedOperation, modelcalltest.Row(completer, "m"))

	_, err := s.callAs(t.Context(), consumerID, modelcalltest.EmbedOperation)

	if err == nil || err.Error() != "Completer: the type does not serve llm.embed" {
		t.Errorf("err = %v", err)
	}
}

func TestARowThatNamesNoConnectionIsSkipped(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, connection.Row{ConnectionID: "c0000dead", Options: map[string]string{"model": "m"}}, modelcalltest.Row(s.two, "m"))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressTwo {
		t.Errorf("Complete = %v, %v", result, err)
	}
}

func TestAConnectionThatChangedWhileItsKeyWasReadIsReadAgain(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	changes := 0
	s.world.Keys.OnGet = func() {
		if changes == 0 {
			s.world.ChangeRevision(t, s.one.ID)
		}
		changes++
	}

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressOne {
		t.Errorf("Complete = %v, %v, want the answer of the first row", result, err)
	}
	if got := len(s.world.Provider.CallsTo(addressOne)); got != 1 {
		t.Errorf("the first row was called %d times, want once with its new snapshot", got)
	}
}

func TestAConnectionThatKeepsChangingWhileItsKeyIsReadIsSkipped(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Keys.OnGet = func() { s.world.ChangeRevision(t, s.one.ID) }

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressTwo {
		t.Errorf("Complete = %v, %v, want the answer of the second row", result, err)
	}
	if len(s.world.Provider.CallsTo(addressOne)) != 0 {
		t.Error("a provider was called with the snapshot of a connection that changed")
	}
}

func TestACallFailsWhenTheListOfRowsCannotBeRead(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Rows.Err = errors.New("the settings are unreadable")

	if _, err := s.complete(t); err == nil || errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want the failure to read the rows", err)
	}
}

func TestTheOperationAndThePayloadReachTheProviderWhole(t *testing.T) {
	s := newSetup(t, modelcalltest.EmbedOperation)
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))

	if _, err := s.callAs(t.Context(), consumerID, modelcalltest.EmbedOperation); err != nil {
		t.Fatal(err)
	}

	got := s.world.Provider.CallsTo(addressTwo)[0]
	if got.Operation != modelcalltest.EmbedOperation || string(got.Payload) != "fix this" {
		t.Errorf("call = %+v", got)
	}
}

func TestTheDataFramesAndTheFinalPayloadReachTheConsumerInOrder(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	s.world.Provider.Script(addressOne, nil, "The answer", " is short.")

	got, err := s.callAs(t.Context(), consumerID, modelcalltest.ChatOperation)

	if err != nil || !slices.Equal(got.Frames, []string{"The answer", " is short."}) || got.Text != "answer from "+addressOne {
		t.Errorf("answer = %+v, %v", got, err)
	}
}

func (s *setup) attempt(t *testing.T) {
	t.Helper()
	if _, err := s.complete(t); err != nil {
		t.Logf("complete: %v", err)
	}
}
