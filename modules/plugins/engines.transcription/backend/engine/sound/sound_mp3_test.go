package sound

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hajimehoshi/go-mp3"

	"github.com/jiva-studio/numen/modules/plugins/engines.transcription/backend/engine/resample"
	"github.com/jiva-studio/numen/modules/plugins/engines.transcription/backend/engine/signal"
)

// bitWriter packs bits, most significant first.
type bitWriter struct {
	bytes []byte
	count int
}

func (w *bitWriter) write(value uint32, width int) {
	for i := width - 1; i >= 0; i-- {
		if w.count%8 == 0 {
			w.bytes = append(w.bytes, 0)
		}
		w.bytes[len(w.bytes)-1] |= byte(value>>uint(i)&1) << uint(7-w.count%8)
		w.count++
	}
}

// The codes of the first Huffman table for the pairs it holds, as bits and
// width, for the pair (x, y) with each of x and y in {0, 1}.
var pairCodes = [2][2][2]uint32{
	{{1, 1}, {1, 3}},
	{{2, 3}, {0, 3}},
}

// The bytes of one 44.1 kHz stereo frame at 128 kbit/s, no padding.
const mp3FrameSize = 417

// encodeMP3 is an mp3 of count frames in which every granule of both channels
// carries a few hundred small spectral values drawn from the seed.
func encodeMP3(count int, seed int64) []byte {
	random := rand.New(rand.NewSource(seed))
	var out bytes.Buffer
	for range count {
		var main bitWriter
		var lengths [4]int
		for part := range lengths {
			before := main.count
			writeGranules(random, &main)
			lengths[part] = main.count - before
		}

		var side bitWriter
		side.write(0, 9) // main_data_begin
		side.write(0, 3) // private bits
		side.write(0, 8) // scfsi
		for part := range lengths {
			side.write(uint32(lengths[part]), 12) // part2_3_length
			side.write(100, 9)                    // big_values
			side.write(180, 8)                    // global_gain
			side.write(0, 4)                      // scalefac_compress
			side.write(0, 1)                      // window_switching
			side.write(1, 5)                      // table_select
			side.write(1, 5)
			side.write(1, 5)
			side.write(0, 4) // region0_count
			side.write(0, 3) // region1_count
			side.write(0, 3) // preflag, scalefac_scale, count1table_select
		}

		frame := make([]byte, mp3FrameSize)
		copy(frame, []byte{0xFF, 0xFB, 0x90, 0x00})
		copy(frame[4:], side.bytes)
		copy(frame[4+len(side.bytes):], main.bytes)
		out.Write(frame)
	}
	return out.Bytes()
}

func writeGranules(random *rand.Rand, main *bitWriter) {
	for range 100 {
		x, y := random.Intn(2), random.Intn(2)
		code := pairCodes[x][y]
		main.write(code[0], int(code[1]))
		if x == 1 {
			main.write(uint32(random.Intn(2)), 1)
		}
		if y == 1 {
			main.write(uint32(random.Intn(2)), 1)
		}
	}
}

// decodeMP3Whole is the whole stereo output of the decoder, averaged to one
// channel once it is all in memory.
func decodeMP3Whole(t testing.TB, raw []byte) []float32 {
	t.Helper()
	decoder, err := mp3.NewDecoder(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatal(err)
	}
	stereo := make([]float32, len(body)/2)
	for i := range stereo {
		stereo[i] = float32(int16(binary.LittleEndian.Uint16(body[2*i:]))) / 32768
	}
	out := make([]float32, len(stereo)/2)
	for i := range out {
		out[i] = (stereo[2*i] + stereo[2*i+1]) / 2
	}
	return out
}

// The synthesized stream is read as an mp3 with a sound in it.
func TestTheSynthesizedMP3IsAudible(t *testing.T) {
	sound := decodeMP3Whole(t, encodeMP3(8, 1))
	if len(sound) != 8*1152 {
		t.Fatalf("eight frames came out as %d moments", len(sound))
	}
	var peak float32
	for _, one := range sound {
		peak = max(peak, one, -one)
	}
	if peak < 0.001 {
		t.Errorf("the synthesized mp3 is silent, its peak is %v", peak)
	}
}

// An mp3 comes out as the same samples, whatever block it is read in.
func TestMP3IsDecodedToTheSamplesOfTheWholeOutput(t *testing.T) {
	// 4096 moments to a block: 40 frames is more than a block and not a
	// multiple of one.
	raw := encodeMP3(40, 2)
	want := decodeMP3Whole(t, raw)

	got, rate, err := Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 44100 {
		t.Errorf("the mp3 is at %d hertz", rate)
	}
	if len(got) != len(want) {
		t.Fatalf("%d moments came out of the mp3, %d expected", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("moment %d is %v, expected %v", i, got[i], want[i])
		}
	}
}

// The memory an mp3 holds while it decodes follows the one channel it comes
// out as.
func TestMP3DecodeContainsMemoryInProportionToOneChannel(t *testing.T) {
	raw := encodeMP3(2000, 3)
	moments := 2000 * 1152

	peak := peakHeap(func() {
		got, _, err := fromMP3(raw)
		if err != nil {
			t.Error(err)
		}
		if len(got) != moments {
			t.Errorf("%d moments came out, %d expected", len(got), moments)
		}
	})
	if limit := 2*4*moments + 4<<20; peak > limit {
		t.Errorf("decoding %d moments held %d bytes at once, more than %d", moments, peak, limit)
	}
}

// peakHeap is the most the live heap grew while run ran, sampled with the
// collector running at every few allocated bytes.
func peakHeap(run func()) int {
	defer debug.SetGCPercent(debug.SetGCPercent(5))
	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	var peak atomic.Uint64
	stop, isDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(isDone)
		var now runtime.MemStats
		for {
			runtime.ReadMemStats(&now)
			if now.HeapAlloc > peak.Load() {
				peak.Store(now.HeapAlloc)
			}
			select {
			case <-stop:
				return
			case <-time.After(500 * time.Microsecond):
			}
		}
	}()
	run()
	close(stop)
	<-isDone
	return int(peak.Load()) - int(base.HeapAlloc)
}

// A recording is segmented from an mp3 as it is from any other container, and
// the context is honoured on the way.
func TestAnMP3RecordingHonoursItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	sound, rate, err := Read(encodeMP3(40, 4))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resample.Convert(ctx, sound, rate, signal.SampleRate, 0); err != context.Canceled {
		t.Errorf("resampling under a cancelled context returned %v", err)
	}
}
