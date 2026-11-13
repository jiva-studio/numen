package providerbridge_test

import (
	"errors"
	"testing"
	"testing/synctest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
)

func countDetectionsAfterCall(t *testing.T, connectionID string, failure error) int {
	t.Helper()
	f := newDetectionFixture()
	f.source.mu.Lock()
	f.source.failCall = failure
	f.source.mu.Unlock()
	provider, release, err := providerbridge.NewRedetectingSource(f.source, f.detection).OpenProvider(t.Context(), agentsPluginID)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	_, callErr := provider.Call(t.Context(), "acp.task", modelcall.ProviderCall{ConnectionID: connectionID}, nil, func([]byte) error { return nil })
	synctest.Wait()

	if !errors.Is(callErr, failure) {
		t.Errorf("err = %v, want %v", callErr, failure)
	}
	return f.countCalls()
}

func TestRedetectingSourceDetectsAgainWhenADetectedConnectionIsUnavailable(t *testing.T) {
	for name, failure := range map[string]error{
		"an unavailable service": &connection.TryNextError{Reason: connection.FailureReasonUnavailable},
		"an error of no kind":    errors.New("rpc error: connection reset"),
	} {
		synctest.Test(t, func(t *testing.T) {
			if got := countDetectionsAfterCall(t, kiteID, failure); got != 1 {
				t.Errorf("%s: connections.detected was asked %d times, want once", name, got)
			}
		})
	}
}

func TestRedetectingSourceDoesNotDetectAgain(t *testing.T) {
	for name, tc := range map[string]struct {
		connectionID string
		failure      error
	}{
		"for a rejected key": {kiteID, &connection.TryNextError{Reason: connection.FailureReasonRejectedKey}},
		"for a stop":         {kiteID, &connection.StopError{Reason: connection.FailureReasonInvalidRequest}},
		"for a stored connection": {
			"c00000001", &connection.TryNextError{Reason: connection.FailureReasonUnavailable},
		},
	} {
		synctest.Test(t, func(t *testing.T) {
			if got := countDetectionsAfterCall(t, tc.connectionID, tc.failure); got != 0 {
				t.Errorf("%s: connections.detected was asked %d times, want none", name, got)
			}
		})
	}
}
