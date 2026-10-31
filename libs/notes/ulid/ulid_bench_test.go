package ulid_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/ulid"
)

func BenchmarkNew(b *testing.B) {
	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	b.ReportAllocs()
	for b.Loop() {
		ulid.New(now)
	}
}

func BenchmarkValid(b *testing.B) {
	id := ulid.New(time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC))
	b.ReportAllocs()
	for b.Loop() {
		if !ulid.Valid(id) {
			b.Fatal("invalid")
		}
	}
}
