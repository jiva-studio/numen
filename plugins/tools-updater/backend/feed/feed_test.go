package feed

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func serve(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server.URL
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingReadCloser struct {
	reader   io.Reader
	readErr  error
	closeErr error
}

func (f *failingReadCloser) Read(p []byte) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return f.reader.Read(p)
}

func (f *failingReadCloser) Close() error {
	return f.closeErr
}

func TestNewSetsTheRequestTimeout(t *testing.T) {
	if got := New().http.Timeout; got != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", got)
	}
}

func TestFetchReadsTheManifestTheEndpointPublishes(t *testing.T) {
	endpoint := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			http.Error(w, "wrong accept", http.StatusBadRequest)
			return
		}
		if _, err := w.Write([]byte(`{"version":"2026.10.1","build":950,"channel":"stable"}`)); err != nil {
			t.Error(err)
		}
	})

	manifest, err := New().Fetch(t.Context(), endpoint)

	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "2026.10.1" || manifest.Build != 950 || manifest.Channel != "stable" {
		t.Errorf("manifest = %+v", manifest)
	}
}

func TestFetchBoundsTheManifestBody(t *testing.T) {
	prefix := `{"version":"2026.10.1","build":950,"channel":"stable"`
	exactBody := prefix + strings.Repeat(" ", (1<<20)-len(prefix)-1) + "}"
	exactEndpoint := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(exactBody)); err != nil {
			t.Error(err)
		}
	})
	if _, err := New().Fetch(t.Context(), exactEndpoint); err != nil {
		t.Fatalf("Fetch at body limit failed: %v", err)
	}

	overBody := prefix + strings.Repeat(" ", (1<<20)-len(prefix)) + "}"
	overEndpoint := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(overBody)); err != nil {
			t.Error(err)
		}
	})
	if _, err := New().Fetch(t.Context(), overEndpoint); err == nil {
		t.Error("Fetch beyond body limit succeeded")
	}
}

func TestFetchFailsOnAnAnswerOtherThan200(t *testing.T) {
	endpoint := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})

	_, err := New().Fetch(t.Context(), endpoint)

	if !errors.Is(err, ErrStatus) {
		t.Errorf("error = %v, want a status error", err)
	}
}

func TestFetchFailsOnABodyThatIsNoManifest(t *testing.T) {
	endpoint := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("<html>")); err != nil {
			t.Error(err)
		}
	})

	if _, err := New().Fetch(t.Context(), endpoint); err == nil {
		t.Error("a page that is no manifest was read")
	}
}

func TestFetchFailsWhereNothingListens(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()

	_, err := New().Fetch(t.Context(), endpoint)
	if !errors.As(err, new(*url.Error)) {
		t.Errorf("error = %v, want url.Error", err)
	}
}

func TestFetchReportsReadAndCloseFailures(t *testing.T) {
	errRead := errors.New("read failed")
	readClient := &Client{
		http: &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       &failingReadCloser{readErr: errRead},
				}, nil
			}),
		},
	}
	if _, err := readClient.Fetch(t.Context(), "https://dl.numen.md/latest/latest.json"); !errors.Is(err, errRead) {
		t.Errorf("read error = %v, want %v", err, errRead)
	}

	errClose := errors.New("close failed")
	closeClient := &Client{
		http: &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: &failingReadCloser{
						reader:   strings.NewReader(`{"version":"2026.10.1","build":950}`),
						closeErr: errClose,
					},
				}, nil
			}),
		},
	}
	if _, err := closeClient.Fetch(t.Context(), "https://dl.numen.md/latest/latest.json"); !errors.Is(err, errClose) {
		t.Errorf("close error = %v, want %v", err, errClose)
	}
}

func TestFetchFailsOnAnEndpointThatIsNoWebAddress(t *testing.T) {
	if err := CheckEndpoint("https://dl.numen.md/latest/latest.json"); err != nil {
		t.Fatalf("CheckEndpoint(https) = %v", err)
	}
	for _, endpoint := range []string{"", "latest.json", "file:///etc/passwd", "ftp://example.org/latest.json", "http://"} {
		_, err := New().Fetch(t.Context(), endpoint)

		if !errors.Is(err, ErrNotWebAddress) {
			t.Errorf("Fetch(%q) error = %v, want a refusal of the address", endpoint, err)
		}
	}
}
