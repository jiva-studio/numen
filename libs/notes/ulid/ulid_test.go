package ulid_test

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/ulid"
)

func init() {
	ulid.DefaultEntropy = rand.Reader
}

func TestShapeAndAlphabet(t *testing.T) {
	id := ulid.New(time.Now())
	if len(id) != 26 {
		t.Fatalf("len = %d, want 26 (%s)", len(id), id)
	}
	if !ulid.Valid(id) {
		t.Fatalf("%s did not validate", id)
	}
	// The confusable glyphs are the point of Crockford base32: an identifier
	// read off a screen and retyped must not become a different one.
	if strings.ContainsAny(id, "ILOU") {
		t.Errorf("%s contains a confusable character", id)
	}
}

func TestOrderMatchesTime(t *testing.T) {
	// Lexicographic order equalling chronological order is why ULID was chosen
	// over UUID, so it is worth a test and not a comment.
	base := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	earlier := ulid.New(base)
	later := ulid.New(base.Add(time.Second))
	if !(earlier < later) {
		t.Errorf("%s should sort before %s", earlier, later)
	}
}

func TestDistinctWithinTheSameMillisecond(t *testing.T) {
	stamp := time.Now()
	observed := map[string]bool{}
	for range 1000 {
		id := ulid.New(stamp)
		if observed[id] {
			t.Fatalf("duplicate within one millisecond: %s", id)
		}
		observed[id] = true
	}
}

func TestValidRejectsNearMisses(t *testing.T) {
	good := ulid.New(time.Now())
	for _, bad := range []string{
		"", good[:25], good + "0",
		strings.Replace(good, string(good[10]), "I", 1),
		strings.ToLower(good),
	} {
		if ulid.Valid(bad) {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestValidRejectsBytesOutsideTheAlphabet(t *testing.T) {
	const base = "0123456789ABCDEFGHJKMNPQRS"
	for name, s := range map[string]string{
		"lower case": base[:25] + "a",
		"letter I":   base[:25] + "I",
		"letter U":   base[:25] + "U",
		"non-ascii":  base[:24] + "É",
		"high byte":  base[:25] + "\xff",
		"nul byte":   base[:25] + "\x00",
		"too long":   base + "T",
		"too short":  base[:25],
	} {
		if ulid.Valid(s) {
			t.Errorf("%s: %q validated", name, s)
		}
	}
}

// The first ten characters are the instant: two zero bits and then the 48 bits
// of milliseconds since the epoch, most significant first.
func TestTheFirstTenCharactersAreTheMillisecond(t *testing.T) {
	for _, c := range []struct {
		name string
		ms   int64
		want string
	}{
		{"the epoch", 0, "0000000000"},
		{"one millisecond after it", 1, "0000000001"},
		{"the example the format is published with", 1469918176385, "01ARYZ6S41"},
		{"every byte of the instant distinct", 0xFEDCBA987654, "7YVJX9GXJM"},
		{"the last millisecond 48 bits hold", 1<<48 - 1, "7ZZZZZZZZZ"},
	} {
		t.Run(c.name, func(t *testing.T) {
			for range 16 {
				id := ulid.New(time.UnixMilli(c.ms))
				if got := id[:10]; got != c.want {
					t.Fatalf("%s starts %s, want %s", id, got, c.want)
				}
			}
		})
	}
}

// The last 80 bits are random: across identifiers made in one millisecond,
// every one of them is set in some and clear in others.
func TestEveryRandomBitVaries(t *testing.T) {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	stamp := time.UnixMilli(1469918176385)
	var isSet, isClear [130]bool
	for range 64 {
		id := ulid.New(stamp)
		for bit := 50; bit < 130; bit++ {
			value := strings.IndexByte(alphabet, id[bit/5])
			if value>>(4-bit%5)&1 == 1 {
				isSet[bit] = true
			} else {
				isClear[bit] = true
			}
		}
	}
	for bit := 50; bit < 130; bit++ {
		if !isSet[bit] || !isClear[bit] {
			t.Errorf("random bit %d never changed across 64 identifiers", bit-50)
		}
	}
}
