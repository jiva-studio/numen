package pending_test

import (
	"maps"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/pending"
)

func TestParseReadsWhatEncodeWrote(t *testing.T) {
	files := map[int64]string{7: "books/ohm.pdf", 9: "lectures/ohm.mp3"}
	content, err := pending.Encode(files)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := pending.Parse(content)

	if !ok || !maps.Equal(got, files) {
		t.Errorf("Parse = %v, %v, want %v", got, ok, files)
	}
}

func TestParseRefusesContentOfAnotherVersion(t *testing.T) {
	for _, content := range []string{`{"version":2,"files":{}}`, `not json`} {
		if _, ok := pending.Parse([]byte(content)); ok {
			t.Errorf("Parse(%s) was accepted", content)
		}
	}
}
