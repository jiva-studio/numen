package openai_test

import (
	"net/http"
	"testing"
)

func BenchmarkClientEmbed_Sequential(b *testing.B) {
	s := server(b, func(w http.ResponseWriter, r *http.Request) {
		answer(w, read(b, r), 4)
	})
	c := client(b, s.URL, 4)
	texts := []string{"first text for embedding", "second text for embedding"}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := c.Embed(b.Context(), texts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClientEmbed_Concurrent(b *testing.B) {
	s := server(b, func(w http.ResponseWriter, r *http.Request) {
		answer(w, read(b, r), 4)
	})
	c := client(b, s.URL, 4)
	texts := []string{"first text for embedding", "second text for embedding"}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := c.Embed(b.Context(), texts); err != nil {
				b.Fatal(err)
			}
		}
	})
}
