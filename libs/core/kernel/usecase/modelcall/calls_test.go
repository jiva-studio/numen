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

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
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
		calls: modelcall.NewCalls(w.Resolve, w.ReadSecrets, w.Rows, w.Source, w.Clock, w.Waits.Wait, w.Notices),
	}
}

func (s *setup) complete(t *testing.T) (modelcalldomain.CompleteResult, error) {
	t.Helper()
	return s.calls.Complete(t.Context(), consumerID, modelcalldomain.CompleteRequest{
		Messages: []modelcalldomain.Message{{Role: "user", Text: "fix this"}},
	}, modelcalldomain.PriorityBackground)
}

func tryNext(reason connection.FailureReason, retryAfter time.Duration) error {
	return &connection.TryNextError{Reason: reason, RetryAfter: retryAfter}
}

func stop(reason connection.FailureReason) error {
	return &connection.StopError{Reason: reason}
}

func TestTheFirstRowThatAnswersServesTheCall(t *testing.T) {
	s := newSetup(t, "complete")

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressOne {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the second row was called though the first answered")
	}
}

func TestTheCallCarriesTheSnapshotOfTheRow(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Rows.Set(consumerID, "complete", connection.Row{ConnectionID: s.one.ID, Model: "model-one", Reasoning: "low"})

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	got := s.world.Provider.CallsTo(addressOne)[0].Call
	if got.Type != "server" || got.Model != "model-one" || got.Reasoning != "low" ||
		got.Fields["baseUrl"] != addressOne || got.Secrets["apiKey"] != keyOne {
		t.Errorf("call = %+v", got)
	}
}

func TestAFailureToTryTheNextGoesToTheNextRow(t *testing.T) {
	s := newSetup(t, "complete")
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
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, stop(connection.FailureReasonBadRequest))

	_, err := s.complete(t)

	var rejected *connection.RejectedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "First" || rejected.Reason != "bad request" {
		t.Fatalf("err = %v, want a rejection by First for a bad request", err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the call went on to the next row after a stop")
	}
}

func TestAStopSetsNoCooldown(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Fail(addressOne, stop(connection.FailureReasonBadRequest))
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
		connection.FailureReasonKeyRejected:     "key rejected",
		connection.FailureReasonPaymentRequired: "payment required",
		connection.FailureReasonRateLimited:     "rate limited",
		connection.FailureReasonUnreachable:     "unreachable",
		connection.FailureReasonTimeout:         "timeout",
		connection.FailureReasonBadRequest:      "bad request",
		connection.FailureReasonUnknown:         "unknown",
	} {
		s := newSetup(t, "complete")
		s.world.Provider.FailAlways(addressOne, tryNext(reason, 0))
		s.world.Provider.FailAlways(addressTwo, stop(reason))

		_, err := s.complete(t)

		var rejected *connection.RejectedError
		if !errors.As(err, &rejected) || rejected.Reason != want {
			t.Errorf("reason %v: err = %v, want a rejection for %q", reason, err, want)
		}
	}
}

func TestWhenEveryRowFailsTheErrorNamesEachConnectionAndReason(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))

	_, err := s.complete(t)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v, want *AllFailedError", err)
	}
	want := []connection.Failure{{ConnectionName: "First", Reason: "payment required"}, {ConnectionName: "Second", Reason: "rate limited"}}
	if !slices.Equal(failed.Failures, want) {
		t.Errorf("failures = %v, want %v", failed.Failures, want)
	}
	if err.Error() != "First: payment required; Second: rate limited" {
		t.Errorf("text = %q", err.Error())
	}
}

func TestARowSetAsideEarlierStillAppearsInTheErrorWithItsReason(t *testing.T) {
	s := newSetup(t, "complete")
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
	s := newSetup(t, "complete")

	_, err := s.calls.Complete(t.Context(), "md.example.stranger", modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
	if len(s.world.Source.Opened()) != 0 {
		t.Error("a provider was started for a plugin with no row")
	}
}

func TestTheRowsOfAnotherOperationAreNotUsed(t *testing.T) {
	s := newSetup(t, "embed")

	_, err := s.complete(t)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}

func TestAPluginReachesOnlyTheConnectionsOfItsOwnRows(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Rows.Set("md.example.first", "complete", modelcalltest.Row(s.one, "m"))
	s.world.Rows.Set("md.example.second", "complete", modelcalltest.Row(s.two, "m"))

	first, err := s.calls.Complete(t.Context(), "md.example.first", modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.calls.Complete(t.Context(), "md.example.second", modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}

	if first.Text != "answer from "+addressOne || second.Text != "answer from "+addressTwo {
		t.Errorf("first = %q, second = %q", first.Text, second.Text)
	}
	if _, err := s.calls.Complete(t.Context(), "md.example.third", modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground); !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("a plugin with no row reached a connection: %v", err)
	}
}

func TestTheKeysGoOnlyToThePluginThatProvidesTheType(t *testing.T) {
	s := newSetup(t, "complete")
	other := s.world.Add(t, modelcalltest.OtherServerTypeID, "Other", "https://other.test/v1", "sk-other")
	s.world.Rows.Set(consumerID, "complete", modelcalltest.Row(other, "m"), modelcalltest.Row(s.one, "m"))
	s.world.Provider.FailAlways("https://other.test/v1", tryNext(connection.FailureReasonUnreachable, 0))

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
			s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonKeyRejected, 0))
			s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonPaymentRequired, 0))
		},
		"a rejection": func(s *setup) { s.world.Provider.FailAlways(addressOne, stop(connection.FailureReasonBadRequest)) },
		"a plain error of the provider": func(s *setup) {
			s.world.Provider.FailAlways(addressOne, errors.New("HTTP 401: Authorization: Bearer "+keyOne))
			s.world.Provider.FailAlways(addressTwo, errors.New("HTTP 401: Authorization: Bearer "+keyTwo))
		},
	} {
		s := newSetup(t, "complete")
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

func TestAnErrorOfTheProviderThatIsNoFailureKindCountsAsUnreachable(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, errors.New("rpc error: connection refused"))
	s.world.Provider.FailAlways(addressTwo, errors.New("rpc error: connection refused"))

	_, err := s.complete(t)

	if err == nil || err.Error() != "First: unreachable; Second: unreachable" {
		t.Errorf("err = %v", err)
	}
}

func TestAProviderThatCannotStartSetsTheRowAsideForTheCallOnly(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Source.FailOpen(modelcalltest.ProviderPluginID, errors.New("the program is missing"))

	_, err := s.complete(t)
	if err == nil || err.Error() != "First: unreachable; Second: unreachable" {
		t.Fatalf("err = %v", err)
	}

	s.world.Source.FailOpen(modelcalltest.ProviderPluginID, nil)
	if _, err := s.complete(t); err != nil {
		t.Errorf("the next call did not start the provider again: %v", err)
	}
}

func TestAProviderRestartedInTheMiddleOfACallIsStartedAgainForTheRetry(t *testing.T) {
	s := newSetup(t, "complete")
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
	s := newSetup(t, "complete")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonTimeout, 0))

	s.attempt(t)
	s.world.Provider.FailAlways(addressTwo, stop(connection.FailureReasonBadRequest))
	s.attempt(t)

	if opened, released := len(s.world.Source.Opened()), s.world.Source.Released(); opened != released {
		t.Errorf("opened %d providers and released %d", opened, released)
	}
}

func TestACallCanceledBeforeItStartsReachesNoProvider(t *testing.T) {
	s := newSetup(t, "complete")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.calls.Complete(ctx, consumerID, modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(s.world.Provider.Calls()) != 0 {
		t.Error("a provider was called for a canceled call")
	}
}

func TestACallCanceledWhileTheProviderWorksReturnsTheCancellationAndSetsNothingAside(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Provider.Block(addressOne)
	ctx, cancel := context.WithCancel(t.Context())
	failed := make(chan error, 1)
	go func() {
		_, err := s.calls.Complete(ctx, consumerID, modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)
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
	s := newSetup(t, "complete")
	clear(s.world.Keys.Values)
	s.world.Rows.Set(consumerID, "complete", modelcalltest.Row(s.one, "m"))

	_, err := s.complete(t)
	if err == nil || err.Error() != "First: key missing" {
		t.Errorf("err = %v", err)
	}
}

func TestARowWhoseTypeIsNotProvidedIsSkipped(t *testing.T) {
	s := newSetup(t, "complete")
	other := s.world.Add(t, modelcalltest.OtherServerTypeID, "Other", "https://other.test/v1", "sk-other")
	s.world.SetAvailable(modelcalltest.ProviderPluginID, false)
	s.world.Rows.Set(consumerID, "complete", modelcalltest.Row(s.one, "m"), modelcalltest.Row(other, "m"))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from https://other.test/v1" {
		t.Errorf("Complete = %v, %v", result, err)
	}
}

func TestARowWhoseTypeDoesNotHaveTheOperationIsSkippedWithTheReason(t *testing.T) {
	s := newSetup(t, "embed")
	completer := s.world.Add(t, modelcalltest.CompleterTypeID, "Completer", "https://completer.test/v1", "")
	s.world.Rows.Set(consumerID, "embed", modelcalltest.Row(completer, "m"))

	_, err := s.calls.Embed(t.Context(), consumerID, []string{"a"}, modelcalldomain.PriorityBackground)

	if err == nil || err.Error() != "Completer: this type cannot embed" {
		t.Errorf("err = %v", err)
	}
}

func TestARowThatNamesNoConnectionIsSkipped(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Rows.Set(consumerID, "complete", connection.Row{ConnectionID: "c0000dead", Model: "m"}, modelcalltest.Row(s.two, "m"))

	result, err := s.complete(t)

	if err != nil || result.Text != "answer from "+addressTwo {
		t.Errorf("Complete = %v, %v", result, err)
	}
}

func TestACallIsRefusedWhenTheConnectionChangedWhileItsKeyWasRead(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Keys.OnGet = func() { s.world.ChangeRevision(t, s.one.ID) }

	_, err := s.complete(t)

	if !errors.Is(err, connection.ErrStaleRevision) {
		t.Errorf("err = %v, want ErrStaleRevision", err)
	}
	if len(s.world.Provider.Calls()) != 0 {
		t.Error("a provider was called with the snapshot of a connection that changed")
	}
}

func TestACallFailsWhenTheListOfRowsCannotBeRead(t *testing.T) {
	s := newSetup(t, "complete")
	s.world.Rows.Err = errors.New("the settings are unreadable")

	if _, err := s.complete(t); err == nil || errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want the failure to read the rows", err)
	}
}

func TestEmbedReturnsTheVectorsOfTheFirstRowThatAnswers(t *testing.T) {
	s := newSetup(t, "embed")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))

	vectors, err := s.calls.Embed(t.Context(), consumerID, []string{"ab", "abcd"}, modelcalldomain.PriorityBackground)

	if err != nil || len(vectors) != 2 || vectors[1][0] != 4 {
		t.Fatalf("Embed = %v, %v", vectors, err)
	}
	if got := s.world.Provider.CallsTo(addressTwo)[0].Operation; got != "embed" {
		t.Errorf("operation = %q", got)
	}
}

func TestEmbedOfNoTextCallsNoProvider(t *testing.T) {
	s := newSetup(t, "embed")

	vectors, err := s.calls.Embed(t.Context(), consumerID, nil, modelcalldomain.PriorityBackground)

	if err != nil || len(vectors) != 0 || len(s.world.Provider.Calls()) != 0 {
		t.Errorf("Embed = %v, %v, calls = %v", vectors, err, s.world.Provider.Calls())
	}
}

func TestGetEmbeddingProfileReturnsTheProfileWithTheModelOfTheRowThatAnswered(t *testing.T) {
	s := newSetup(t, "embed")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))

	profile, err := s.calls.GetEmbeddingProfile(t.Context(), consumerID)

	want := modelcalldomain.EmbeddingProfile{
		Model: "model-two", Source: "server " + addressTwo, Pooling: "mean", QueryPrefix: "query: ", DocumentPrefix: "passage: ",
		Dimensions: 3, MaxTokens: 8191,
	}
	if err != nil || profile != want {
		t.Errorf("profile = %+v, %v, want %+v", profile, err, want)
	}
}

func TestGetEmbeddingProfileProbesTheWidthOfAModelTheProviderDoesNotKnow(t *testing.T) {
	s := newSetup(t, "embed")
	s.world.Provider.IsUnknown = true

	profile, err := s.calls.GetEmbeddingProfile(t.Context(), consumerID)

	if err != nil || profile.Dimensions != 7 || profile.Model != "model-one" {
		t.Errorf("profile = %+v, %v, want the probed width 7", profile, err)
	}
	operations := []string{}
	for _, one := range s.world.Provider.Calls() {
		operations = append(operations, one.Operation)
	}
	if !slices.Equal(operations, []string{"get model profile", "probe embedding width"}) {
		t.Errorf("operations = %v", operations)
	}
}

func TestACompleteRequestReachesTheProviderWhole(t *testing.T) {
	s := newSetup(t, "complete")
	temperature := 0.3

	_, err := s.calls.Complete(t.Context(), consumerID, modelcalldomain.CompleteRequest{
		Messages: []modelcalldomain.Message{{Role: "system", Text: "s"}, {Role: "user", Text: "u"}}, Temperature: &temperature, MaxTokens: 50,
	}, modelcalldomain.PriorityBackground)

	if err != nil {
		t.Fatal(err)
	}
	got := s.world.Provider.Calls()[0]
	if got.Operation != "complete" || got.Request.MaxTokens != 50 || got.Request.Temperature == nil || *got.Request.Temperature != 0.3 ||
		len(got.Request.Messages) != 2 || got.Request.Messages[1].Text != "u" {
		t.Errorf("call = %+v", got)
	}
}

func (s *setup) attempt(t *testing.T) {
	t.Helper()
	if _, err := s.complete(t); err != nil {
		t.Logf("complete: %v", err)
	}
}
