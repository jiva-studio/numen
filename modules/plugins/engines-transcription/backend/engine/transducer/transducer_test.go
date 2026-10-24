package transducer

import (
	"context"
	"errors"
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
func TestDecodeReturnsWhatTheJoinerNames(t *testing.T) {
	const blank, steps = 3, 3
	named := map[int][]float32{
		0: answer(blank, 1, 2, steps),
		2: answer(blank, blank, 1, steps),
		3: answer(blank, 2, 1, steps),
	}

	var request []int
	response, err := Decoder{
		Frames:  4,
		Blank:   blank,
		Encoded: 1,
		Encoder: func(pos int, dst []float32) []float32 {
			dst[0] = float32(pos)
			return dst
		},
		Predictor: func(token int) ([]float32, error) {
			request = append(request, token)
			return []float32{float32(token)}, nil
		},
		Joint: func(frame, _ []float32) ([]float32, error) {
			return named[int(frame[0])], nil
		},
	}.Decode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(response) != 2 || response[0] != 1 || response[1] != 2 {
		t.Errorf("the frames said %v", response)
	}
	// The predictor opens on the blank and is moved on by each token said.
	if len(request) != 3 || request[0] != blank || request[1] != 1 || request[2] != 2 {
		t.Errorf("the predictor was given %v", request)
	}
}

// of them ends.
func TestDecodePreservesAFrameTheJoinerCoversWithNothing(t *testing.T) {
	const blank, steps = 3, 3
	visited := 0
	response, err := Decoder{
		Frames:    4,
		Blank:     blank,
		Encoder:   func(int, []float32) []float32 { visited++; return nil },
		Predictor: func(int) ([]float32, error) { return nil, nil },
		Joint: func([]float32, []float32) ([]float32, error) {
			return answer(blank, blank, 0, steps), nil
		},
	}.Decode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(response) != 0 {
		t.Errorf("frames carrying nothing said %v", response)
	}
	if visited != 4 {
		t.Errorf("visited %d frames, want 4", visited)
	}
}

func TestDecodeRefusesAJoinerThatIsTooNarrow(t *testing.T) {
	_, err := Decoder{
		Frames:    1,
		Blank:     8192,
		Encoder:   func(int, []float32) []float32 { return nil },
		Predictor: func(int) ([]float32, error) { return nil, nil },
		Joint: func([]float32, []float32) ([]float32, error) {
			return make([]float32, 16), nil
		},
	}.Decode(t.Context())
	if err == nil {
		t.Error("a joiner too narrow for the tokens was read")
	}
}

func TestDecodeRefusesJoinerWithNoStepScores(t *testing.T) {
	const blank = 3
	_, err := Decoder{
		Frames:    1,
		Blank:     blank,
		Encoder:   func(int, []float32) []float32 { return nil },
		Predictor: func(int) ([]float32, error) { return []float32{0}, nil },
		Joint: func([]float32, []float32) ([]float32, error) {
			return make([]float32, blank+1), nil
		},
	}.Decode(t.Context())
	if err == nil {
		t.Error("expected error for missing step scores, got nil")
	}
}

// An error from the context, the predictor, or the joiner stops decoding and is returned.
func TestDecodeErrors(t *testing.T) {
	errBoom := errors.New("boom")
	const blank, steps = 3, 3

	t.Run("initial predictor error", func(t *testing.T) {
		_, err := Decoder{
			Blank: blank,
			Predictor: func(int) ([]float32, error) {
				return nil, errBoom
			},
		}.Decode(t.Context())
		if !errors.Is(err, errBoom) {
			t.Fatalf("got %v, want %v", err, errBoom)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := Decoder{
			Frames:    2,
			Blank:     blank,
			Encoded:   1,
			Encoder:   func(int, []float32) []float32 { return nil },
			Predictor: func(int) ([]float32, error) { return []float32{0}, nil },
			Joint: func(_, _ []float32) ([]float32, error) {
				return answer(blank, 1, 1, steps), nil
			},
		}.Decode(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want %v", err, context.Canceled)
		}
	})

	t.Run("predictor error during decoding", func(t *testing.T) {
		_, err := Decoder{
			Frames:  2,
			Blank:   blank,
			Encoded: 1,
			Encoder: func(int, []float32) []float32 { return nil },
			Predictor: func(token int) ([]float32, error) {
				if token == 1 {
					return nil, errBoom
				}
				return []float32{0}, nil
			},
			Joint: func(_, _ []float32) ([]float32, error) {
				return answer(blank, 1, 1, steps), nil
			},
		}.Decode(t.Context())
		if !errors.Is(err, errBoom) {
			t.Fatalf("got %v, want %v", err, errBoom)
		}
	})

	t.Run("joiner error", func(t *testing.T) {
		_, err := Decoder{
			Frames:    2,
			Blank:     blank,
			Encoded:   1,
			Encoder:   func(int, []float32) []float32 { return nil },
			Predictor: func(int) ([]float32, error) { return []float32{0}, nil },
			Joint: func(_, _ []float32) ([]float32, error) {
				return nil, errBoom
			},
		}.Decode(t.Context())
		if !errors.Is(err, errBoom) {
			t.Fatalf("got %v, want %v", err, errBoom)
		}
	})
}

// A frame that gives a duration of zero moves on once mostPerFrame tokens have been said.
func TestDecodeAdvancesAfterMostPerFrame(t *testing.T) {
	const blank, steps = 3, 3
	callsPerFrame := make(map[int]int)

	response, err := Decoder{
		Frames:  3,
		Blank:   blank,
		Encoded: 1,
		Encoder: func(pos int, dst []float32) []float32 {
			dst[0] = float32(pos)
			return dst
		},
		Predictor: func(int) ([]float32, error) { return []float32{0}, nil },
		Joint: func(frame, _ []float32) ([]float32, error) {
			offset := int(frame[0])
			callsPerFrame[offset]++
			if offset < 2 {
				return answer(blank, 1, 0, steps), nil
			}
			return answer(blank, blank, 1, steps), nil
		},
	}.Decode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	wantTokens := 2 * mostPerFrame
	if len(response) != wantTokens {
		t.Fatalf("got %d tokens, want %d", len(response), wantTokens)
	}
	if callsPerFrame[0] != mostPerFrame {
		t.Errorf("frame 0 was called %d times, want %d", callsPerFrame[0], mostPerFrame)
	}
	if callsPerFrame[1] != mostPerFrame {
		t.Errorf("frame 1 was called %d times, want %d", callsPerFrame[1], mostPerFrame)
	}
	if callsPerFrame[2] != 1 {
		t.Errorf("frame 2 was called %d times, want 1", callsPerFrame[2])
	}
}

// A blank token with a step greater than one advances by that step.
func TestDecodeBlankWithStepAdvancesByStep(t *testing.T) {
	const blank, steps = 3, 3
	visited := []int{}

	_, err := Decoder{
		Frames:  4,
		Blank:   blank,
		Encoded: 1,
		Encoder: func(pos int, dst []float32) []float32 {
			visited = append(visited, pos)
			return dst
		},
		Predictor: func(int) ([]float32, error) { return []float32{0}, nil },
		Joint: func(_, _ []float32) ([]float32, error) {
			return answer(blank, blank, 2, steps), nil
		},
	}.Decode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(visited) != 2 || visited[0] != 0 || visited[1] != 2 {
		t.Errorf("visited %v, want [0 2]", visited)
	}
}

// Token scores and step scores are read from their respective halves of the scores slice.
func TestChooseSeparatesTokenAndStepScores(t *testing.T) {
	d := Decoder{
		Blank: 1,
		Joint: func(_, _ []float32) ([]float32, error) {
			return []float32{0, 0, 10, 0}, nil
		},
	}
	ch, err := d.choose(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ch.token != 0 {
		t.Errorf("ch.token = %d, want 0", ch.token)
	}
	if ch.step != 0 {
		t.Errorf("ch.step = %d, want 0", ch.step)
	}
}

// The largest score is chosen, and ties are broken by first position.
func TestFindLargest(t *testing.T) {
	if got := findLargest([]float32{1, 1, 0}); got != 0 {
		t.Errorf("findLargest tie got %d, want 0", got)
	}
	if got := findLargest([]float32{10, 5, 2}); got != 0 {
		t.Errorf("findLargest index 0 got %d, want 0", got)
	}
	if got := findLargest([]float32{3}); got != 0 {
		t.Errorf("findLargest single got %d, want 0", got)
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
		_, _ = Decoder{
			Frames:  frames,
			Blank:   blank,
			Encoded: encoded,
			Encoder: func(pos int, dst []float32) []float32 {
				for c := range dst {
					dst[c] = data[c*frames+pos]
				}
				return dst
			},
			Predictor: func(_ int) ([]float32, error) {
				return nil, nil
			},
			Joint: func(_, _ []float32) ([]float32, error) {
				return scores, nil
			},
		}.Decode(ctx)
	}
}
