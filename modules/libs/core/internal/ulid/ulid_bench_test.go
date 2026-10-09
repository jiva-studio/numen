package ulid_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
)

func BenchmarkNew(b *testing.B) {
	now := time.Now()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ulid.New(now); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValid(b *testing.B) {
	id, err := ulid.New(time.Now())
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
