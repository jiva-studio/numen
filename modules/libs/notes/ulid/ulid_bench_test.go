package ulid_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/ulid"
)

func BenchmarkNew(b *testing.B) {
	now := time.Now()
	b.ReportAllocs()
	for b.Loop() {
		ulid.New(now)
	}
}

func BenchmarkValid(b *testing.B) {
	id := ulid.New(time.Now())
	b.ReportAllocs()
	for b.Loop() {
		if !ulid.Valid(id) {
			b.Fatal("invalid")
		}
	}
}
