package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/feed"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

func (s *toolFixture) newFollower() *feed.Follower {
	return newFollower(s.launcher, func(id domain.VaultID) feed.Vault {
		return feed.Vault{Feed: s.vault.With(id), Store: s.vault.Store().With(id)}
	}, nil)
}

func (s *toolFixture) runUpdate(t *testing.T) {
	t.Helper()
	if _, err := s.runHandlerWithin(t, command.Request{Command: UpdateCommand, Path: scanPath}); err != nil {
		t.Fatal(err)
	}
}

func TestTheUpdateCommandRecognizesTheScanOfTheFeedOnce(t *testing.T) {
	s := newToolFixture(t)

	s.runUpdate(t)
	s.waitUntilStarted(t)
	s.runUpdate(t)
	close(s.recognizer.release)
	s.waitUntilFinished(t, recognition.GetTaskID("Physics", scanPath))
	s.runUpdate(t)

	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the scan has no recognition once its task ended")
	}
	if count := s.openCount.Load(); count != 1 {
		t.Errorf("the recognizer was opened %d times, want once", count)
	}
}

func TestTheUpdateCommandSkipsAScanWithItsRecognition(t *testing.T) {
	s := newToolFixture(t)
	s.putRecognition(t)

	s.runUpdate(t)

	s.checkNoRun(t)
}
