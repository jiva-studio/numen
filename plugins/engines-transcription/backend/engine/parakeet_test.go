package engine

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/resample"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/signal"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/sound"
)

// openTranscriber returns the models available on the machine and the recording
// to transcribe with them. The models are hundreds of megabytes and are not in
// the repository, so a machine without them skips the test and the rest of the
// package is still tested. NUMEN_MODEL names the model the files belong to,
// defaulting to the default model when unset.
func openTranscriber(t *testing.T) (*Transcriber, []byte) {
	t.Helper()
	dir, recorded := os.Getenv("NUMEN_MODELS"), os.Getenv("NUMEN_RECORDING")
	if dir == "" || recorded == "" {
		t.Skip("NUMEN_MODELS and NUMEN_RECORDING name no models and no recording")
	}

	cfg := Defaults()
	if name := os.Getenv("NUMEN_MODEL"); name != "" {
		model, ok := FindModel(name)
		if !ok {
			t.Fatalf("NUMEN_MODEL names %s, which the engine does not run", name)
		}
		cfg.Model = model
	}
	cfg.Dir, cfg.ShouldDownload = dir, false
	delta, err := Open(t.Context(), cfg)
	if err != nil {
		t.Skipf("the models in %s: %v", dir, err)
	}
	cleanup.CloseAfter(t, delta)

	raw, err := os.ReadFile(recorded)
	if err != nil {
		t.Fatal(err)
	}
	return delta, raw
}

// A recording is opened, cut into the stretches that carry speech, and each of
// them heard.
func TestHearingARecording(t *testing.T) {
	delta, raw := openTranscriber(t)
	recorded := os.Getenv("NUMEN_RECORDING")

	sound, err := delta.Open(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, sound)
	t.Logf("%s is %d ms", recorded, sound.GetDuration())

	found, err := sound.GetSegments(t.Context(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("the recording carries no speech")
	}

	var out []string
	for _, one := range found {
		text, err := delta.Transcribe(t.Context(), one)
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
	delta, raw := openTranscriber(t)

	samples, rate, err := sound.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	pos, err := resample.Convert(t.Context(), samples, rate, signal.SampleRate, 0)
	if err != nil {
		t.Fatal(err)
	}
	text, err := delta.Transcribe(t.Context(), Audio{Samples: pos})
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("the recording said nothing")
	}
	t.Logf("%s", text)
}
