package transcription

import (
	"context"
	"fmt"
)

// A transducer is the three graphs as the decoding sees them: how many frames
// there are, how to reach one, what the predictor says after a token, and what
// the two say together.
type transducer struct {
	frames    int
	blank     int
	encoder   func(at int) []float32
	predictor func(token int) ([]float32, error)
	joint     func(frame, said []float32) ([]float32, error)
}

// decode reads the frames from the first to the last and answers with the
// tokens they carry.
//
// The joiner names a token and how many frames it covers. A token that is not
// the blank is said and moves the predictor on; the frame is left behind either
// way, so that a stretch is always read to its end.
func (d transducer) decode(ctx context.Context) ([]int, error) {
	// The predictor opens on the blank: nothing has been said yet.
	upto, err := d.predictor(d.blank)
	if err != nil {
		return nil, err
	}

	var said []int
	// How many words one frame has given. A frame may give several: the model
	// answers with a duration of nothing to say it has more to say here.
	spoken := 0
	for at := 0; at < d.frames; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scores, err := d.joint(d.encoder(at), upto)
		if err != nil {
			return nil, err
		}
		if len(scores) <= d.blank+1 {
			return nil, fmt.Errorf("the joiner answered with %d scores and the blank is %d", len(scores), d.blank)
		}
		token := findLargest(scores[:d.blank+1])
		step := findLargest(scores[d.blank+1:])
		if token != d.blank {
			said = append(said, token)
			if upto, err = d.predictor(token); err != nil {
				return nil, err
			}
			spoken++
		}
		// The blank moves on whatever it says, and a frame that has given all
		// the words one frame may give moves on too.
		if step < 1 && (token == d.blank || spoken >= mostPerFrame) {
			step = 1
		}
		if step > 0 {
			spoken = 0
		}
		at += step
	}
	return said, nil
}

// mostPerFrame is how many words one frame may give before the decoding moves
// on whatever the model says. It is what keeps a run of durations of nothing
// from standing on one frame for ever.
const mostPerFrame = 10

// findLargest is where the highest of a run of scores stands.
func findLargest(scores []float32) int {
	at := 0
	for i, v := range scores {
		if v > scores[at] {
			at = i
		}
	}
	return at
}
