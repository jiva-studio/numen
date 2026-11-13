package sourcebridge_test

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

func failureReports(bridge *sourcebridge.Source) chan error {
	reports := make(chan error, 8)
	bridge.OnDescriptionReadError(func(err error) { reports <- err })
	return reports
}

// A read of the model that fails after a setting of the plugin is reported, with the plugin it concerns.
func TestAFailedReadOfTheModelAfterASettingIsReported(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.describeCalls.Load()

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
	reads := mock.describeCalls.Load()

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
	reads := mock.describeCalls.Load()

	mock.modelName.Store("another-model")
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := bridge.Describe().Name; got != "test-model" {
		t.Errorf("the model is %q after a failed read, want test-model", got)
	}
}

// The change that follows a failed read is announced once, with the model that was read last.
func TestTheNextChangeIsAnnouncedOnceAfterAFailedRead(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	failureReports(bridge)
	reads := mock.describeCalls.Load()

	mock.modelName.Store("another-model")
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	mock.failRead.Store(false)
	reads = mock.describeCalls.Load()
	if err := updateSetting(t.Context(), host, embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the model change was announced %d times after the failed read, want once", got)
	}
	if got := bridge.Describe().Name; got != "another-model" {
		t.Errorf("the model is %q, want another-model", got)
	}
}

// A read that succeeds is reported once, with no error.
func TestASuccessfulReadOfTheModelReportsNoError(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.describeCalls.Load()

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
	reads := mock.describeCalls.Load()
	mock.failRead.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)
	<-reports

	reads = mock.describeCalls.Load()
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

// A plugin whose settings do not allow the read yet, such as a connection source with no row chosen, is reported with the
// sentence the plugin gives, so the person reads what to do and not a code.
func TestAReadThatTheSettingsDoNotAllowIsReportedWithThePlainSentence(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reports := failureReports(bridge)
	reads := mock.describeCalls.Load()

	mock.failPrecondition.Store(true)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	select {
	case err := <-reports:
		if err == nil || err.Error() != noConnection {
			t.Errorf("a read the settings do not allow was reported as %v, want %q", err, noConnection)
		}
	case <-time.After(time.Second):
		t.Fatal("the read was not reported")
	}
}

// A source opened for a window while its settings do not allow it is opened with no description, and the
// sentence of the plugin is reported.
func TestASourceTheSettingsDoNotAllowOpensWatchingWithNoDescription(t *testing.T) {
	mock := &mockSourceServer{}
	mock.failPrecondition.Store(true)
	reports := make(chan error, 8)

	roles := pluginhost.Join(startConfigurableSource(t, mock))
	bridge, err := sourcebridge.OpenWatching(t.Context(), roles, roles, nil, func(err error) { reports <- err })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, bridge) })

	if err := <-reports; err == nil || err.Error() != noConnection {
		t.Errorf("the first read was reported as %v, want %q", err, noConnection)
	}
	if described := bridge.Describe(); described.IsDescribed() {
		t.Errorf("a source the settings do not allow is described as %+v", described)
	}
}

// The setting that lets a source opened with no description describe itself announces the change once, and
// clears the reported failure.
func TestASettingThatAllowsTheSourceAnnouncesItsDescription(t *testing.T) {
	mock := &mockSourceServer{}
	mock.failPrecondition.Store(true)
	host := startConfigurableSource(t, mock)
	reports := make(chan error, 8)
	roles := pluginhost.Join(host)
	bridge, err := sourcebridge.OpenWatching(t.Context(), roles, roles, nil, func(err error) { reports <- err })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, bridge) })
	<-reports
	var announced atomic.Int32
	bridge.OnDescriptionChange(func() { announced.Add(1) })

	reads := mock.describeCalls.Load()
	mock.failPrecondition.Store(false)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "connection", "garden-row"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the description was announced %d times, want once", got)
	}
	if got := bridge.Describe(); !got.IsDescribed() || got.Name != "test-model" {
		t.Errorf("the source is described as %+v, want test-model", got)
	}
	select {
	case err := <-reports:
		if err != nil {
			t.Errorf("the read that succeeded was reported with %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("the read that succeeded was not reported")
	}
}
