package filesystem

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestAFileIsTheKindItsNameMakesIt(t *testing.T) {
	cases := []struct {
		name string
		want domain.SourceKind
	}{
		{"notes/today.md", domain.KindNote},
		{"library/A Book.epub", domain.KindBook},
		{"library/A Scan.pdf", domain.KindBook},
		{"talks/A Lecture.mp3", domain.KindRecording},
		{"talks/A Lecture.WAV", domain.KindRecording},
		{"talks/A Lecture.flac", domain.KindRecording},
	}
	for _, c := range cases {
		kind, ok := Options{}.kind(c.name)
		if !ok || kind != c.want {
			t.Errorf("%s is %q, want %q", c.name, kind, c.want)
		}
	}
	if _, ok := (Options{}).kind("talks/A Lecture.opus"); ok {
		t.Error("a name on none of the lists is a source")
	}
}

func TestANameOnTwoListsIsANote(t *testing.T) {
	opts := Options{RecordingExtensions: []string{".md"}}
	if kind, ok := opts.kind("talks/spoken.md"); !ok || kind != domain.KindNote {
		t.Errorf("a name both lists claim is %q", kind)
	}
}

func TestTheRecordingListIsWhatAVaultAsksFor(t *testing.T) {
	opts := Options{RecordingExtensions: []string{".ogg"}}
	if kind, ok := opts.kind("talks/A Lecture.ogg"); !ok || kind != domain.KindRecording {
		t.Errorf("what the vault asked for is %q", kind)
	}
	if _, ok := opts.kind("talks/A Lecture.mp3"); ok {
		t.Error("the defaults answer beside what the vault asked for")
	}
}
