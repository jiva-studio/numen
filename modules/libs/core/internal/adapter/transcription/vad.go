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
// and that is how finely a span can be cut.
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
	scores, err := t.scoreSpeech(ctx, sound)
	if err != nil {
		return nil, err
	}

	windows := func(ms int) int { return ms * sampleRate / (1000 * speechWindow) }
	found := joinShortSpans(
		runs(scores, t.cutting.threshold(),
			windows(t.cutting.silence()), windows(t.cutting.pad()),
			windows(t.cutting.longest()), windows(t.cutting.shortest())),
		windows(t.cutting.least()), windows(t.cutting.longest()),
	)

	var out []port.Audio
	for _, one := range found {
		from := min(one.From*speechWindow, len(sound))
		to := min(one.To*speechWindow, len(sound))
		out = append(out, port.Audio{
			Samples: sound[from:to],
			From:    millis(from, sampleRate),
			To:      millis(to, sampleRate),
		})
	}
	return out, nil
}

// joinShortSpans puts a span too short to stand on its own together with the one
// after it, up to the longest a span may run to.
//
// A person pausing in the middle of a sentence closes a span, and what comes
// back is a line holding one word. A line is a thing somebody reads, and the
// model hears a sentence better than it hears a word out of one.
func joinShortSpans(found []span, least, longest int) []span {
	out := make([]span, 0, len(found))
	for _, one := range found {
		if len(out) == 0 {
			out = append(out, one)
			continue
		}
		last := &out[len(out)-1]
		short := last.To-last.From < least
		if short && one.To-last.From <= longest {
			last.To = one.To
			continue
		}
		out = append(out, one)
	}
	return out
}

// scoreSpeech is how sure the model is that each window of the recording carries
// speech. A last window short of the model's own is filled out with silence.
func (t *Transcriber) scoreSpeech(ctx context.Context, sound []float32) ([]float32, error) {
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

// A span is a run of the recording, counted in windows: from the first, up
// to but not including the last.
type span struct {
	From, To int
}

// runs are the spans of speech a run of scores holds.
//
// A span is opened by a window the model is sure enough about and closed by
// quiet windows enough after it, so that the pause between two words does not
// cut a sentence in half. Each is then widened by pad at both ends, and two
// that now meet are one.
func runs(scores []float32, threshold float32, silence, pad, longest, shortest int) []span {
	var out []span
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
			out = append(out, span{open, last + 1})
			open = -1
		}
	}
	if open >= 0 {
		out = append(out, span{open, last + 1})
	}

	var wider []span
	for _, one := range out {
		one.From = max(one.From-pad, 0)
		one.To = min(one.To+pad, len(scores))
		if n := len(wider); n > 0 && one.From <= wider[n-1].To {
			wider[n-1].To = one.To
			continue
		}
		wider = append(wider, one)
	}

	var cut []span
	for _, one := range wider {
		cut = append(cut, splitLongSpan(one, scores, longest)...)
	}

	out = out[:0]
	for _, one := range cut {
		if one.To-one.From >= shortest {
			out = append(out, one)
		}
	}
	return out
}

// splitLongSpan cuts a span that runs on too long into pieces the model is given
// one at a time. Each cut falls on the quietest window of the second half of
// what is left, so that a sentence is broken where the speaker paused.
func splitLongSpan(one span, scores []float32, longest int) []span {
	if longest <= 1 {
		return []span{one}
	}
	var out []span
	for one.To-one.From > longest {
		at := quietest(scores, one.From+longest/2, one.From+longest)
		out = append(out, span{one.From, at})
		one.From = at
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
