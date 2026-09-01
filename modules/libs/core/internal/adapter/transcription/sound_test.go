package transcription

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// written is a wav file of one signal: a header saying what the samples are,
// and the samples.
func written(rate, channels int, samples []int16) []byte {
	body := new(bytes.Buffer)
	for _, one := range samples {
		binary.Write(body, binary.LittleEndian, one)
	}

	out := new(bytes.Buffer)
	out.WriteString("RIFF")
	binary.Write(out, binary.LittleEndian, uint32(36+body.Len()))
	out.WriteString("WAVEfmt ")
	binary.Write(out, binary.LittleEndian, uint32(16))
	binary.Write(out, binary.LittleEndian, uint16(wavInteger))
	binary.Write(out, binary.LittleEndian, uint16(channels))
	binary.Write(out, binary.LittleEndian, uint32(rate))
	binary.Write(out, binary.LittleEndian, uint32(rate*channels*2))
	binary.Write(out, binary.LittleEndian, uint16(channels*2))
	binary.Write(out, binary.LittleEndian, uint16(16))
	out.WriteString("data")
	binary.Write(out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

// How long a recording is comes out of its header, before a sample is decoded.
func TestDurationFromTheHeader(t *testing.T) {
	raw := written(16000, 1, make([]int16, 8000))
	got, err := duration(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != 500 {
		t.Errorf("half a second of one channel is %d ms", got)
	}

	raw = written(8000, 2, make([]int16, 8000))
	if got, err = duration(raw); err != nil || got != 500 {
		t.Errorf("half a second of two channels is %d ms (%v)", got, err)
	}
}

// A file in a container this does not read is said to be one, and is not read
// as something else.
func TestARecordingInNoContainerThisReads(t *testing.T) {
	if _, err := duration([]byte("this is not a recording")); err == nil {
		t.Error("a file that is not a recording was read")
	}
}

// Two channels are heard as one: what they say at each moment, averaged.
func TestChannelsAreHeardAsOne(t *testing.T) {
	raw := written(16000, 2, []int16{16384, 0, -16384, 0})
	sound, rate, err := samples(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 16000 {
		t.Fatalf("the recording is at %d hertz", rate)
	}
	if len(sound) != 2 {
		t.Fatalf("two channels of four samples came out as %d", len(sound))
	}
	if math.Abs(float64(sound[0])-0.25) > 1e-6 || math.Abs(float64(sound[1])+0.25) > 1e-6 {
		t.Errorf("the two channels came out as %v", sound)
	}
}

// A recording at another rate is brought to the one the models take, and the
// tone it carries is still the tone it carried.
func TestResamplingKeepsTheTone(t *testing.T) {
	const from, hz = 48000, 440.0
	in := make([]float32, from/2)
	for n := range in {
		in[n] = float32(math.Sin(2 * math.Pi * hz * float64(n) / from))
	}

	out := resampled(in, from, sampleRate)
	if want := sampleRate / 2; out == nil || len(out) < want-8 || len(out) > want+8 {
		t.Fatalf("half a second at %d hertz came out as %d samples", from, len(out))
	}

	// Away from the ends, where the kernel reaches past the signal, the sample
	// is what the tone says at that moment.
	for n := 64; n < len(out)-64; n++ {
		want := math.Sin(2 * math.Pi * hz * float64(n) / sampleRate)
		if math.Abs(float64(out[n])-want) > 1e-2 {
			t.Fatalf("sample %d is %g, and the tone is %g there", n, out[n], want)
		}
	}
}

// A recording already at the rate the models take is left alone.
func TestResamplingWhatIsAlreadyRight(t *testing.T) {
	in := []float32{1, 2, 3}
	if out := resampled(in, sampleRate, sampleRate); &out[0] != &in[0] {
		t.Error("a recording at the right rate was resampled")
	}
}

func TestSignedSamplesOfAnyWidth(t *testing.T) {
	for _, one := range []struct {
		raw  []byte
		want int64
	}{
		{[]byte{0x00, 0x40}, 16384},
		{[]byte{0x00, 0xC0}, -16384},
		{[]byte{0xFF, 0xFF, 0x7F}, 8388607},
		{[]byte{0x00, 0x00, 0x80}, -8388608},
	} {
		if got := signed(one.raw); got != one.want {
			t.Errorf("%v is %d and should be %d", one.raw, got, one.want)
		}
	}
}
