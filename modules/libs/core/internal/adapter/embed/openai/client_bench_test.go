package openai_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
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

func benchClient(tb testing.TB, baseURL string, batchChars int) *openai.Client {
	tb.Helper()
	tb.Setenv(embed.KeyEnvVar, "test-key")
	cfg := embed.Defaults()
	cfg.Model.Dimensions = 4
	cfg, service := setServiceModel(tb, cfg, func(at *embed.ServiceModel) {
		at.BaseURL, at.Name = baseURL, "test-embed"
		if batchChars > 0 {
			at.BatchCharacters = batchChars
		}
	})
	c, err := openai.New(cfg.GetStoredModel(), service)
	if err != nil {
		tb.Fatal(err)
	}
	c.Delay = 0
	return c
}

func BenchmarkEmbed(b *testing.B) {
	sample := strings.Repeat("The quick brown fox jumps over the lazy dog. A paragraph with multiple sentences for embedding. ", 5)
	texts256 := make([]string, 256)
	for i := range texts256 {
		texts256[i] = fmt.Sprintf("%s chunk %d", sample, i)
	}

	for _, tc := range []struct {
		name       string
		batchChars int
	}{
		{name: "texts=256/batchChars=8000", batchChars: 8000},
		{name: "texts=256/batchChars=32000", batchChars: 32000},
	} {
		b.Run(tc.name, func(b *testing.B) {
			s := server(b, func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Millisecond)
				answer(w, read(b, r), 4)
			})
			c := benchClient(b, s.URL, tc.batchChars)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := c.Embed(b.Context(), texts256); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
