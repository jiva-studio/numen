package blobcipher_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
)

const chunkSize = mebibyte

type spySource struct {
	data     []byte
	requests []span
}

type span struct{ off, length int64 }

func (s *spySource) ReadAt(p []byte, off int64) (int, error) {
	s.requests = append(s.requests, span{off: off, length: int64(len(p))})
	if off >= int64(len(s.data)) {
		return 0, io.EOF
	}
	n := copy(p, s.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func newSealer(tb testing.TB, l layout, size int) *blobcipher.Sealer {
	tb.Helper()
	sealer, err := blobcipher.NewSealer(l.revisionKey, l.fileID, l.salt, int64(size))
	if err != nil {
		tb.Fatal(err)
	}
	return sealer
}

func readAllStored(tb testing.TB, sealer *blobcipher.Sealer, plain io.ReaderAt, step int) []byte {
	tb.Helper()
	stored := make([]byte, 0, sealer.StoredSize())
	buffer := make([]byte, step)
	for off := int64(0); off < sealer.StoredSize(); {
		n, err := sealer.ReadStoredAt(buffer, off, plain)
		stored = append(stored, buffer[:n]...)
		off += int64(n)
		if err != nil && !errors.Is(err, io.EOF) {
			tb.Fatalf("read at %d: %v", off, err)
		}
		if n == 0 {
			tb.Fatalf("read at %d returned no bytes and error %v", off, err)
		}
	}
	return stored
}

func TestSealer_StoresTheBytesTheFormatDefinesForPlainBodiesOfEverySizeAroundAStep(t *testing.T) {
	t.Parallel()
	l := newLayout()

	for _, size := range []int{0, 1, 4087, mebibyte - 8, mebibyte - 7, 3*mebibyte + 1} {
		plain := newPlain(size)
		sealer := newSealer(t, l, size)

		got := readAllStored(t, sealer, bytes.NewReader(plain), 700_000)

		want := buildBlob(t, l, plain)
		if !bytes.Equal(got, want) {
			t.Errorf("%d bytes: stored differs from the reference of the format at byte %d", size, firstDifference(got, want))
		}
		if sealer.StoredSize() != int64(len(want)) {
			t.Errorf("%d bytes: StoredSize = %d, want %d", size, sealer.StoredSize(), len(want))
		}
	}
}

func TestSealer_ReturnsTheBytesOfSealForTheSameSaltKeyAndPlainWhenNothingIsCompressed(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(2*mebibyte + 5)

	got := readAllStored(t, newSealer(t, l, len(plain)), bytes.NewReader(plain), 4*mebibyte)

	if want := sealWith(t, l, plain, false); !bytes.Equal(got, want) {
		t.Errorf("sealer differs from Seal at byte %d", firstDifference(got, want))
	}
}

func TestSealer_AnswersAnyRangeWithTheSliceOfTheWholeStream(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(3*mebibyte + 1)
	want := buildBlob(t, l, plain)
	sealer := newSealer(t, l, len(plain))
	state := uint32(88172645)

	for range 40 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		off := int(state) % len(want)
		length := int(state>>7)%(2*mebibyte) + 1
		buffer := make([]byte, length)

		n, err := sealer.ReadStoredAt(buffer, int64(off), bytes.NewReader(plain))

		wantEnd := min(off+length, len(want))
		if !bytes.Equal(buffer[:n], want[off:wantEnd]) {
			t.Fatalf("range %d+%d: bytes differ from the whole stream", off, length)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("range %d+%d: error = %v", off, length, err)
		}
	}
}

func TestSealer_ReturnsEndOfFileWithTheCountAtTheEndOfTheStream(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(10)
	sealer := newSealer(t, l, len(plain))
	size := sealer.StoredSize()
	buffer := make([]byte, 100)

	n, err := sealer.ReadStoredAt(buffer, size-30, bytes.NewReader(plain))
	if n != 30 || !errors.Is(err, io.EOF) {
		t.Errorf("30 bytes before the end: n = %d, error = %v, want 30 and io.EOF", n, err)
	}
	n, err = sealer.ReadStoredAt(buffer, size, bytes.NewReader(plain))
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Errorf("at the end: n = %d, error = %v, want 0 and io.EOF", n, err)
	}
}

func TestSealer_ReadsOnlyThePlainRangeOfTheChunksThatOverlapTheRequest(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(5 * mebibyte)
	source := &spySource{data: plain}
	sealer := newSealer(t, l, len(plain))
	inThirdChunk := blobcipher.GetChunkOffset(20, 2) + 100

	if _, err := sealer.ReadStoredAt(make([]byte, 1000), inThirdChunk, source); err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}

	for _, request := range source.requests {
		low, high := int64(2*chunkSize-8), int64(3*chunkSize-8)
		if request.off < low || request.off+request.length > high {
			t.Errorf("plain read %d+%d is outside the plain range %d to %d of the third chunk", request.off, request.length, low, high)
		}
	}
	if len(source.requests) == 0 {
		t.Error("no plain read for a request inside a chunk")
	}
}

func TestSealer_NeverAsksTheSourceForMoreThanOneChunkAtOnceForAFileOfSeventyMebibytes(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(70 * mebibyte)
	source := &spySource{data: plain}
	sealer := newSealer(t, l, len(plain))
	hash := sha256.New()
	buffer := make([]byte, 16*mebibyte)

	for off := int64(0); off < sealer.StoredSize(); {
		n, err := sealer.ReadStoredAt(buffer, off, source)
		hash.Write(buffer[:n])
		off += int64(n)
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
	}

	for _, request := range source.requests {
		if request.length > chunkSize {
			t.Fatalf("a plain read of %d bytes is larger than one chunk", request.length)
		}
	}
	want := sha256.Sum256(buildBlob(t, l, plain))
	if !bytes.Equal(hash.Sum(nil), want[:]) {
		t.Error("hash of the stored stream differs from the reference of the format")
	}
}

func TestSealer_ReturnsTheSameBytesForTheSameSaltAndOtherBytesForAnotherSalt(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(mebibyte + 1)
	again := readAllStored(t, newSealer(t, l, len(plain)), bytes.NewReader(plain), mebibyte)
	first := readAllStored(t, newSealer(t, l, len(plain)), bytes.NewReader(plain), 100_000)
	other := l
	other.salt[0] ^= 1

	second := readAllStored(t, newSealer(t, other, len(plain)), bytes.NewReader(plain), mebibyte)

	if !bytes.Equal(first, again) {
		t.Error("two sealers of one key, salt and file id give different bytes")
	}
	if bytes.Equal(first, second) {
		t.Error("two salts give the same bytes")
	}
}

func TestSealer_FailsWhenThePlainSourceIsShorterThanTheDeclaredSize(t *testing.T) {
	t.Parallel()
	l := newLayout()
	sealer := newSealer(t, l, 3*mebibyte)

	_, err := sealer.ReadStoredAt(make([]byte, 2*mebibyte), 2*mebibyte, bytes.NewReader(newPlain(mebibyte)))

	if !errors.Is(err, blobcipher.ErrShortPlain) {
		t.Errorf("error = %v, want ErrShortPlain", err)
	}
}

func TestNewSealer_RefusesAStoredSizeOverTheMultipartLimit(t *testing.T) {
	t.Parallel()
	l := newLayout()
	const overLimit = 5 << 40

	sealer, err := blobcipher.NewSealer(l.revisionKey, l.fileID, l.salt, overLimit)

	if !errors.Is(err, blobcipher.ErrTooLarge) || sealer != nil {
		t.Errorf("sealer = %v, error = %v, want nil and ErrTooLarge", sealer, err)
	}
}

func TestGenerateSalt_ReadsSixteenBytesFromTheReaderAndReportsAShortReader(t *testing.T) {
	t.Parallel()
	source := bytes.Repeat([]byte{7}, 16)

	salt, err := blobcipher.GenerateSalt(bytes.NewReader(source))
	if err != nil || !bytes.Equal(salt[:], source) {
		t.Errorf("salt = %x, error = %v, want the 16 bytes of the reader", salt, err)
	}
	if _, err := blobcipher.GenerateSalt(bytes.NewReader(source[:5])); err == nil {
		t.Error("a reader with 5 bytes gave no error")
	}
}

func readPayloadOf(tb testing.TB, l layout, stored []byte) ([]byte, error) {
	tb.Helper()
	reader, err := blobcipher.OpenStream(l.revisionKey, l.fileID, bytes.NewReader(stored))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func TestOpenStream_ReturnsThePayloadWithoutThePrefixAndThePadding(t *testing.T) {
	t.Parallel()
	l := newLayout()

	for _, size := range []int{0, 1, 4088, mebibyte - 8, mebibyte - 7, 3*mebibyte + 1} {
		plain := newPlain(size)

		got, err := readPayloadOf(t, l, buildBlob(t, l, plain))

		if err != nil || !bytes.Equal(got, plain) {
			t.Errorf("%d bytes: read %d bytes, error = %v, want the payload", size, len(got), err)
		}
	}
}

func TestOpenStream_ReadsNothingOfAChunkThatDoesNotVerify(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(3*mebibyte + 1)
	stored := buildBlob(t, l, plain)
	stored[blobcipher.GetChunkOffset(20, 2)+10] ^= 1

	got, err := readPayloadOf(t, l, stored)

	if !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("error = %v, want ErrCannotOpen", err)
	}
	if len(got) > 2*chunkSize-8 || !bytes.Equal(got, plain[:len(got)]) {
		t.Errorf("read %d bytes before the error, want at most the two verified chunks", len(got))
	}
}

func TestOpenStream_RefusesWhatOpenRefuses(t *testing.T) {
	t.Parallel()
	l := newLayout()
	stored := buildBlob(t, l, newPlain(3*mebibyte+1))
	header, chunks := splitChunks(t, stored, 4)
	unmarked := l
	unmarked.unmarkLast = true
	foreignKey := l
	foreignKey.revisionKey[0] ^= 1
	flipped := bytes.Clone(stored)
	flipped[headerSize+5] ^= 1
	badCommit := l
	badCommit.commit = bytes.Repeat([]byte{0x77}, 32)

	for name, blob := range map[string][]byte{
		"truncated at a chunk":     joinChunks(header, chunks[:2]...),
		"chunks swapped":           joinChunks(header, chunks[0], chunks[2], chunks[1], chunks[3]),
		"a chunk repeated":         joinChunks(header, chunks[0], chunks[1], chunks[1], chunks[3]),
		"the last chunk twice":     joinChunks(header, chunks[0], chunks[1], chunks[2], chunks[3], chunks[3]),
		"a flipped bit":            flipped,
		"a data tail":              append(bytes.Clone(stored), 0),
		"no chunk":                 header,
		"the last not marked":      buildBlob(t, unmarked, newPlain(100)),
		"a commitment of no key":   buildBlob(t, badCommit, newPlain(100)),
		"a prefix over the stream": withPrefix(t, l, 1<<40),
	} {
		if _, err := readPayloadOf(t, l, blob); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
	if _, err := readPayloadOf(t, foreignKey, stored); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("another key: error = %v, want ErrCannotOpen", err)
	}
}

func withPrefix(tb testing.TB, l layout, length uint64) []byte {
	tb.Helper()
	stream := buildStream([]byte("milk"))
	putLength(stream, length)
	return buildBlobOfStream(tb, l, stream)
}

func TestOpenStream_RefusesACompressedBody(t *testing.T) {
	t.Parallel()
	l := newLayout()
	l.flags = flagCompressed

	_, err := readPayloadOf(t, l, buildBlob(t, l, compress(t, []byte("milk milk milk"))))

	if !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("error = %v, want ErrCannotOpen", err)
	}
}

func TestOpenStreamFrom_ContinuesAtEveryChunkWithTheHeaderOfTheFirstRequest(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(3*mebibyte + 1)
	stored := buildBlob(t, l, plain)
	var header [56]byte
	copy(header[:], stored)

	for index := uint64(1); index < 4; index++ {
		reader, err := blobcipher.OpenStreamFrom(l.revisionKey, l.fileID, header, index, int64(len(plain)), bytes.NewReader(stored[blobcipher.GetChunkOffset(20, index):]))
		if err != nil {
			t.Fatalf("chunk %d: %v", index, err)
		}

		got, err := io.ReadAll(reader)

		want := plain[int(index)*chunkSize-8:]
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("chunk %d: read %d bytes, error = %v, want the %d bytes after it", index, len(got), err, len(want))
		}
	}
}

func TestOpenStreamFrom_RefusesAChunkOfAnotherPositionAndAHeaderOfAnotherKey(t *testing.T) {
	t.Parallel()
	l := newLayout()
	stored := buildBlob(t, l, newPlain(3*mebibyte+1))
	var header [56]byte
	copy(header[:], stored)
	foreign := l.revisionKey
	foreign[3] ^= 1

	reader, err := blobcipher.OpenStreamFrom(l.revisionKey, l.fileID, header, 3, 3*mebibyte+1, bytes.NewReader(stored[blobcipher.GetChunkOffset(20, 2):]))
	if err == nil {
		_, err = io.ReadAll(reader)
	}
	if !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("a chunk read as another position: error = %v, want ErrCannotOpen", err)
	}
	_, err = blobcipher.OpenStreamFrom(foreign, l.fileID, header, 1, 3*mebibyte+1, bytes.NewReader(stored[blobcipher.GetChunkOffset(20, 1):]))
	if !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("another key: error = %v, want ErrCannotOpen", err)
	}
}

func TestGetChunkOffset_IsTheHeaderPlusWholeStoredChunks(t *testing.T) {
	t.Parallel()

	for _, one := range []struct {
		log2  byte
		index uint64
		want  int64
	}{
		{20, 0, 56},
		{20, 1, 56 + mebibyte + 16},
		{20, 69, 56 + 69*(mebibyte+16)},
		{16, 3, 56 + 3*(1<<16+16)},
		{22, 2, 56 + 2*(1<<22+16)},
	} {
		if got := blobcipher.GetChunkOffset(one.log2, one.index); got != one.want {
			t.Errorf("GetChunkOffset(%d, %d) = %d, want %d", one.log2, one.index, got, one.want)
		}
	}
}

func TestParseHeader_ReadsFlagsAndChunkSizeAndRefusesWhatIsNotAHeader(t *testing.T) {
	t.Parallel()
	l := newLayout()
	l.chunkLog2 = 18
	stored := buildBlob(t, l, []byte("milk"))

	parsed, err := blobcipher.ParseHeader(stored[:56])
	if err != nil || parsed.ChunkLog2 != 18 || parsed.Flags != 0 {
		t.Errorf("parsed = %+v, error = %v, want chunk log2 18 and no flags", parsed, err)
	}
	for name, header := range map[string][]byte{
		"short":     stored[:55],
		"bad magic": append([]byte("XSB1"), stored[4:56]...),
		"log2 15":   append(append(bytes.Clone(stored[:5]), 15), stored[6:56]...),
	} {
		if _, err := blobcipher.ParseHeader(header); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
}

func TestOpenStreamFrom_ReturnsNothingWhenTheResumePointIsInsideThePadding(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(3*mebibyte + 1)
	stored := buildBlob(t, l, plain)
	var header [56]byte
	copy(header[:], stored)

	reader, err := blobcipher.OpenStreamFrom(l.revisionKey, l.fileID, header, 2, 1, bytes.NewReader(stored[blobcipher.GetChunkOffset(20, 2):]))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)

	if err != nil || len(got) != 0 {
		t.Errorf("read %d bytes, error = %v, want no bytes and no error", len(got), err)
	}
}
