package ulid_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

func TestShapeAndAlphabet(t *testing.T) {
	id, err := ulid.New(time.Now())
	if err != nil {
		t.Fatal(err)
	}
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
	// over UUID, so it is worth a test rather than a comment.
	base := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	earlier, err := ulid.New(base)
	if err != nil {
		t.Fatal(err)
	}
	later, err := ulid.New(base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !(earlier < later) {
		t.Errorf("%s should sort before %s", earlier, later)
	}
}

func TestDistinctWithinTheSameMillisecond(t *testing.T) {
	at := time.Now()
	seen := map[string]bool{}
	for range 1000 {
		id, err := ulid.New(at)
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate within one millisecond: %s", id)
		}
		seen[id] = true
	}
}

func TestValidRejectsNearMisses(t *testing.T) {
	good, err := ulid.New(time.Now())
	if err != nil {
		t.Fatal(err)
	}
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
