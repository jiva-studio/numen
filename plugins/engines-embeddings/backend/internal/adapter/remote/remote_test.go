package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func encodeJSON(tb testing.TB, w http.ResponseWriter, value any) {
	tb.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		tb.Errorf("encode the response: %v", err)
	}
}

func newService(t *testing.T, mode *string) *httptest.Server {
	t.Helper()
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch *mode {
		case "status500":
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("server exploded")); err != nil {
				t.Error(err)
			}
		case "badjson":
			if _, err := w.Write([]byte("{invalid json")); err != nil {
				t.Error(err)
			}
		case "emptydata":
			encodeJSON(t, w, map[string]any{"data": []any{}})
		case "emptyvec":
			encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{}, "index": 0}}})
		case "singlefloat":
			encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.42}, "index": 0}}})
		default:
			if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer secret" {
				t.Errorf("headers: %v", r.Header)
			}
			encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.1, 0.2, 0.3, 0.4, 0.5}, "index": 0}}})
		}
	}))
	t.Cleanup(service.Close)
	return service
}

func TestAServiceAnswersOneVectorForEachText(t *testing.T) {
	mode := "ok"
	service := newService(t, &mode)

	vectors, err := FetchVectors(t.Context(), "secret", service.URL, "remote-test", []string{"one", "two"})

	if err != nil || len(vectors) != 2 || len(vectors[0]) != 5 {
		t.Fatalf("FetchVectors = %v, %v", vectors, err)
	}
}

func TestAFailedTextNamesItsPositionAndTheStatus(t *testing.T) {
	mode := "status500"
	service := newService(t, &mode)

	_, err := FetchVectors(t.Context(), "k", service.URL, "m", []string{"hello"})

	if err == nil || !strings.Contains(err.Error(), "service text 0: status 500: server exploded") {
		t.Fatalf("status500 err = %v", err)
	}
}

func TestAnAnswerWithoutAVectorIsAnError(t *testing.T) {
	mode := "badjson"
	service := newService(t, &mode)
	if _, err := fetchVector(t.Context(), "k", service.URL, "m", "t"); err == nil {
		t.Fatal("expected error on invalid json")
	}

	mode = "emptydata"
	if _, err := fetchVector(t.Context(), "k", service.URL, "m", "t"); err == nil || !strings.Contains(err.Error(), "no embedding returned") {
		t.Fatalf("emptydata err = %v", err)
	}

	mode = "emptyvec"
	if _, err := fetchVector(t.Context(), "k", service.URL, "m", "t"); err == nil || !strings.Contains(err.Error(), "no embedding returned") {
		t.Fatalf("emptyvec err = %v", err)
	}

	mode = "singlefloat"
	if vec, err := fetchVector(t.Context(), "k", service.URL, "m", "t"); err != nil || len(vec) != 1 || vec[0] != 0.42 {
		t.Fatalf("singlefloat = %v, %v", vec, err)
	}
}

func TestAnAddressThatCannotBeReachedIsAnError(t *testing.T) {
	if _, err := fetchVector(t.Context(), "k", "://bad-url", "m", "t"); err == nil {
		t.Fatal("expected error on bad URL")
	}

	if _, err := fetchVector(t.Context(), "k", "http://127.0.0.1:1", "m", "t"); err == nil {
		t.Fatal("expected error on unreachable host")
	}
}
