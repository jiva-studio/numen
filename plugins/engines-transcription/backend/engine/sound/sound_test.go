package sound

import (
	"encoding/binary"
	"math"
	"math/rand"
	"slices"
	"strings"
	"testing"
)

func encodeWav(rate, channels int, samples []int16) []byte {
	var body []byte
	for _, one := range samples {
		body = binary.LittleEndian.AppendUint16(body, uint16(one))
	}

	out := []byte("RIFF")
	out = binary.LittleEndian.AppendUint32(out, uint32(36+len(body)))
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, uint16(channels))
	out = binary.LittleEndian.AppendUint32(out, uint32(rate))
	out = binary.LittleEndian.AppendUint32(out, uint32(rate*channels*2))
	out = binary.LittleEndian.AppendUint16(out, uint16(channels*2))
	out = binary.LittleEndian.AppendUint16(out, 16)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(body)))
	return append(out, body...)
}

func encodeCustomWav(format, channels, rate, bits int, fmtExtra, data []byte) []byte {
	fmtBody := make([]byte, 16+len(fmtExtra))
	binary.LittleEndian.PutUint16(fmtBody[0:2], uint16(format))
	binary.LittleEndian.PutUint16(fmtBody[2:4], uint16(channels))
	binary.LittleEndian.PutUint32(fmtBody[4:8], uint32(rate))
	binary.LittleEndian.PutUint16(fmtBody[14:16], uint16(bits))
	copy(fmtBody[16:], fmtExtra)

	out := []byte("RIFF")
	out = binary.LittleEndian.AppendUint32(out, uint32(20+len(fmtBody)+len(fmtBody)%2+len(data)))
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(fmtBody)))
	out = append(out, fmtBody...)
	if len(fmtBody)%2 == 1 {
		out = append(out, 0)
	}
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	return append(out, data...)
}

// How long a recording is comes out of its header, before a sample is decoded.
func TestDurationFromTheHeader(t *testing.T) {
	raw := encodeWav(16000, 1, make([]int16, 8000))
	got, err := GetDuration(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != 500 {
		t.Errorf("half a second of one channel is %d ms", got)
	}

	raw = encodeWav(8000, 2, make([]int16, 8000))
	if got, err = GetDuration(raw); err != nil || got != 500 {
		t.Errorf("half a second of two channels is %d ms (%v)", got, err)
	}

	// 32-bit stereo WAV distinguishes bits/8 from bits/7 in sample frame width.
	raw32 := encodeCustomWav(3, 2, 16000, 32, nil, make([]byte, 8000*8))
	if got, err = GetDuration(raw32); err != nil || got != 500 {
		t.Errorf("half a second of 32-bit stereo is %d ms (%v)", got, err)
	}

	if got, err = GetDuration([]byte("RIFF\x04\x00\x00\x00WAVE")); err == nil || got != 0 {
		t.Errorf("headerless wav duration = %d, err = %v", got, err)
	}
}

// A file in a container this does not read is reported as one, and is not read
// as something else.
func TestARecordingInNoContainerThisReads(t *testing.T) {
	if got, err := GetDuration([]byte("this is not a recording")); err == nil || got != 0 {
		t.Errorf("a file that is not a recording returned %d, %v", got, err)
	}
}

// Two channels are heard as one: what they say at each moment, averaged.
func TestChannelsAreHeardAsOne(t *testing.T) {
	raw := encodeWav(16000, 2, []int16{16384, 0, -16384, 0, 8192, 8192, -8192, -8192})
	sound, rate, err := Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 16000 {
		t.Fatalf("the recording is at %d hertz", rate)
	}
	if len(sound) != 4 {
		t.Fatalf("two channels of eight samples came out as %d", len(sound))
	}
	if math.Abs(float64(sound[0])-0.25) > 1e-6 || math.Abs(float64(sound[1])+0.25) > 1e-6 {
		t.Errorf("the two channels came out as %v", sound)
	}
}

func TestContainerDetectionAndMonoBoundaries(t *testing.T) {
	if isWav([]byte("RIFF\x00\x00\x00\x00WAV")) {
		t.Fatal("expected 11-byte slice not to be wav")
	}
	if !isWav([]byte("RIFF\x00\x00\x00\x00WAVE")) {
		t.Fatal("expected 12-byte RIFF/WAVE slice to be wav")
	}
	if isWav([]byte("XXXX\x00\x00\x00\x00WAVE")) {
		t.Fatal("expected non-RIFF prefix not to be wav")
	}
	if isWav([]byte("RIFF\x00\x00\x00\x00XXXX")) {
		t.Fatal("expected non-WAVE tag not to be wav")
	}

	if !isMP3([]byte("ID3")) {
		t.Fatal("expected ID3 prefix to be mp3")
	}
	if isMP3([]byte{0xFF}) {
		t.Fatal("expected 1-byte slice not to be mp3")
	}
	if !isMP3([]byte{0xFF, 0xE0}) {
		t.Fatal("expected sync mark 0xFF 0xE0 to be mp3")
	}
	if isMP3([]byte{0xFF, 0x00}) {
		t.Fatal("expected 0xFF 0x00 not to be mp3")
	}
	if isMP3([]byte{0xFE, 0xE0}) {
		t.Fatal("expected 0xFE 0xE0 not to be mp3")
	}
	if isMP3([]byte{0x00, 0x00}) {
		t.Fatal("expected 0x00 0x00 not to be mp3")
	}

	negZero := float32(math.Copysign(0, -1))
	gotMono := mono([]float32{negZero}, 1)
	if len(gotMono) != 1 || math.Float32bits(gotMono[0]) != 0x80000000 {
		t.Fatalf("expected mono(1) to return slice unchanged with -0.0, got %v", gotMono)
	}
	gotZeroCh := mono([]float32{negZero}, 0)
	if len(gotZeroCh) != 1 || math.Float32bits(gotZeroCh[0]) != 0x80000000 {
		t.Fatalf("expected mono(0) to return slice unchanged, got %v", gotZeroCh)
	}
}

func TestWavParsingAndSampleEncodings(t *testing.T) {
	// Missing fmt header returns "no header".
	if _, _, err := Read([]byte("RIFF\x04\x00\x00\x00WAVE")); err == nil || !strings.Contains(err.Error(), "no header") {
		t.Fatalf("expected no header error, got %v", err)
	}

	// Short fmt header (< 16 bytes).
	shortFmt := []byte("RIFF\x1b\x00\x00\x00WAVEfmt \x0f\x00\x00\x00123456789012345")
	if _, _, err := Read(shortFmt); err == nil || !strings.Contains(err.Error(), "header is 15 bytes") {
		t.Fatalf("expected 15-byte header error, got %v", err)
	}

	// Data chunk first, followed by 16-byte fmt chunk at end of buffer (cap(body) == 16), plus empty chunk before fmt.
	var dataFirst []byte
	dataFirst = append(dataFirst, "RIFF\x00\x00\x00\x00WAVE"...)
	dataFirst = append(dataFirst, "JUNK\x00\x00\x00\x00"...)
	dataFirst = append(dataFirst, "JUNK\x03\x00\x00\x00abc\x00"...)
	dataFirst = append(dataFirst, "data\x02\x00\x00\x00\x00\x40"...)
	fmtEnd := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtEnd[0:2], 1)
	binary.LittleEndian.PutUint16(fmtEnd[2:4], 1)
	binary.LittleEndian.PutUint32(fmtEnd[4:8], 16000)
	binary.LittleEndian.PutUint16(fmtEnd[14:16], 16)
	dataFirst = append(dataFirst, "fmt \x10\x00\x00\x00"...)
	dataFirst = append(dataFirst, fmtEnd...)
	parsed, err := parseWav(slices.Clip(dataFirst))
	if err != nil || len(parsed.samples) != 2 {
		t.Fatalf("expected dataFirst wav to parse, got %+v, err %v", parsed, err)
	}

	// Trailing 7 bytes (< 8) are ignored; empty data chunk at exact end sets non-nil samples; oversized chunk is clamped.
	var clamped []byte
	clamped = append(clamped, "RIFF\x00\x00\x00\x00WAVEfmt \x10\x00\x00\x00"...)
	clamped = append(clamped, fmtEnd...)
	clamped = append(clamped, "data\x00\x00\x00\x00"...)
	parsedEmpty, err := parseWav(clamped)
	if err != nil || parsedEmpty.samples == nil || len(parsedEmpty.samples) != 0 {
		t.Fatalf("expected empty data chunk to set non-nil empty samples, got %#v, err %v", parsedEmpty.samples, err)
	}
	withTrailing7 := append(append([]byte{}, clamped...), "1234567"...)
	if _, err := parseWav(withTrailing7); err != nil {
		t.Fatalf("expected 7 trailing bytes to be ignored, got %v", err)
	}
	oversized := append([]byte{}, "RIFF\x00\x00\x00\x00WAVEfmt \x10\x00\x00\x00"...)
	oversized = append(oversized, fmtEnd...)
	oversized = append(oversized, "data\xff\x00\x00\x00\x00\x40"...)
	if parsedOver, err := parseWav(oversized); err != nil || len(parsedOver.samples) != 2 {
		t.Fatalf("expected oversized data chunk to be clamped to 2 bytes, got %d, err %v", len(parsedOver.samples), err)
	}

	// Invalid header parameters: channels=0, rate=0, bits=7; valid boundary rate=1, bits=8.
	if _, err := parseWav(encodeCustomWav(1, 0, 16000, 16, nil, nil)); err == nil {
		t.Fatal("expected channels=0 to fail")
	}
	if _, err := parseWav(encodeCustomWav(1, 1, 0, 16, nil, nil)); err == nil {
		t.Fatal("expected rate=0 to fail")
	}
	if _, err := parseWav(encodeCustomWav(1, 1, 16000, 7, nil, nil)); err == nil {
		t.Fatal("expected bits=7 to fail")
	}
	if _, err := parseWav(encodeCustomWav(1, 1, 1, 8, nil, []byte{128})); err != nil {
		t.Fatalf("expected rate=1, bits=8 to succeed, got %v", err)
	}

	// Extended header: 26 bytes (10 extra) reads subformat at [24:26]; 25 bytes (9 extra) leaves format=0xFFFE.
	extGood := make([]byte, 10)
	binary.LittleEndian.PutUint16(extGood[8:10], 1)
	if samples, _, err := Read(encodeCustomWav(0xFFFE, 1, 16000, 16, extGood, []byte{0x00, 0x40})); err != nil || len(samples) != 1 || math.Abs(float64(samples[0])-0.5) > 1e-6 {
		t.Fatalf("expected 26-byte extended wav to decode 0.5, got %v, err %v", samples, err)
	}
	if _, _, err := Read(encodeCustomWav(0xFFFE, 1, 16000, 16, make([]byte, 9), []byte{0x00, 0x40})); err == nil {
		t.Fatal("expected 25-byte extended wav without subformat to fail")
	}
	extIgnoredForPCM := make([]byte, 10)
	binary.LittleEndian.PutUint16(extIgnoredForPCM[8:10], 99)
	if samples, _, err := Read(encodeCustomWav(1, 1, 16000, 16, extIgnoredForPCM, []byte{0x00, 0x40})); err != nil || len(samples) != 1 {
		t.Fatalf("expected non-extended wav to ignore bytes 24:26, got %v, err %v", samples, err)
	}

	// readSound width==0 (bits < 8).
	if _, err := (riffWave{format: 1, channels: 1, rate: 16000, bits: 7}).readSound(); err == nil {
		t.Fatal("expected readSound with bits=7 to fail")
	}

	// 32-bit float, 64-bit float, 8-bit unsigned int, 24-bit signed int, and unsupported sample formats.
	f32Bytes := binary.LittleEndian.AppendUint32(nil, math.Float32bits(0.25))
	if samples, _, err := Read(encodeCustomWav(3, 1, 16000, 32, nil, f32Bytes)); err != nil || len(samples) != 1 || samples[0] != 0.25 {
		t.Fatalf("expected 32-bit float sample 0.25, got %v, err %v", samples, err)
	}
	f64Bytes := binary.LittleEndian.AppendUint64(nil, math.Float64bits(-0.5))
	if samples, _, err := Read(encodeCustomWav(3, 1, 16000, 64, nil, f64Bytes)); err != nil || len(samples) != 1 || samples[0] != -0.5 {
		t.Fatalf("expected 64-bit float sample -0.5, got %v, err %v", samples, err)
	}
	if samples, _, err := Read(encodeCustomWav(1, 1, 16000, 8, nil, []byte{192})); err != nil || len(samples) != 1 || samples[0] != 0.5 {
		t.Fatalf("expected 8-bit sample 0.5, got %v, err %v", samples, err)
	}
	if samples, _, err := Read(encodeCustomWav(1, 1, 16000, 24, nil, []byte{0x00, 0x00, 0x40})); err != nil || len(samples) != 1 || samples[0] != 0.5 {
		t.Fatalf("expected 24-bit sample 0.5, got %v, err %v", samples, err)
	}
	if _, _, err := Read(encodeCustomWav(3, 1, 16000, 16, nil, []byte{0x00, 0x40})); err == nil {
		t.Fatal("expected 16-bit float wav to fail")
	}
	if _, _, err := Read(encodeCustomWav(99, 1, 16000, 16, nil, []byte{0x00, 0x40})); err == nil {
		t.Fatal("expected unknown format 99 wav to fail")
	}

	// MP3 exact block multiple (32 frames = 36864 moments = 9 * 4096 moments) hits io.EOF on block boundary.
	if samples, _, err := Read(encodeMP3(32, 5)); err != nil || len(samples) != 32*1152 {
		t.Fatalf("expected 32-frame mp3 to decode 36864 moments, got %d, err %v", len(samples), err)
	}
	if _, _, err := Read([]byte("ID3invalid")); err == nil {
		t.Fatal("expected invalid mp3 to fail")
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
		if got := readSigned(one.raw); got != one.want {
			t.Errorf("%v is %d and should be %d", one.raw, got, one.want)
		}
	}
}

func BenchmarkWavDecode16Bit(b *testing.B) {
	pcm := make([]int16, 44100*60)
	rng := rand.New(rand.NewSource(1))
	for i := range pcm {
		pcm[i] = int16(rng.Intn(65536) - 32768)
	}
	w, err := parseWav(encodeWav(44100, 1, pcm))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(w.samples)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := w.readSound(); err != nil {
			b.Fatal(err)
		}
	}
}

// The 16-bit decoder agrees bit for bit with the per-sample decoder on random
// samples of every length remainder.
func TestInt16DecoderMatchesPerSampleDecoder(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, count := range []int{0, 1, 2, 3, 4, 5, 7, 8, 1001, 65536} {
		raw := make([]byte, count*2)
		rng.Read(raw)
		got := make([]float32, count)
		readInt16(got, raw)
		for i := range got {
			want := float32(readSigned(raw[i*2:(i+1)*2])) / float32(int64(1)<<15)
			if math.Float32bits(got[i]) != math.Float32bits(want) {
				t.Fatalf("count %d sample %d is %v and should be %v", count, i, got[i], want)
			}
		}
	}
}
