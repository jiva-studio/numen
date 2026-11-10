package textversion_test

import (
	"slices"
	"strings"
	"testing"
)

func (s *scene) reportFilesChanged(t *testing.T, paths ...string) []string {
	t.Helper()
	if err := s.versions.Changes.ReportFilesChanged(t.Context(), s.vault, paths); err != nil {
		t.Fatal(err)
	}
	return s.listener.take()
}

func TestReportFilesChanged_ReportsAFileWhoseContentEndsTheLiveVersion(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	s.listener.take()

	s.files[scanPath] = []byte("%PDF scan, second printing")

	if got := s.reportFilesChanged(t, scanPath, "notes/plain.md"); !slices.Equal(got, []string{scanPath}) {
		t.Errorf("reported %v, want %s alone", got, scanPath)
	}
}

func TestReportFilesChanged_ReportsAFileWhoseContentMakesAVersionLiveAgain(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	first := s.files[scanPath]
	s.files[scanPath] = []byte("%PDF scan, second printing")
	s.reportFilesChanged(t, scanPath)

	s.files[scanPath] = first

	if got := s.reportFilesChanged(t, scanPath); !slices.Equal(got, []string{scanPath}) {
		t.Errorf("reported %v, want %s", got, scanPath)
	}
}

func TestReportFilesChanged_ReportsNothingWhileTheLiveVersionStays(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	s.listener.take()

	if got := s.reportFilesChanged(t, scanPath); len(got) != 0 {
		t.Errorf("reported %v for a file whose live version stayed", got)
	}
}

func TestReportFilesChanged_ReportsANoteWhosePropertyEndsTheLiveVersion(t *testing.T) {
	s := newScene(t)
	s.files[talkPath] = []byte(talkNote)
	s.putTalkVersion(t, "article", "The article.")
	s.listener.take()

	s.files[talkPath] = []byte(strings.Replace(talkNote, "title: A talk", "title: A long talk", 1))
	if got := s.reportFilesChanged(t, talkPath); len(got) != 0 {
		t.Errorf("an edit of another property reported %v", got)
	}

	s.files[talkPath] = []byte(strings.Replace(talkNote, talkURL, "https://example.com/other", 1))
	if got := s.reportFilesChanged(t, talkPath); !slices.Equal(got, []string{talkPath}) {
		t.Errorf("a new url reported %v, want %s", got, talkPath)
	}
}

func TestReportFilesChanged_ReportsAFileThatLosesTheVersionMadeAfterItsLastCheck(t *testing.T) {
	s := newScene(t)
	s.reportFilesChanged(t, scanPath)
	s.requestText(t)
	s.listener.take()

	s.files[scanPath] = []byte("%PDF scan, second printing")

	if got := s.reportFilesChanged(t, scanPath); !slices.Equal(got, []string{scanPath}) {
		t.Errorf("reported %v, want %s", got, scanPath)
	}
}
