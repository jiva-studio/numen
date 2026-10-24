package engine

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/plugins/engines-transcription/backend/engine/resample"
	"github.com/jiva-studio/numen/modules/plugins/engines-transcription/backend/engine/signal"
	"github.com/jiva-studio/numen/modules/plugins/engines-transcription/backend/engine/sound"
)

// A recording is a file opened for listening. Its length is read out of the
// header, and its samples are decoded when somebody asks where the speech is:
type recording struct {
	owner  *Transcriber
	raw    []byte
	length int

	// segments is where the speech is, and cut says the recording has been
	// through the segmenter. A recording carrying no speech is cut and holds
	// none.
	segments []Audio
	isCut    bool
	mu       sync.Mutex
}

// Open is a recording, ready to be listened to.
func (t *Transcriber) Open(ctx context.Context, raw []byte) (Recording, error) {
	length, err := sound.GetDuration(raw)
	if err != nil {
		return nil, err
	}
	return &recording{owner: t, raw: raw, length: length}, nil
}

// GetDuration is how long the recording is, in milliseconds.
func (r *recording) GetDuration() int { return r.length }

// GetSegments is what the segmenter cut the recording into, from a millisecond
// onward, at most count of them. A count of none is all of them.
//
// The whole file is cut at once, when the first stretch is asked for. The
// segmenter reads it from end to end, and where one stretch ends is decided by
// the silence after it.
func (r *recording) GetSegments(ctx context.Context, from, count int) ([]Audio, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isCut {
		samples, rate, err := sound.Read(r.raw)
		if err != nil {
			return nil, err
		}
		pos, err := resample.Convert(ctx, samples, rate, signal.SampleRate, r.owner.workers)
		if err != nil {
			return nil, err
		}
		cut, err := r.owner.cutter.Split(ctx, pos)
		if err != nil {
			return nil, err
		}
		r.segments = make([]Audio, len(cut))
		for i, one := range cut {
			r.segments[i] = Audio(one)
		}
		r.raw, r.isCut = nil, true
	}

	var out []Audio
	for _, one := range r.segments {
		if one.From < from {
			continue
		}
		out = append(out, one)
		if count > 0 && len(out) == count {
			break
		}
	}
	return out, nil
}

func (r *recording) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raw, r.segments = nil, nil
	return nil
}
