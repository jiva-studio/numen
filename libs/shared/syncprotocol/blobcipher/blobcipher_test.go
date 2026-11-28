package blobcipher_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
)

func openWith(l layout, blob []byte) ([]byte, error) {
	return blobcipher.Open(l.revisionKey, l.fileID, blob)
}

func sealWith(tb testing.TB, l layout, plain []byte, isText bool) []byte {
	tb.Helper()
	sealed, err := blobcipher.Seal(bytes.NewReader(l.salt[:]), l.revisionKey, l.fileID, plain, isText)
	if err != nil {
		tb.Fatal(err)
	}
	return sealed
}

func TestPaddedLength_FollowsTheLadderOfTheFormat(t *testing.T) {
	t.Parallel()

	for _, one := range []struct{ payload, want int }{
		{0, 4096},
		{1, 4096},
		{4088, 4096},
		{4089, 8192},
		{8184, 8192},
		{8185, 16384},
		{mebibyte/2 - 8, mebibyte / 2},
		{mebibyte/2 - 7, mebibyte},
		{mebibyte - 8, mebibyte},
		{mebibyte - 7, 2 * mebibyte},
		{2*mebibyte - 8, 2 * mebibyte},
		{2*mebibyte - 7, 3 * mebibyte},
		{3*mebibyte + 1, 4 * mebibyte},
	} {
		if got := blobcipher.ComputePaddedLength(one.payload); got != one.want {
			t.Errorf("ComputePaddedLength(%d) = %d, want %d", one.payload, got, one.want)
		}
	}
}

func TestSeal_ProducesTheFrozenBytesOfTheFormatForAnEmptyNoteAndForOneKibibyte(t *testing.T) {
	t.Parallel()
	l := newLayout()

	for name, plain := range map[string][]byte{"empty note": {}, "1 KiB note": newPlain(1024)} {
		got := sealWith(t, l, plain, false)

		want := buildBlob(t, l, plain)
		if !bytes.Equal(got, want) {
			t.Errorf("%s: sealed differs from the reference of the format at byte %d", name, firstDifference(got, want))
		}
		if len(got) != headerSize+4096+tagSize {
			t.Errorf("%s: stored size = %d, want %d", name, len(got), headerSize+4096+tagSize)
		}
	}
}

func TestSeal_ProducesTheFrozenBytesOfTheFormatForANoteOfThreeMebibytesAndOneByte(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := newPlain(3*mebibyte + 1)

	got := sealWith(t, l, plain, false)

	want := buildBlob(t, l, plain)
	if !bytes.Equal(got, want) {
		t.Errorf("sealed differs from the reference of the format at byte %d", firstDifference(got, want))
	}
	if wantSize := headerSize + 4*mebibyte + 4*tagSize; len(got) != wantSize {
		t.Errorf("stored size = %d, want %d", len(got), wantSize)
	}
}

func TestSeal_WritesTheHeaderFieldsOfTheFormat(t *testing.T) {
	t.Parallel()
	l := newLayout()

	plainHeader := sealWith(t, l, newPlain(10), false)[:headerSize]
	textHeader := sealWith(t, l, bytes.Repeat([]byte("milk "), 100), true)[:headerSize]

	if string(plainHeader[:4]) != "NSB1" || plainHeader[4] != 0 || plainHeader[5] != 20 || plainHeader[6] != 0 || plainHeader[7] != 0 {
		t.Errorf("header of a plain body = %x, want NSB1, no flags, chunk log2 20 and two zero bytes", plainHeader[:8])
	}
	if textHeader[4] != flagCompressed {
		t.Errorf("flags of a text body = %d, want the zstd bit", textHeader[4])
	}
	if !bytes.Equal(plainHeader[8:24], l.salt[:]) {
		t.Errorf("salt = %x, want the 16 bytes the random reader gave", plainHeader[8:24])
	}
}

func TestOpen_ReturnsWhatSealStoredForPlainAndTextBodiesOfEverySizeAroundALadderStep(t *testing.T) {
	t.Parallel()
	l := newLayout()

	for _, size := range []int{0, 1, 1024, 4087, 4088, 4089, mebibyte - 8, mebibyte - 7, mebibyte, 3*mebibyte + 1} {
		plain := newPlain(size)
		for _, isText := range []bool{false, true} {
			sealed := sealWith(t, l, plain, isText)
			opened, err := openWith(l, sealed)
			if err != nil || !bytes.Equal(opened, plain) {
				t.Errorf("%d bytes, text %v: opened %d bytes, error = %v, want the plain body", size, isText, len(opened), err)
			}
		}
	}
}

func TestSeal_StoresATextBodyCompressedAndPaddedToTheSmallestStep(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := bytes.Repeat([]byte("milk, bread, eggs\n"), 50_000)

	sealed := sealWith(t, l, plain, true)

	if len(sealed) != headerSize+4096+tagSize {
		t.Errorf("stored size = %d, want %d for a body that compresses to under one step", len(sealed), headerSize+4096+tagSize)
	}
	opened, err := openWith(l, sealed)
	if err != nil || !bytes.Equal(opened, plain) {
		t.Errorf("opened %d bytes, error = %v, want the plain body", len(opened), err)
	}
}

func TestSeal_NeverStoresAnyPartOfThePlainBody(t *testing.T) {
	t.Parallel()
	l := newLayout()
	plain := bytes.Repeat([]byte("secret shopping list "), 60)

	for _, isText := range []bool{false, true} {
		if sealed := sealWith(t, l, plain, isText); bytes.Contains(sealed, []byte("shopping")) {
			t.Errorf("text %v: the stored bytes hold a word of the plain body", isText)
		}
	}
}

func TestSeal_UsesANewSaltAndSoNewBytesForTheSameBodyEachTime(t *testing.T) {
	t.Parallel()
	var key [32]byte
	var fileID [16]byte

	first, err := blobcipher.Seal(rand.Reader, key, fileID, []byte("milk"), false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := blobcipher.Seal(rand.Reader, key, fileID, []byte("milk"), false)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) || bytes.Equal(first[8:24], second[8:24]) {
		t.Error("two seals of one body have the same salt or the same bytes")
	}
}

func TestSeal_ReportsARandomReaderThatFails(t *testing.T) {
	t.Parallel()
	var key [32]byte
	var fileID [16]byte

	sealed, err := blobcipher.Seal(bytes.NewReader([]byte{1, 2, 3}), key, fileID, []byte("milk"), false)

	if err == nil || sealed != nil {
		t.Errorf("sealed = %x, error = %v, want nil and an error for a reader with 3 bytes", sealed, err)
	}
}

func TestSeal_RefusesABodyWhoseStoredSizeIsOverTheLimit(t *testing.T) {
	t.Parallel()
	l := newLayout()
	const limit = 64 << 20

	over := make([]byte, limit-7-8)
	if sealed, err := blobcipher.Seal(bytes.NewReader(l.salt[:]), l.revisionKey, l.fileID, over, false); !errors.Is(err, blobcipher.ErrTooLarge) || sealed != nil {
		t.Errorf("a body of %d bytes: sealed %d bytes, error = %v, want ErrTooLarge", len(over), len(sealed), err)
	}
}

func TestSeal_AcceptsTheLargestBodyThatStaysWithinTheLimit(t *testing.T) {
	t.Parallel()
	l := newLayout()
	largest := make([]byte, 63*mebibyte-8)

	sealed, err := blobcipher.Seal(bytes.NewReader(l.salt[:]), l.revisionKey, l.fileID, largest, false)

	if err != nil {
		t.Fatal(err)
	}
	if want := headerSize + 63*mebibyte + 63*tagSize; len(sealed) != want || len(sealed) > 64<<20 {
		t.Errorf("stored size = %d, want %d within the limit", len(sealed), want)
	}
}

func TestOpen_RefusesAnyChangedByteOfASmallBody(t *testing.T) {
	t.Parallel()
	l := newLayout()
	sealed := sealWith(t, l, []byte("milk"), false)

	for i := range sealed {
		changed := bytes.Clone(sealed)
		changed[i] ^= 0x01
		if _, err := openWith(l, changed); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Fatalf("byte %d changed: error = %v, want ErrCannotOpen", i, err)
		}
	}
}

func TestOpen_RefusesAnotherRevisionKeyAndAnotherFileID(t *testing.T) {
	t.Parallel()
	l := newLayout()
	sealed := sealWith(t, l, []byte("milk"), false)
	otherKey, otherFile := l, l
	otherKey.revisionKey[0] ^= 1
	otherFile.fileID[0] ^= 1

	if _, err := openWith(otherKey, sealed); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("another revision key: error = %v, want ErrCannotOpen", err)
	}
	if _, err := openWith(otherFile, sealed); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("another file id: error = %v, want ErrCannotOpen", err)
	}
}

func TestOpen_RefusesABodyThatIsTruncatedOrTooShort(t *testing.T) {
	t.Parallel()
	l := newLayout()
	sealed := sealWith(t, l, []byte("milk"), false)

	for _, size := range []int{0, 1, 4, headerSize - 1, headerSize, headerSize + 1, headerSize + tagSize, len(sealed) - 1} {
		if _, err := openWith(l, sealed[:size]); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%d of %d bytes: error = %v, want ErrCannotOpen", size, len(sealed), err)
		}
	}
	if _, err := openWith(l, nil); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("nil body: error = %v, want ErrCannotOpen", err)
	}
}

func splitChunks(tb testing.TB, blob []byte, count int) (header []byte, chunks [][]byte) {
	tb.Helper()
	const stored = mebibyte + tagSize
	if len(blob) != headerSize+count*stored {
		tb.Fatalf("a body of %d bytes does not hold %d chunks of %d bytes", len(blob), count, stored)
	}
	for i := range count {
		start := headerSize + i*stored
		chunks = append(chunks, blob[start:start+stored])
	}
	return blob[:headerSize], chunks
}

func joinChunks(header []byte, chunks ...[]byte) []byte {
	joined := bytes.Clone(header)
	for _, chunk := range chunks {
		joined = append(joined, chunk...)
	}
	return joined
}

func TestOpen_RefusesABodyTruncatedAtTheBoundaryOfAChunk(t *testing.T) {
	t.Parallel()
	l := newLayout()
	header, chunks := splitChunks(t, sealWith(t, l, newPlain(3*mebibyte+1), false), 4)

	for keep := 1; keep < len(chunks); keep++ {
		if _, err := openWith(l, joinChunks(header, chunks[:keep]...)); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("first %d of 4 chunks: error = %v, want ErrCannotOpen", keep, err)
		}
	}
	if _, err := openWith(l, header); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("no chunk: error = %v, want ErrCannotOpen", err)
	}
}

func TestOpen_RefusesChunksInAnotherOrderAndAChunkRepeated(t *testing.T) {
	t.Parallel()
	l := newLayout()
	header, chunks := splitChunks(t, sealWith(t, l, newPlain(3*mebibyte+1), false), 4)

	for name, blob := range map[string][]byte{
		"two chunks swapped":    joinChunks(header, chunks[0], chunks[2], chunks[1], chunks[3]),
		"first and last swap":   joinChunks(header, chunks[3], chunks[1], chunks[2], chunks[0]),
		"a chunk repeated":      joinChunks(header, chunks[0], chunks[1], chunks[1], chunks[3]),
		"the last chunk twice":  joinChunks(header, chunks[0], chunks[1], chunks[2], chunks[3], chunks[3]),
		"the first chunk first": joinChunks(header, chunks[0], chunks[0], chunks[0], chunks[3]),
	} {
		if _, err := openWith(l, blob); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
}

func TestOpen_RefusesDataAfterTheLastChunk(t *testing.T) {
	t.Parallel()
	l := newLayout()
	sealed := sealWith(t, l, []byte("milk"), false)

	for name, blob := range map[string][]byte{
		"one byte":                 append(bytes.Clone(sealed), 0),
		"a tag":                    append(bytes.Clone(sealed), make([]byte, tagSize)...),
		"a copy of the last chunk": append(bytes.Clone(sealed), sealed[headerSize:]...),
	} {
		if _, err := openWith(l, blob); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
}

func TestOpen_RefusesAFinalChunkThatIsNotMarkedLast(t *testing.T) {
	t.Parallel()
	l := newLayout()
	l.unmarkLast = true

	if _, err := openWith(l, buildBlob(t, l, []byte("milk"))); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("error = %v, want ErrCannotOpen", err)
	}
}

func TestOpen_RefusesAHeaderWhoseCommitmentIsNotTheOneOfTheKeyEvenWithValidTags(t *testing.T) {
	t.Parallel()
	l := newLayout()
	l.commit = bytes.Repeat([]byte{0x77}, 32)

	if _, err := openWith(l, buildBlob(t, l, []byte("milk"))); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("error = %v, want ErrCannotOpen", err)
	}
}

func TestOpen_RefusesAHeaderOfAnotherFormatEvenWithValidTags(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*layout){
		"an unknown flag":         func(l *layout) { l.flags = 0x02 },
		"all flags":               func(l *layout) { l.flags = 0xFF },
		"a reserved byte":         func(l *layout) { l.reserved[0] = 1 },
		"the other reserved byte": func(l *layout) { l.reserved[1] = 1 },
		"chunk size 2^15":         func(l *layout) { l.chunkLog2 = 15 },
		"chunk size 2^23":         func(l *layout) { l.chunkLog2 = 23 },
		"chunk size 2^0":          func(l *layout) { l.chunkLog2 = 0 },
	} {
		l := newLayout()
		change(&l)
		if _, err := openWith(l, buildBlob(t, l, []byte("milk"))); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
}

func TestOpen_AcceptsEveryChunkSizeFromSixteenToTwentyTwo(t *testing.T) {
	t.Parallel()
	plain := newPlain(300_000)

	for log2 := byte(16); log2 <= 22; log2++ {
		l := newLayout()
		l.chunkLog2 = log2
		opened, err := openWith(l, buildBlob(t, l, plain))
		if err != nil || !bytes.Equal(opened, plain) {
			t.Errorf("chunk size 2^%d: opened %d bytes, error = %v, want the plain body", log2, len(opened), err)
		}
	}
}

func TestOpen_ReadsACompressedBodyThatAnotherEncoderMade(t *testing.T) {
	t.Parallel()
	l := newLayout()
	l.flags = flagCompressed
	plain := bytes.Repeat([]byte("milk, bread, eggs\n"), 1000)

	opened, err := openWith(l, buildBlob(t, l, compress(t, plain)))

	if err != nil || !bytes.Equal(opened, plain) {
		t.Errorf("opened %d bytes, error = %v, want the plain body", len(opened), err)
	}
}

func TestOpen_RefusesACompressedBodyThatIsNotValidZstd(t *testing.T) {
	t.Parallel()
	l := newLayout()
	l.flags = flagCompressed

	if _, err := openWith(l, buildBlob(t, l, []byte("not a zstd frame"))); !errors.Is(err, blobcipher.ErrCannotOpen) {
		t.Errorf("error = %v, want ErrCannotOpen", err)
	}
}

func TestOpen_ChecksTheLengthPrefixAgainstTheStream(t *testing.T) {
	t.Parallel()
	l := newLayout()
	stream := buildStream([]byte("milk"))

	exact := bytes.Clone(stream)
	putLength(exact, uint64(len(stream)-8))
	if opened, err := openWith(l, buildBlobOfStream(t, l, exact)); err != nil || len(opened) != len(stream)-8 {
		t.Errorf("a prefix that fills the stream: opened %d bytes, error = %v, want the whole stream after the prefix", len(opened), err)
	}
	for name, length := range map[string]uint64{
		"one more than the stream": uint64(len(stream) - 7),
		"the whole stream":         uint64(len(stream)),
		"a huge number":            1 << 62,
		"the largest number":       ^uint64(0),
	} {
		longer := bytes.Clone(stream)
		putLength(longer, length)
		if _, err := openWith(l, buildBlobOfStream(t, l, longer)); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
}

func TestOpen_RefusesAStreamThatIsShorterThanItsLengthPrefix(t *testing.T) {
	t.Parallel()
	l := newLayout()

	for _, size := range []int{1, 7} {
		if _, err := openWith(l, buildBlobOfStream(t, l, make([]byte, size))); !errors.Is(err, blobcipher.ErrCannotOpen) {
			t.Errorf("a stream of %d bytes: error = %v, want ErrCannotOpen", size, err)
		}
	}
}

func putLength(stream []byte, length uint64) {
	for i := range 8 {
		stream[i] = byte(length >> (8 * (7 - i)))
	}
}

func firstDifference(a, b []byte) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func TestSeal_RefusesATextBodyOverTheDecoderLimitThatCompressesBelowTheStoredLimit(t *testing.T) {
	t.Parallel()
	l := newLayout()
	over := bytes.Repeat([]byte("milk\n"), (256<<20)/5+1)

	sealed, err := blobcipher.Seal(bytes.NewReader(l.salt[:]), l.revisionKey, l.fileID, over, true)

	if !errors.Is(err, blobcipher.ErrTooLarge) || sealed != nil {
		t.Errorf("a text body of %d bytes: sealed %d bytes, error = %v, want ErrTooLarge", len(over), len(sealed), err)
	}
}
