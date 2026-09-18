package transcription

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/hajimehoshi/go-mp3"
	"github.com/mewkiz/flac"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A recording is a file opened for listening. Its length is read out of the
// header, and its samples are decoded when somebody asks where the speech is:
// a person is told how long a recording is while it is still being taken apart.
type recording struct {
	owner  *Transcriber
	raw    []byte
	length int

	// segments is where the speech is, and cut says the recording has been
	// through the segmenter. A recording carrying no speech is cut and holds
	// none.
	segments []port.Audio
	isCut    bool
	mu       sync.Mutex
}

// Open is a recording, ready to be listened to.
func (t *Transcriber) Open(ctx context.Context, raw []byte) (port.Recording, error) {
	length, err := duration(raw)
	if err != nil {
		return nil, err
	}
	return &recording{owner: t, raw: raw, length: length}, nil
}

// Length is how long the recording is, in milliseconds.
func (r *recording) Length() int { return r.length }

// Segments is what the segmenter cut the recording into, from a millisecond
// onward, at most count of them. A count of none is all of them.
//
// The whole file is cut at once, when the first stretch is asked for. The
// segmenter reads it from end to end, and where one stretch ends is decided by
// the silence after it.
func (r *recording) Segments(ctx context.Context, from, count int) ([]port.Audio, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isCut {
		sound, rate, err := samples(r.raw)
		if err != nil {
			return nil, err
		}
		at, err := resample(ctx, sound, rate, sampleRate)
		if err != nil {
			return nil, err
		}
		if r.segments, err = r.owner.segments(ctx, at); err != nil {
			return nil, err
		}
		r.raw, r.isCut = nil, true
	}

	var out []port.Audio
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

// The first bytes of each container this reads.
var (
	riff = []byte("RIFF")
	wave = []byte("WAVE")
	fLaC = []byte("fLaC")
	id3  = []byte("ID3")
)

// duration is how long a recording is, in milliseconds, out of its header
// alone.
func duration(raw []byte) (int, error) {
	switch {
	case isWav(raw):
		found, err := wav(raw)
		if err != nil {
			return 0, err
		}
		return millis(len(found.samples)/(found.channels*found.bits/8), found.rate), nil

	case bytes.HasPrefix(raw, fLaC):
		stream, err := flac.New(bytes.NewReader(raw))
		if err != nil {
			return 0, fmt.Errorf("the flac recording: %w", err)
		}
		defer stream.Close()
		return millis(int(stream.Info.NSamples), int(stream.Info.SampleRate)), nil

	case isMP3(raw):
		decoder, err := mp3.NewDecoder(bytes.NewReader(raw))
		if err != nil {
			return 0, fmt.Errorf("the mp3 recording: %w", err)
		}
		// Every mp3 comes out as two channels of sixteen bits, whatever it was
		// recorded as.
		return millis(int(decoder.Length()/4), decoder.SampleRate()), nil
	}
	return 0, fmt.Errorf("the recording is in no container this reads: wav, mp3 and flac are")
}

// samples is a whole recording as one channel of floating point, at whatever
// rate it was recorded at.
func samples(raw []byte) ([]float32, int, error) {
	switch {
	case isWav(raw):
		found, err := wav(raw)
		if err != nil {
			return nil, 0, err
		}
		sound, err := found.sound()
		if err != nil {
			return nil, 0, err
		}
		return mono(sound, found.channels), found.rate, nil

	case bytes.HasPrefix(raw, fLaC):
		return fromFlac(raw)

	case isMP3(raw):
		return fromMP3(raw)
	}
	return nil, 0, fmt.Errorf("the recording is in no container this reads: wav, mp3 and flac are")
}

func isWav(raw []byte) bool {
	return len(raw) >= 12 && bytes.HasPrefix(raw, riff) && bytes.Equal(raw[8:12], wave)
}

// isMP3 says whether a file is an mp3: a tag of what it holds, or the first
// frame's own mark.
func isMP3(raw []byte) bool {
	if bytes.HasPrefix(raw, id3) {
		return true
	}
	return len(raw) >= 2 && raw[0] == 0xFF && raw[1]&0xE0 == 0xE0
}

// millis is how long a count of samples at one rate lasts.
func millis(count, rate int) int {
	if rate <= 0 {
		return 0
	}
	return int(int64(count) * 1000 / int64(rate))
}

// A riffWave is what a wav file's header says, and the bytes of the samples
// themselves.
type riffWave struct {
	format   int
	channels int
	rate     int
	bits     int
	samples  []byte
}

// The two ways a wav file writes a sample, and the mark of a header that says
// which of them further along.
const (
	wavInteger  = 1
	wavFloating = 3
	wavExtended = 0xFFFE
)

// wav reads a wav file's header and finds its samples.
//
// A file is a run of chunks, and only two of them are wanted, so the rest are
// stepped over by their length. Each chunk sits on an even byte.
func wav(raw []byte) (riffWave, error) {
	var out riffWave
	var seen bool
	for at := 12; at+8 <= len(raw); {
		kind := string(raw[at : at+4])
		size := int(binary.LittleEndian.Uint32(raw[at+4 : at+8]))
		at += 8
		if size < 0 || at+size > len(raw) {
			size = len(raw) - at
		}
		body := raw[at : at+size]
		at += size
		if size%2 == 1 {
			at++
		}

		switch kind {
		case "fmt ":
			if len(body) < 16 {
				return out, fmt.Errorf("the wav recording's header is %d bytes", len(body))
			}
			out.format = int(binary.LittleEndian.Uint16(body[0:2]))
			out.channels = int(binary.LittleEndian.Uint16(body[2:4]))
			out.rate = int(binary.LittleEndian.Uint32(body[4:8]))
			out.bits = int(binary.LittleEndian.Uint16(body[14:16]))
			// An extended header says what the samples really are in its own
			// tail, where the first two bytes stand for the whole format.
			if out.format == wavExtended && len(body) >= 26 {
				out.format = int(binary.LittleEndian.Uint16(body[24:26]))
			}
			seen = true
		case "data":
			out.samples = body
		}
	}
	if !seen {
		return out, fmt.Errorf("the wav recording has no header")
	}
	// A sample narrower than a byte is a compressed container wearing a wav
	// header, which this reads none of.
	if out.channels <= 0 || out.rate <= 0 || out.bits < 8 {
		return out, fmt.Errorf("the wav recording is %d channels of %d bits at %d hertz",
			out.channels, out.bits, out.rate)
	}
	return out, nil
}

// sound is a wav file's samples as floating point, one channel after another as
// the file interleaves them.
func (w riffWave) sound() ([]float32, error) {
	width := w.bits / 8
	if width == 0 {
		return nil, fmt.Errorf("the wav recording is %d bits a sample", w.bits)
	}
	count := len(w.samples) / width
	out := make([]float32, count)
	for i := 0; i < count; i++ {
		raw := w.samples[i*width : (i+1)*width]
		switch {
		case w.format == wavFloating && w.bits == 32:
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw))
		case w.format == wavFloating && w.bits == 64:
			out[i] = float32(math.Float64frombits(binary.LittleEndian.Uint64(raw)))
		case w.format == wavInteger && w.bits == 8:
			// Eight bit samples are written without a sign, silence in the
			// middle of their range.
			out[i] = (float32(raw[0]) - 128) / 128
		case w.format == wavInteger:
			out[i] = float32(readSigned(raw)) / float32(int64(1)<<(w.bits-1))
		default:
			return nil, fmt.Errorf("the wav recording writes its samples as %d of %d bits", w.format, w.bits)
		}
	}
	return out, nil
}

// readSigned is one little-endian sample of any width, as a number.
func readSigned(raw []byte) int64 {
	var out int64
	for i := len(raw) - 1; i >= 0; i-- {
		out = out<<8 | int64(raw[i])
	}
	if raw[len(raw)-1]&0x80 != 0 {
		out -= int64(1) << (8 * len(raw))
	}
	return out
}

// fromMP3 decodes an mp3, which comes out as two channels of sixteen bits at
// the rate it was recorded at.
func fromMP3(raw []byte) ([]float32, int, error) {
	decoder, err := mp3.NewDecoder(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, fmt.Errorf("the mp3 recording: %w", err)
	}
	body, err := io.ReadAll(decoder)
	if err != nil {
		return nil, 0, fmt.Errorf("the mp3 recording: %w", err)
	}
	out := make([]float32, len(body)/2)
	for i := range out {
		out[i] = float32(int16(binary.LittleEndian.Uint16(body[2*i:]))) / 32768
	}
	return mono(out, 2), decoder.SampleRate(), nil
}

// fromFlac decodes a flac, frame by frame, one subframe to a channel.
func fromFlac(raw []byte) ([]float32, int, error) {
	stream, err := flac.New(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, fmt.Errorf("the flac recording: %w", err)
	}
	defer stream.Close()

	channels := int(stream.Info.NChannels)
	full := float32(int64(1) << (stream.Info.BitsPerSample - 1))
	out := make([]float32, 0, stream.Info.NSamples)
	for {
		frame, err := stream.ParseNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("the flac recording: %w", err)
		}
		for i := range frame.Subframes[0].Samples {
			var sum float32
			for _, channel := range frame.Subframes {
				sum += float32(channel.Samples[i]) / full
			}
			out = append(out, sum/float32(channels))
		}
	}
	return out, int(stream.Info.SampleRate), nil
}

// mono is many channels as one: what they say at each moment, averaged.
func mono(interleaved []float32, channels int) []float32 {
	if channels <= 1 {
		return interleaved
	}
	out := make([]float32, len(interleaved)/channels)
	for i := range out {
		var sum float32
		for c := 0; c < channels; c++ {
			sum += interleaved[i*channels+c]
		}
		out[i] = sum / float32(channels)
	}
	return out
}
