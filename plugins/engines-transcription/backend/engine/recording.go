package engine

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/resample"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/signal"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/sound"
)

// A recording is an audio file opened for transcription. Its length is read from
// the header, and its samples are decoded when the speech segments are requested.
type recording struct {
	owner  *Transcriber
	raw    []byte
	length int

	// segments holds the speech segments, and isCut reports that the recording has
	// been through the segmenter. A recording with no speech is cut and has no
	// segments.
	segments []Audio
	isCut    bool
	mu       sync.Mutex
}

// Open is a recording, ready to be listened to.
func (t *Transcriber) Open(_ context.Context, raw []byte) (Recording, error) {
	length, err := sound.GetDuration(raw)
	if err != nil {
		return nil, err
	}
	return &recording{
		owner:    t,
		raw:      raw,
		length:   length,
		segments: nil,
		isCut:    false,
		mu:       sync.Mutex{},
	}, nil
}

// GetDuration is how long the recording is, in milliseconds.
func (r *recording) GetDuration() int { return r.length }

// GetSegments returns the segments the segmenter cut the recording into, from a
// millisecond offset onward, at most count of them. A count of zero returns all.
//
// The whole file is cut at once, on the first request. The segmenter reads it
// from start to end, and the end of a segment is determined by the silence
// after it.
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
