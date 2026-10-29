// Package transducer decodes the token sequence of a transducer model from its encoder frames.
package transducer

import (
	"context"
	"fmt"
)

// Decoder holds the three graphs used for decoding: the encoder (frame count
// and per-frame access), the predictor (output after a token) and the joiner
// (combines both).
type Decoder struct {
	Frames    int
	Blank     int
	Encoded   int
	Encoder   func(pos int, frame []float32) []float32
	Predictor func(token int) ([]float32, error)
	Joint     func(frame, predicted []float32) ([]float32, error)
}

// Decode reads the frames from first to last and returns the tokens they carry.
//
// The joiner names a token and how many frames it covers. A step of zero stays
// on the frame, except for the blank or a frame at mostPerFrame, which advance
// by one.
func (d Decoder) Decode(ctx context.Context) ([]int, error) {
	// The predictor starts from the blank token: no token has been emitted yet.
	upto, err := d.Predictor(d.Blank)
	if err != nil {
		return nil, err
	}

	frame := make([]float32, d.Encoded)
	var tokens []int
	// How many words one frame has given. A frame may give several: the model
	// answers with a duration of nothing to say it has more to say here.
	tokensInFrame := 0
	for pos := 0; pos < d.Frames; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		selected, err := d.choose(d.Encoder(pos, frame), upto)
		if err != nil {
			return nil, err
		}
		if selected.token != d.Blank {
			tokens = append(tokens, selected.token)
			if upto, err = d.Predictor(selected.token); err != nil {
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
func (d Decoder) choose(frame, upto []float32) (choice, error) {
	scores, err := d.Joint(frame, upto)
	if err != nil {
		return choice{}, err
	}
	if len(scores) <= d.Blank+1 {
		return choice{}, fmt.Errorf("the joiner answered with %d scores and the blank is %d", len(scores), d.Blank)
	}
	return choice{token: findLargest(scores[:d.Blank+1]), step: findLargest(scores[d.Blank+1:])}, nil
}

// getStep returns how far the decoding advances. The blank advances by at
// least one, and a frame that has emitted mostPerFrame tokens advances too.
func (d Decoder) getStep(c choice, tokensInFrame int) int {
	if c.step < 1 && (c.token == d.Blank || tokensInFrame >= mostPerFrame) {
		return 1
	}
	return c.step
}

// mostPerFrame is how many tokens one frame may emit before decoding advances
// regardless of the model's output. It bounds a run of zero durations on one
// frame.
const mostPerFrame = 10

// findLargest returns the index of the highest score.
func findLargest(scores []float32) int {
	pos := 0
	for i, v := range scores {
		if v > scores[pos] {
			pos = i
		}
	}
	return pos
}
