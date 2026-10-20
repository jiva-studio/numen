package cardid_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/cardid"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cardid.New()
	}
}

func BenchmarkIsValid(b *testing.B) {
	id := cardid.New()
	b.ReportAllocs()
	for b.Loop() {
		if !cardid.IsValid(id) {
			b.Fatal("invalid")
		}
	}
}
