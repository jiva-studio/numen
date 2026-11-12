package format_test

import (
	"crypto/rand"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
)

func BenchmarkCreateID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		format.CreateID(readTestEntropy)
	}
}

func BenchmarkIsValid(b *testing.B) {
	id := format.CreateID(readTestEntropy)
	b.ReportAllocs()
	for b.Loop() {
		if !format.IsValidID(id) {
			b.Fatal("invalid")
		}
	}
}

func readTestEntropy(b []byte) {
	rand.Read(b)
}
