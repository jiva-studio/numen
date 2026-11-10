package textversion_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func (s *scene) listTargetVersions(t *testing.T) []plugin.TargetVersion {
	t.Helper()
	got, err := s.versions.Texts.ListTargetVersions(t.Context(), s.vault, scanPath)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestTargetVersionsMarkTheSourceOfTheLiveVersionValid(t *testing.T) {
	s := newScene(t)
	s.runEntry(t, ocr)
	s.runEntry(t, proofreader)

	got := s.listTargetVersions(t)

	want := []plugin.TargetVersion{
		{Kind: string(kindRecognition), PluginID: ocr, IsLive: false, IsValid: true},
		{Kind: string(kindRecognition), PluginID: proofreader, IsLive: true, IsValid: true},
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("versions %+v, want %+v", got, want)
	}
}

func TestAVersionOfAFileThatChangedIsNotValid(t *testing.T) {
	s := newScene(t)
	s.runEntry(t, ocr)
	s.files[scanPath] = []byte("%PDF scan, second printing")

	got := s.listTargetVersions(t)

	if len(got) != 1 || got[0].IsValid || got[0].IsLive {
		t.Errorf("versions %+v, want the recognition neither valid nor live", got)
	}
}
