package service

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/feed"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

func (s *toolFixture) newAutomatic(t *testing.T) *automatic {
	t.Helper()
	follower := newFollower(s.launcher, func(id domain.VaultID) feed.Vault {
		return feed.Vault{Feed: s.vault.With(id), Store: s.vault.Store().With(id)}
	}, nil)
	auto := newAutomatic(follower, s.launcher.GetShownVaultID, func(what string, err error) { t.Errorf("%s: %v", what, err) })
	t.Cleanup(auto.close)
	return auto
}

func (s *toolFixture) deliver(t *testing.T, auto *automatic, event string) {
	t.Helper()
	if err := auto.handleEvent(t.Context(), event); err != nil {
		t.Fatal(err)
	}
}

func TestAFilesChangedEventRecognizesTheScanOfTheFeedOnceInTheBackground(t *testing.T) {
	s := newToolFixture(t)
	auto := s.newAutomatic(t)
	auto.isOn.Store(true)

	s.deliver(t, auto, "onFilesChanged:Physics")
	s.waitUntilStarted(t)
	s.deliver(t, auto, "onFilesChanged:Physics")
	close(s.recognizer.release)
	s.waitUntilFinished(t, recognition.GetTaskID("Physics", scanPath))
	s.deliver(t, auto, "onFilesChanged:Physics")

	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the scan has no recognition once its task ended")
	}
	if count := s.openCount.Load(); count != 1 {
		t.Errorf("the recognizer was opened %d times, want once", count)
	}
	if got := s.queue.calls.listUserStarted(); len(got) == 0 || slices.Contains(got, true) {
		t.Errorf("the run asked the queue as started by a person: %v", got)
	}
}

func TestAFilesChangedEventRecognizesNothingWhileTheSettingIsOff(t *testing.T) {
	s := newToolFixture(t)
	auto := s.newAutomatic(t)

	s.deliver(t, auto, "onFilesChanged:Physics")
	s.deliver(t, auto, "onIndexChanged:Physics")

	s.checkNoRun(t)
}

func TestTurningTheSettingOnRecognizesTheScansOfTheShownVault(t *testing.T) {
	s := newToolFixture(t)
	auto := s.newAutomatic(t)

	auto.updateSettings(map[string]string{settingAutomatic: "true"})
	s.waitUntilStarted(t)
	close(s.recognizer.release)
	s.waitUntilFinished(t, recognition.GetTaskID("Physics", scanPath))

	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the scan has no recognition once its task ended")
	}
}

func TestTheUpdateWithARecognitionSkipsTheScan(t *testing.T) {
	s := newToolFixture(t)
	s.putRecognition(t)
	auto := s.newAutomatic(t)
	auto.isOn.Store(true)

	s.deliver(t, auto, "onFilesChanged:Physics")

	s.checkNoRun(t)
}
