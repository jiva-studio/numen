package transcription

import (
	"testing"
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
		encoded: 1,
		encoder: func(at int, dst []float32) []float32 {
			dst[0] = float32(at)
			return dst
		},
		predictor: func(token int) ([]float32, error) {
			asked = append(asked, token)
			return []float32{float32(token)}, nil
		},
		joint: func(frame, _ []float32) ([]float32, error) {
			return named[int(frame[0])], nil
		},
	}.decode(t.Context())
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
		frames:    4,
		blank:     blank,
		encoder:   func(int, []float32) []float32 { return nil },
		predictor: func(int) ([]float32, error) { return nil, nil },
		joint: func([]float32, []float32) ([]float32, error) {
			return answer(blank, blank, 0, steps), nil
		},
	}.decode(t.Context())
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
		frames:    1,
		blank:     8192,
		encoder:   func(int, []float32) []float32 { return nil },
		predictor: func(int) ([]float32, error) { return nil, nil },
		joint: func([]float32, []float32) ([]float32, error) {
			return make([]float32, 16), nil
		},
	}.decode(t.Context())
	if err == nil {
		t.Error("a joiner too narrow for the tokens was read")
	}
}

func BenchmarkTransducerDecode(b *testing.B) {
	const (
		frames  = 500
		encoded = 1024
		blank   = 1024
		steps   = 3
	)
	data := make([]float32, encoded*frames)
	scores := answer(blank, blank, 1, steps)
	ctx := b.Context()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = transducer{
			frames:  frames,
			blank:   blank,
			encoded: encoded,
			encoder: func(at int, dst []float32) []float32 {
				for c := range dst {
					dst[c] = data[c*frames+at]
				}
				return dst
			},
			predictor: func(token int) ([]float32, error) {
				return nil, nil
			},
			joint: func(frame, said []float32) ([]float32, error) {
				return scores, nil
			},
		}.decode(ctx)
	}
}
