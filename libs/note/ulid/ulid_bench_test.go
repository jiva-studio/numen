package ulid_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/ulid"
)

func BenchmarkNew(b *testing.B) {
	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	generator := ulid.NewGenerator(rand.Reader)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := generator.NewID(now); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValid(b *testing.B) {
	id, err := ulid.NewGenerator(rand.Reader).NewID(time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if !ulid.Valid(id) {
			b.Fatal("invalid")
		}
	}
}
