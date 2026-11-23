package slip39_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

func repeating(b byte) io.Reader {
	return iotest.OneByteReader(bytes.NewReader(bytes.Repeat([]byte{b}, 1<<16)))
}

func mustSplit(tb testing.TB, random io.Reader, secret []byte, threshold, count int) []slip39.Share {
	tb.Helper()
	shares, err := slip39.Split(random, secret, threshold, count)
	if err != nil {
		tb.Fatalf("Split(%d bytes, %d of %d) error = %v", len(secret), threshold, count, err)
	}
	return shares
}

func mustCombine(tb testing.TB, shares []slip39.Share) []byte {
	tb.Helper()
	secret, err := slip39.Combine(shares, nil)
	if err != nil {
		tb.Fatalf("Combine error = %v", err)
	}
	return secret
}

func secretOf(size int) []byte {
	secret := make([]byte, size)
	for i := range secret {
		secret[i] = byte(i*7 + 1)
	}
	return secret
}

func TestSplit_WritesTheSharesOfTheFixedForm(t *testing.T) {
	t.Parallel()
	shares := mustSplit(t, repeating(0x5A), secretOf(32), 2, 3)
	if len(shares) != 3 {
		t.Fatalf("Split gave %d shares, want 3", len(shares))
	}
	for i, share := range shares {
		if int(share.MemberIndex) != i {
			t.Errorf("share %d has member index %d", i, share.MemberIndex)
		}
		if share.MemberThreshold != 2 || share.GroupThreshold != 1 || share.GroupCount != 1 || share.GroupIndex != 0 {
			t.Errorf("share %d: member threshold %d, group %d of %d at %d; want 2, 1 of 1 at 0", i, share.MemberThreshold, share.GroupThreshold, share.GroupCount, share.GroupIndex)
		}
		if !share.Extendable {
			t.Errorf("share %d is not extendable", i)
		}
		if share.IterationExponent != 1 {
			t.Errorf("share %d has iteration exponent %d, want 1", i, share.IterationExponent)
		}
		if share.Identifier != shares[0].Identifier {
			t.Errorf("share %d has another identifier", i)
		}
		if share.Identifier >= 1<<15 {
			t.Errorf("identifier %d does not fit 15 bits", share.Identifier)
		}
		if len(share.Value) != 32 {
			t.Errorf("share %d value has %d bytes, want 32", i, len(share.Value))
		}
		if got := len(share.Words()); got != 33 {
			t.Errorf("share %d has %d words, want 33", i, got)
		}
	}
}

func TestSplit_Writes20WordsForA128BitSecret(t *testing.T) {
	t.Parallel()
	for _, share := range mustSplit(t, repeating(1), secretOf(16), 2, 3) {
		if got := len(share.Words()); got != 20 {
			t.Errorf("a share of a 16-byte secret has %d words, want 20", got)
		}
	}
}

func TestCombine_RestoresTheSecretFromAnyTwoOfThree(t *testing.T) {
	t.Parallel()
	secret := secretOf(32)
	shares := mustSplit(t, repeating(0x33), secret, 2, 3)
	for _, pair := range [][2]int{{0, 1}, {0, 2}, {1, 2}, {2, 1}} {
		got := mustCombine(t, []slip39.Share{shares[pair[0]], shares[pair[1]]})
		if !bytes.Equal(got, secret) {
			t.Errorf("shares %v restore %x, want %x", pair, got, secret)
		}
	}
}

func TestCombine_RefusesOneShareOfTwoOfThree(t *testing.T) {
	t.Parallel()
	shares := mustSplit(t, repeating(0x33), secretOf(32), 2, 3)
	if _, err := slip39.Combine(shares[:1], nil); !errors.Is(err, slip39.ErrNotEnoughShares) {
		t.Errorf("error = %v, want ErrNotEnoughShares", err)
	}
	if _, err := slip39.Combine(nil, nil); err == nil {
		t.Error("no shares restore a secret")
	}
}

func TestCombine_RefusesTheSameShareTwice(t *testing.T) {
	t.Parallel()
	shares := mustSplit(t, repeating(0x33), secretOf(32), 2, 3)
	_, err := slip39.Combine([]slip39.Share{shares[1], shares[1]}, nil)
	if !errors.Is(err, slip39.ErrDuplicateMember) {
		t.Errorf("error = %v, want ErrDuplicateMember", err)
	}
}

func TestCombine_RefusesSharesOfTwoSplits(t *testing.T) {
	t.Parallel()
	secret := secretOf(32)
	first := mustSplit(t, repeating(0x11), secret, 2, 3)
	second := mustSplit(t, repeating(0x22), secret, 2, 3)
	_, err := slip39.Combine([]slip39.Share{first[0], second[1]}, nil)
	if !errors.Is(err, slip39.ErrMixedSets) {
		t.Errorf("error = %v, want ErrMixedSets", err)
	}
}

func TestCombine_RefusesAShareWithAChangedValueByDigest(t *testing.T) {
	t.Parallel()
	shares := mustSplit(t, repeating(0x33), secretOf(32), 2, 3)
	changed := shares[1]
	changed.Value = slices.Clone(changed.Value)
	changed.Value[0] ^= 0x01
	if _, err := slip39.Combine([]slip39.Share{shares[0], changed}, nil); !errors.Is(err, slip39.ErrDigest) {
		t.Errorf("error = %v, want ErrDigest", err)
	}
}

func TestCombine_UsesThePassphrase(t *testing.T) {
	t.Parallel()
	secret := secretOf(32)
	shares := mustSplit(t, repeating(0x33), secret, 2, 3)
	got, err := slip39.Combine(shares[:2], []byte("other"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, secret) {
		t.Error("another passphrase restores the secret")
	}
}

func TestSplit_IsDeterministicForOneRandomStream(t *testing.T) {
	t.Parallel()
	first := mustSplit(t, repeating(0x77), secretOf(32), 2, 3)
	second := mustSplit(t, repeating(0x77), secretOf(32), 2, 3)
	for i := range first {
		if strings.Join(first[i].Words(), " ") != strings.Join(second[i].Words(), " ") {
			t.Errorf("share %d differs between two splits of one stream", i)
		}
	}
}

func TestSplit_DiffersForTwoRandomStreams(t *testing.T) {
	t.Parallel()
	first := mustSplit(t, repeating(0x11), secretOf(32), 2, 3)
	second := mustSplit(t, repeating(0x22), secretOf(32), 2, 3)
	if first[0].Identifier == second[0].Identifier {
		t.Error("two splits have one identifier")
	}
	if bytes.Equal(first[0].Value, second[0].Value) {
		t.Error("two splits have one value")
	}
}

func TestSplit_DoesNotLeakTheSecretInAValue(t *testing.T) {
	t.Parallel()
	for _, secret := range [][]byte{secretOf(32), make([]byte, 32), make([]byte, 16)} {
		for i, share := range mustSplit(t, repeating(0x44), secret, 2, 3) {
			if bytes.Equal(share.Value, secret) {
				t.Errorf("share %d value equals the secret %x", i, secret)
			}
			if bytes.Equal(share.Value, make([]byte, len(share.Value))) {
				t.Errorf("share %d of the secret %x is all zero", i, secret)
			}
		}
	}
}

func TestSplit_FailsWhenTheRandomSourceFails(t *testing.T) {
	t.Parallel()
	if _, err := slip39.Split(iotest.ErrReader(errors.New("no entropy")), secretOf(32), 2, 3); err == nil {
		t.Error("Split ignores a failing random source")
	}
	if _, err := slip39.Split(bytes.NewReader([]byte{1, 2, 3}), secretOf(32), 2, 3); err == nil {
		t.Error("Split ignores a short random source")
	}
}

func TestSplit_RefusesParametersOutsideTheRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		size             int
		threshold, count int
	}{
		{"secret of 15 bytes", 15, 2, 3},
		{"secret of 17 bytes", 17, 2, 3},
		{"secret of 14 bytes", 14, 2, 3},
		{"secret of 34 bytes", 34, 2, 3},
		{"empty secret", 0, 2, 3},
		{"threshold above count", 32, 4, 3},
		{"threshold zero", 32, 0, 3},
		{"count zero", 32, 1, 0},
		{"count 17", 32, 2, 17},
		{"threshold 1 of 3", 32, 1, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if shares, err := slip39.Split(repeating(1), secretOf(tt.size), tt.threshold, tt.count); err == nil {
				t.Errorf("Split gave %d shares, want an error", len(shares))
			}
		})
	}
}

func TestSplit_AcceptsOneOfOne(t *testing.T) {
	t.Parallel()
	secret := secretOf(16)
	shares := mustSplit(t, repeating(9), secret, 1, 1)
	if got := mustCombine(t, shares); !bytes.Equal(got, secret) {
		t.Errorf("Combine = %x, want %x", got, secret)
	}
}

func TestCombine_RestoresTheSecretOfAnySplit(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.SampledFrom([]int{16, 32}).Draw(t, "size")
		secret := rapid.SliceOfN(rapid.Byte(), size, size).Draw(t, "secret")
		count := rapid.IntRange(2, 16).Draw(t, "count")
		threshold := rapid.IntRange(2, count).Draw(t, "threshold")
		stream := rapid.SliceOfN(rapid.Byte(), 4096, 4096).Draw(t, "stream")
		shares, err := slip39.Split(bytes.NewReader(stream), secret, threshold, count)
		if err != nil {
			t.Fatalf("Split(%d of %d) error = %v", threshold, count, err)
		}
		picked := rapid.Permutation(shares).Draw(t, "order")[:threshold]
		got, err := slip39.Combine(picked, nil)
		if err != nil {
			t.Fatalf("Combine of %d shares error = %v", threshold, err)
		}
		if !bytes.Equal(got, secret) {
			t.Fatalf("Combine = %x, want %x", got, secret)
		}
		if _, err := slip39.Combine(picked[:threshold-1], nil); !errors.Is(err, slip39.ErrNotEnoughShares) {
			t.Fatalf("Combine of %d shares error = %v, want ErrNotEnoughShares", threshold-1, err)
		}
	})
}

func TestParseShare_ReadsTheWordsOfAnyShare(t *testing.T) {
	t.Parallel()
	list := readWordList(t)
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.SampledFrom([]int{16, 32}).Draw(t, "size")
		secret := rapid.SliceOfN(rapid.Byte(), size, size).Draw(t, "secret")
		stream := rapid.SliceOfN(rapid.Byte(), 4096, 4096).Draw(t, "stream")
		shares, err := slip39.Split(bytes.NewReader(stream), secret, 2, 3)
		if err != nil {
			t.Fatal(err)
		}
		share := rapid.SampledFrom(shares).Draw(t, "share")
		words := share.Words()
		parsed, err := slip39.ParseShare(words)
		if err != nil {
			t.Fatalf("the words of a share do not parse: %v", err)
		}
		if !bytes.Equal(parsed.Value, share.Value) || parsed.MemberIndex != share.MemberIndex || parsed.Identifier != share.Identifier {
			t.Fatal("the parsed share differs")
		}
		at := rapid.IntRange(0, len(words)-1).Draw(t, "position")
		other := rapid.SampledFrom(list).Filter(func(word string) bool { return word != words[at] }).Draw(t, "other")
		broken := slices.Clone(words)
		broken[at] = other
		if _, err := slip39.ParseShare(broken); !errors.Is(err, slip39.ErrChecksum) {
			t.Fatalf("one changed word: error = %v, want ErrChecksum", err)
		}
	})
}

func TestSplit_SeparatesTwoSplitsOfOneSecret(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		secret := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "secret")
		a := rapid.SliceOfN(rapid.Byte(), 4096, 4096).Draw(t, "first stream")
		b := rapid.SliceOfN(rapid.Byte(), 4096, 4096).Filter(func(b []byte) bool { return !bytes.Equal(a[:64], b[:64]) }).Draw(t, "second stream")
		first, err := slip39.Split(bytes.NewReader(a), secret, 2, 3)
		if err != nil {
			t.Fatal(err)
		}
		second, err := slip39.Split(bytes.NewReader(b), secret, 2, 3)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(first[0].Words()) == fmt.Sprint(second[0].Words()) {
			t.Fatal("two splits of one secret give one sheet")
		}
	})
}
