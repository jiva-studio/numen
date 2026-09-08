package transcription

import (
	"context"
	"fmt"
	"runtime"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The names the segmenter's graph gives what it is asked and what it answers.
const (
	speechInput = "input"
	speechState = "state"
	speechRate  = "sr"
	speechScore = "output"
	speechAfter = "stateN"
)

// The model takes the recording in windows of this many samples, and carries
// this much state from one to the next. At 16 kHz a window is 32 milliseconds,
// and that is how finely a stretch can be cut.
const (
	speechWindow = 512
	speechMemory = 2 * 128
)

// segments cuts one recording into the segments that carry speech.
//
// The model answers for one window at a time, carrying what it made of the
// windows before. A run of windows it is sure enough about is a segment, closed
// by the quiet after it and widened a little at each end.
func (t *Transcriber) segments(ctx context.Context, sound []float32) ([]port.Audio, error) {
	scores, err := t.voiced(ctx, sound)
	if err != nil {
		return nil, err
	}

	windows := func(ms int) int { return ms * sampleRate / (1000 * speechWindow) }
	found := joined(
		runs(scores, t.cutting.threshold(),
			windows(t.cutting.silence()), windows(t.cutting.pad()),
			windows(t.cutting.longest()), windows(t.cutting.shortest())),
		windows(t.cutting.least()), windows(t.cutting.longest()),
	)

	var out []port.Audio
	for _, one := range found {
		from := min(one.start*speechWindow, len(sound))
		to := min(one.end*speechWindow, len(sound))
		out = append(out, port.Audio{
			Samples: sound[from:to],
			From:    millis(from, sampleRate),
			To:      millis(to, sampleRate),
		})
	}
	return out, nil
}

// joined puts a stretch too short to stand on its own together with the one
// after it, up to the longest a stretch may run to.
//
// A person pausing in the middle of a sentence closes a stretch, and what comes
// back is a line holding one word. A line is a thing somebody reads, and the
// model hears a sentence better than it hears a word out of one.
func joined(found []stretch, least, longest int) []stretch {
	out := make([]stretch, 0, len(found))
	for _, one := range found {
		if len(out) == 0 {
			out = append(out, one)
			continue
		}
		last := &out[len(out)-1]
		short := last.end-last.start < least
		if short && one.end-last.start <= longest {
			last.end = one.end
			continue
		}
		out = append(out, one)
	}
	return out
}

// voiced is how sure the model is that each window of the recording carries
// speech. A last window short of the model's own is filled out with silence.
func (t *Transcriber) voiced(ctx context.Context, sound []float32) ([]float32, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := make([]float32, speechMemory)
	rate := []int64{sampleRate}
	window := make([]float32, speechWindow)

	count := (len(sound) + speechWindow - 1) / speechWindow
	out := make([]float32, 0, count)
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		clear(window)
		copy(window, sound[i*speechWindow:min((i+1)*speechWindow, len(sound))])

		score, next, err := t.listen(window, state, rate)
		if err != nil {
			return nil, err
		}
		out = append(out, score)
		state = next
	}
	return out, nil
}

// listen is one window: how sure the model is, and the state it leaves behind.
func (t *Transcriber) listen(window, state []float32, rate []int64) (float32, []float32, error) {
	sound, err := ort.NewTensor([]int64{1, speechWindow}, window)
	if err != nil {
		return 0, nil, err
	}
	defer sound.Destroy()
	carried, err := ort.NewTensor([]int64{2, 1, speechMemory / 2}, state)
	if err != nil {
		return 0, nil, err
	}
	defer carried.Destroy()
	// The rate is asked for as one number and not as a tensor holding one.
	hertz, err := ort.NewTensor(nil, rate)
	if err != nil {
		return 0, nil, err
	}
	defer hertz.Destroy()

	out, err := t.segmenter.Run(map[string]*ort.Value{
		speechInput: sound, speechState: carried, speechRate: hertz,
	})
	runtime.KeepAlive(window)
	runtime.KeepAlive(state)
	runtime.KeepAlive(rate)
	if err != nil {
		return 0, nil, err
	}
	for _, v := range out {
		defer v.Destroy()
	}

	said, ok := out[speechScore]
	if !ok {
		return 0, nil, fmt.Errorf("the speech model answered with %v and not %s", names(out), speechScore)
	}
	score, err := ort.GetTensorData[float32](said)
	if err != nil {
		return 0, nil, err
	}
	if len(score) == 0 {
		return 0, nil, fmt.Errorf("the speech model answered with no score")
	}
	after, ok := out[speechAfter]
	if !ok {
		return 0, nil, fmt.Errorf("the speech model answered with %v and not %s", names(out), speechAfter)
	}
	raw, err := ort.GetTensorData[float32](after)
	if err != nil {
		return 0, nil, err
	}
	// What the graph answers with lives in the library's own memory, and is
	// copied out before the answer is let go of.
	return score[0], append([]float32(nil), raw...), nil
}

// A stretch is a run of the recording, counted in windows: from the first, up
// to but not including the last.
type stretch struct {
	start, end int
}

// runs are the stretches of speech a run of scores holds.
//
// A stretch is opened by a window the model is sure enough about and closed by
// quiet windows enough after it, so that the pause between two words does not
// cut a sentence in half. Each is then widened by pad at both ends, and two
// that now meet are one.
func runs(scores []float32, threshold float32, silence, pad, longest, shortest int) []stretch {
	var out []stretch
	open, last := -1, -1
	for i, score := range scores {
		if score >= threshold {
			if open < 0 {
				open = i
			}
			last = i
			continue
		}
		if open >= 0 && i-last > silence {
			out = append(out, stretch{open, last + 1})
			open = -1
		}
	}
	if open >= 0 {
		out = append(out, stretch{open, last + 1})
	}

	var wider []stretch
	for _, one := range out {
		one.start = max(one.start-pad, 0)
		one.end = min(one.end+pad, len(scores))
		if n := len(wider); n > 0 && one.start <= wider[n-1].end {
			wider[n-1].end = one.end
			continue
		}
		wider = append(wider, one)
	}

	var cut []stretch
	for _, one := range wider {
		cut = append(cut, divided(one, scores, longest)...)
	}

	out = out[:0]
	for _, one := range cut {
		if one.end-one.start >= shortest {
			out = append(out, one)
		}
	}
	return out
}

// divided cuts a stretch that runs on too long into pieces the model is given
// one at a time. Each cut falls on the quietest window of the second half of
// what is left, so that a sentence is broken where the speaker paused.
func divided(one stretch, scores []float32, longest int) []stretch {
	if longest <= 1 {
		return []stretch{one}
	}
	var out []stretch
	for one.end-one.start > longest {
		at := quietest(scores, one.start+longest/2, one.start+longest)
		out = append(out, stretch{one.start, at})
		one.start = at
	}
	return append(out, one)
}

// quietest is where the lowest score between two windows stands.
func quietest(scores []float32, from, to int) int {
	from, to = max(from, 0), min(to, len(scores))
	at := from
	for i := from; i < to; i++ {
		if scores[i] < scores[at] {
			at = i
		}
	}
	return at
}
