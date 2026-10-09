package download_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/download"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func createHTML(targetBytes int) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html><html><head><title>Benchmark Article</title></head><body><article><h1>Benchmark Article</h1>")
	paragraph := "<p>This is a paragraph of text designed to simulate readable content for benchmarking the download adapter.</p>\n"
	for buf.Len() < targetBytes-100 {
		buf.WriteString(paragraph)
	}
	buf.WriteString("</article></body></html>")
	return buf.Bytes()
}

func BenchmarkDownloadText(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"4KB", 4 * 1024},
		{"64KB", 64 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			payload := createHTML(tc.size)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write(payload)
			}))
			defer server.Close()

			downloader, err := download.New(b.Context(), download.Config{})
			if err != nil {
				b.Fatal(err)
			}
			url := address(b, server.URL+"/benchmark")

			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for b.Loop() {
				article, err := downloader.Text(b.Context(), url, port.PreferredCaptions{})
				if err != nil {
					b.Fatal(err)
				}
				if article.Title == "" {
					b.Fatal("missing title")
				}
			}
		})
	}
}
