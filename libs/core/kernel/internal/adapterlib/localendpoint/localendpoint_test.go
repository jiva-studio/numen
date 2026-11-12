package localendpoint_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/localendpoint"
)

func TestNewTokenDiffersEveryCall(t *testing.T) {
	first, err := localendpoint.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := localendpoint.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || len(first) != 64 {
		t.Fatalf("tokens %q and %q", first, second)
	}
}

func TestRequireBearerTokenAnswersOnlyTheHolder(t *testing.T) {
	served := 0
	handler := localendpoint.RequireBearerToken("s3cret", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served++
		w.WriteHeader(http.StatusNoContent)
	}))
	cases := []struct {
		name, header string
		want         int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer other", http.StatusUnauthorized},
		{"prefix of the token", "Bearer s3cre", http.StatusUnauthorized},
		{"other scheme", "Basic s3cret", http.StatusUnauthorized},
		{"bare token", "s3cret", http.StatusUnauthorized},
		{"the token", "Bearer s3cret", http.StatusNoContent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/numen.v1.NoteService/GetNote", http.NoBody)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("status %d, want %d", rec.Code, c.want)
			}
		})
	}
	if served != 1 {
		t.Fatalf("the handler ran %d times, want once", served)
	}
}

func TestEmptyTokenMatchesNothing(t *testing.T) {
	handler := localendpoint.RequireBearerToken("", http.NotFoundHandler())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", rec.Code)
	}
	if localendpoint.IsTokenEqual("", "") {
		t.Fatal("two empty tokens are equal")
	}
}
