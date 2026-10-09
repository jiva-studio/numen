package cardid_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := cardid.New(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValid(b *testing.B) {
	id, err := cardid.New()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if !cardid.Valid(id) {
			b.Fatal("invalid")
		}
	}
}
