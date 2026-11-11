package transcript_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/transcript"
)

func TestReachedSurvivesEncoding(t *testing.T) {
	raw, err := transcript.Reached{MS: 61000, Size: 2048}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got, err := transcript.ParseReached(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.MS != 61000 || got.Size != 2048 {
		t.Fatalf("got %+v, want 61000 ms and 2048 bytes", got)
	}
}

func TestParseReachedRefusesInvalidJSON(t *testing.T) {
	if _, err := transcript.ParseReached([]byte("{broken")); err == nil {
		t.Fatal("ParseReached accepted invalid JSON")
	}
}
