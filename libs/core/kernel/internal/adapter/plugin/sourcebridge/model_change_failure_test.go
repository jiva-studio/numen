package sourcebridge_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"strings"
	"testing"
	"time"
)

func failureReports(bridge *embedderbridge.Embedder) chan error {
	reports := make(chan error, 8)
	bridge.OnModelReadError(func(err error) { reports <- err })
	return reports
}

// A read of the model that fails after a setting of the plugin is reported, with the plugin it concerns.
func TestAFailedReadOfTheModelAfterASettingIsReported(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.getModelCalls.Load()

	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	select {
	case err := <-reports:
		if !strings.Contains(err.Error(), embeddingPluginID) {
			t.Errorf("the error %q does not name the plugin %s", err, embeddingPluginID)
		}
	case <-time.After(time.Second):
		t.Fatal("the failed read of the model was not reported")
	}
	if len(reports) != 0 {
		t.Errorf("%d more reports for one failed read", len(reports))
	}
}

// A model that could not be read is not announced: the listeners run only for a model that was read.
func TestNoModelChangeIsAnnouncedWhenTheReadOfTheModelFails(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	failureReports(bridge)
	reads := mock.getModelCalls.Load()

	mock.modelName.Store("another-model")
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 0 {
		t.Errorf("a model that was not read was announced %d times", got)
	}
}

// The model the bridge reports is the one it read before, when the next read fails.
func TestTheModelIsTheSameAfterAFailedRead(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	failureReports(bridge)
	reads := mock.getModelCalls.Load()

	mock.modelName.Store("another-model")
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := bridge.GetModel().Name; got != "test-model" {
		t.Errorf("the model is %q after a failed read, want test-model", got)
	}
}

// The change that follows a failed read is announced once, with the model that was read last.
func TestTheNextChangeIsAnnouncedOnceAfterAFailedRead(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	failureReports(bridge)
	reads := mock.getModelCalls.Load()

	mock.modelName.Store("another-model")
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	mock.failRead.Store(false)
	reads = mock.getModelCalls.Load()
	if err := updateSetting(t.Context(), host, embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the model change was announced %d times after the failed read, want once", got)
	}
	if got := bridge.GetModel().Name; got != "another-model" {
		t.Errorf("the model is %q, want another-model", got)
	}
}

// A read that succeeds is reported once, with no error.
func TestASuccessfulReadOfTheModelReportsNoError(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.getModelCalls.Load()

	if err := updateSetting(t.Context(), host, embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if len(reports) != 1 {
		t.Fatalf("%d reports for one successful read, want 1", len(reports))
	}
	if err := <-reports; err != nil {
		t.Errorf("a successful read reported %v", err)
	}
}

// A read that succeeds after a failed one is reported with no error, so what showed the failure can clear it.
func TestASuccessfulReadAfterAFailedReadIsReportedWithNoError(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.getModelCalls.Load()
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)
	<-reports

	reads = mock.getModelCalls.Load()
	mock.failRead.Store(false)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 256); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	select {
	case err := <-reports:
		if err != nil {
			t.Errorf("the successful read was reported with the error %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("the successful read was not reported")
	}
}

// A plugin whose settings do not allow the read yet, such as a connection source with no row chosen, is not failing:
// the read is reported with no error, so no failure is shown.
func TestAReadThatTheSettingsDoNotAllowYetIsNotReportedAsAFailure(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.getModelCalls.Load()

	mock.failPrecondition.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	select {
	case err := <-reports:
		if err != nil {
			t.Errorf("a read the settings do not allow was reported as %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("the read was not reported")
	}
}
