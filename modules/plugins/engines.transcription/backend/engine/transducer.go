package engine

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
	encoded   int
	encoder   func(pos int, frame []float32) []float32
	predictor func(token int) ([]float32, error)
	joint     func(frame, predicted []float32) ([]float32, error)
}

// decode reads the frames from the first to the last and answers with the
// tokens they carry.
//
// The joiner names a token and how many frames it covers. A token that is not
// way, so that a stretch is always read to its end.
func (d transducer) decode(ctx context.Context) ([]int, error) {
	// The predictor opens on the blank: nothing has been said yet.
	upto, err := d.predictor(d.blank)
	if err != nil {
		return nil, err
	}

	frame := make([]float32, d.encoded)
	var tokens []int
	// How many words one frame has given. A frame may give several: the model
	// answers with a duration of nothing to say it has more to say here.
	tokensInFrame := 0
	for pos := 0; pos < d.frames; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		selected, err := d.choose(d.encoder(pos, frame), upto)
		if err != nil {
			return nil, err
		}
		if selected.token != d.blank {
			tokens = append(tokens, selected.token)
			if upto, err = d.predictor(selected.token); err != nil {
				return nil, err
			}
			tokensInFrame++
		}
		step := d.getStep(selected, tokensInFrame)
		if step > 0 {
			tokensInFrame = 0
		}
		pos += step
	}
	return tokens, nil
}

// choice is the token the joiner names for a frame and how many frames it
// covers.
type choice struct {
	token, step int
}

// choose is the choice the joiner makes for a frame.
func (d transducer) choose(frame, upto []float32) (choice, error) {
	scores, err := d.joint(frame, upto)
	if err != nil {
		return choice{}, err
	}
	if len(scores) <= d.blank+1 {
		return choice{}, fmt.Errorf("the joiner answered with %d scores and the blank is %d", len(scores), d.blank)
	}
	return choice{token: findLargest(scores[:d.blank+1]), step: findLargest(scores[d.blank+1:])}, nil
}

// getStep is how far the decoding moves on. The blank moves on whatever it
// says, and a frame that has given all the words one frame may give moves on
// too.
func (d transducer) getStep(c choice, tokensInFrame int) int {
	if c.step < 1 && (c.token == d.blank || tokensInFrame >= mostPerFrame) {
		return 1
	}
	return c.step
}

// mostPerFrame is how many words one frame may give before the decoding moves
// on whatever the model says. It is what keeps a run of durations of nothing
// from standing on one frame for ever.
const mostPerFrame = 10

// findLargest is where the highest of a run of scores stands.
func findLargest(scores []float32) int {
	pos := 0
	for i, v := range scores {
		if v > scores[pos] {
			pos = i
		}
	}
	return pos
}
