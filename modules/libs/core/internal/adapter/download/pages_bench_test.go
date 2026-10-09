package download

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const benchPageHTML = `<!doctype html><html><head><title>Entropy</title></head><body>
<article><h1>Entropy</h1><p>A measure of how many ways the parts of a thing
can be arranged, and it grows.</p></article></body></html>`

func BenchmarkPagesText_Sequential(b *testing.B) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(benchPageHTML))
	}))
	b.Cleanup(site.Close)

	p := newPages()
	at := domain.URL(site.URL + "/entropy")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := p.Text(b.Context(), at, port.PreferredCaptions{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPagesText_Concurrent(b *testing.B) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(benchPageHTML))
	}))
	b.Cleanup(site.Close)

	p := newPages()
	at := domain.URL(site.URL + "/entropy")

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := p.Text(b.Context(), at, port.PreferredCaptions{}); err != nil {
				b.Fatal(err)
			}
		}
	})
}
