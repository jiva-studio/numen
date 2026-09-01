package transcription

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// answer is one frame's scores: the token the joiner names, and how many frames
// it covers.
func answer(blank, token, step, steps int) []float32 {
	out := make([]float32, blank+1+steps)
	out[token] = 1
	out[blank+1+step] = 1
	return out
}

// The decoding says the tokens the joiner names and steps on by the number
// beside them. A blank is a frame carrying nothing and is not said.
func TestDecodeSaysWhatTheJoinerNames(t *testing.T) {
	const blank, steps = 3, 3
	named := map[int][]float32{
		0: answer(blank, 1, 2, steps),
		2: answer(blank, blank, 1, steps),
		3: answer(blank, 2, 1, steps),
	}

	var asked []int
	said, err := transducer{
		frames:  4,
		blank:   blank,
		encoded: func(at int) []float32 { return []float32{float32(at)} },
		predict: func(token int) ([]float32, error) {
			asked = append(asked, token)
			return []float32{float32(token)}, nil
		},
		joint: func(frame, _ []float32) ([]float32, error) {
			return named[int(frame[0])], nil
		},
	}.decode(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(said) != 2 || said[0] != 1 || said[1] != 2 {
		t.Errorf("the frames said %v", said)
	}
	// The predictor opens on the blank and is moved on by each token said.
	if len(asked) != 3 || asked[0] != blank || asked[1] != 1 || asked[2] != 2 {
		t.Errorf("the predictor was given %v", asked)
	}
}

// A frame the joiner covers with nothing is still left behind, and a recording
// of them ends.
func TestDecodeLeavesAFrameTheJoinerCoversWithNothing(t *testing.T) {
	const blank, steps = 3, 3
	said, err := transducer{
		frames:  4,
		blank:   blank,
		encoded: func(int) []float32 { return nil },
		predict: func(int) ([]float32, error) { return nil, nil },
		joint: func([]float32, []float32) ([]float32, error) {
			return answer(blank, blank, 0, steps), nil
		},
	}.decode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 0 {
		t.Errorf("frames carrying nothing said %v", said)
	}
}

// A joiner answering with fewer scores than the blank stands at is not the
// joiner these tokens came from, and is refused.
func TestDecodeRefusesAJoinerThatIsTooNarrow(t *testing.T) {
	_, err := transducer{
		frames:  1,
		blank:   8192,
		encoded: func(int) []float32 { return nil },
		predict: func(int) ([]float32, error) { return nil, nil },
		joint: func([]float32, []float32) ([]float32, error) {
			return make([]float32, 16), nil
		},
	}.decode(context.Background())
	if err == nil {
		t.Error("a joiner too narrow for the tokens was read")
	}
}

// The pieces are joined as they stand, and the mark a word opens with is the
// space before it.
func TestPiecesJoinIntoWords(t *testing.T) {
	p := pieces{"<unk>", "▁what", "▁the", "▁page", " say", "s", "<blk>"}
	if got := p.text([]int{1, 2, 3, 4, 5}); got != "what the page says" {
		t.Errorf("the tokens say %q", got)
	}
	if got := p.text(nil); got != "" {
		t.Errorf("no tokens say %q", got)
	}
	// A number outside the file is a token this transcriber cannot write.
	if got := p.text([]int{1, 900}); got != "what" {
		t.Errorf("an unknown token said %q", got)
	}
	if p.at("<blk>") != 6 || p.at("nothing") != -1 {
		t.Errorf("the blank stands at %d", p.at("<blk>"))
	}
}

func TestTokensReadsTheFilePublishedBesideAModel(t *testing.T) {
	at := t.TempDir() + "/tokens.txt"
	if err := os.WriteFile(at, []byte("<unk> 0\n▁a 1\n▁b 2\n<blk> 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	said, err := tokens(at)
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 4 || said[1] != "▁a" || said.at("<blk>") != 3 {
		t.Errorf("the file names %v", said)
	}
}

// loaded is the models a machine holds, and the recording to hear with them.
// They are hundreds of megabytes and are not in the repository, so a machine
// without them says so and the rest of the package is still tested.
func loaded(t *testing.T) (*Transcriber, []byte) {
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
	by, raw := loaded(t)
	recorded := os.Getenv("NUMEN_RECORDING")

	sound, err := by.Open(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	defer sound.Close()
	t.Logf("%s is %d ms", recorded, sound.Length())

	found, err := sound.Speech(t.Context(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("the recording carries no speech")
	}

	var out []string
	for _, one := range found {
		text, err := by.Hear(t.Context(), one)
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
	by, raw := loaded(t)

	sound, rate, err := samples(raw)
	if err != nil {
		t.Fatal(err)
	}
	text, err := by.Hear(t.Context(), port.Audio{Samples: resampled(sound, rate, sampleRate)})
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("the recording said nothing")
	}
	t.Logf("%s", text)
}
