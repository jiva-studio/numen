package transcription

import (
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// openTranscriber is the models a machine holds, and the recording to hear with them.
// They are hundreds of megabytes and are not in the repository, so a machine
// without them says so and the rest of the package is still tested.
func openTranscriber(t *testing.T) (*Transcriber, []byte) {
	t.Helper()
	dir, recorded := os.Getenv("NUMEN_MODELS"), os.Getenv("NUMEN_RECORDING")
	if dir == "" || recorded == "" {
		t.Skip("NUMEN_MODELS and NUMEN_RECORDING name no models and no recording")
	}

	cfg := Defaults()
	cfg.Dir, cfg.Download = dir, false
	by, err := Open(t.Context(), cfg)
	if err != nil {
		t.Skipf("the models in %s: %v", dir, err)
	}
	t.Cleanup(func() { by.Close() })

	raw, err := os.ReadFile(recorded)
	if err != nil {
		t.Fatal(err)
	}
	return by, raw
}

// A recording is opened, cut into the stretches that carry speech, and each of
// them heard.
func TestHearingARecording(t *testing.T) {
	by, raw := openTranscriber(t)
	recorded := os.Getenv("NUMEN_RECORDING")

	sound, err := by.Open(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	defer sound.Close()
	t.Logf("%s is %d ms", recorded, sound.Length())

	found, err := sound.Segments(t.Context(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("the recording carries no speech")
	}

	var out []string
	for _, one := range found {
		text, err := by.Transcribe(t.Context(), one)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%d-%d ms: %s", one.From, one.To, text)
		if text != "" {
			out = append(out, text)
		}
	}
	if len(out) == 0 {
		t.Fatal("the recording said nothing")
	}
	t.Logf("%s", strings.Join(out, " "))
}

// A whole recording heard as one stretch is what the reference implementation
// is given, and is how the words this writes are compared with the words it
// writes.
func TestHearingAWholeRecording(t *testing.T) {
	by, raw := openTranscriber(t)

	sound, rate, err := samples(raw)
	if err != nil {
		t.Fatal(err)
	}
	at, err := resample(t.Context(), sound, rate, sampleRate)
	if err != nil {
		t.Fatal(err)
	}
	text, err := by.Transcribe(t.Context(), port.Audio{Samples: at})
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("the recording said nothing")
	}
	t.Logf("%s", text)
}
