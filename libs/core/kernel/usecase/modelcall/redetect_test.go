package modelcall_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

const addressKite = "https://kite.test/v1"

type redetectSetup struct {
	*setup
	kite       modelcalltest.Entry
	redetected []string
}

// newRedetectSetup holds the detected connection Kite in the first row and the stored connection First in
// the second, and keeps the plugins asked to detect again.
func newRedetectSetup(t *testing.T) *redetectSetup {
	t.Helper()
	s := &redetectSetup{setup: newSetup(t, modelcalltest.CompleteOperation)}
	w := s.world
	s.kite = w.AddDetected("Kite", addressKite)
	w.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(s.kite, "m"), modelcalltest.Row(s.one, "m"))
	s.calls = modelcall.NewCalls(modelcall.Connections{
		Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows.ListRows,
		Redetect: func(pluginID string) { s.redetected = append(s.redetected, pluginID) },
	}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside)
	return s
}

func TestAnUnavailableDetectedConnectionMakesItsProviderDetectAgain(t *testing.T) {
	s := newRedetectSetup(t)
	s.world.Provider.FailAlways(addressKite, tryNext(connection.FailureReasonUnavailable, 0))

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(s.redetected, []string{modelcalltest.ProviderPluginID}) {
		t.Errorf("redetected = %v, want the provider of Kite once", s.redetected)
	}
}

func TestADetectedConnectionWhosePluginDoesNotStartMakesItsProviderDetectAgain(t *testing.T) {
	s := newRedetectSetup(t)
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(s.kite, "m"))
	s.world.Source.FailOpen(modelcalltest.ProviderPluginID, errors.New("missing program"))

	if _, err := s.complete(t); err == nil {
		t.Fatal("the call through a plugin that does not start succeeded")
	}

	if !slices.Equal(s.redetected, []string{modelcalltest.ProviderPluginID}) {
		t.Errorf("redetected = %v, want the provider of Kite once", s.redetected)
	}
}

func TestADetectedConnectionThatFailsForAnotherReasonIsNotDetectedAgain(t *testing.T) {
	s := newRedetectSetup(t)
	s.world.Provider.FailAlways(addressKite, tryNext(connection.FailureReasonRejectedKey, 0))

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	if len(s.redetected) != 0 {
		t.Errorf("redetected = %v, want none", s.redetected)
	}
}

func TestAnUnavailableStoredConnectionIsNotDetectedAgain(t *testing.T) {
	s := newRedetectSetup(t)
	s.world.Rows.Set(consumerID, modelcalltest.CompleteOperation, modelcalltest.Row(s.one, "m"), modelcalltest.Row(s.kite, "m"))
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonUnavailable, 0))

	if _, err := s.complete(t); err != nil {
		t.Fatal(err)
	}

	if len(s.redetected) != 0 {
		t.Errorf("redetected = %v, want none", s.redetected)
	}
}
