package server

import (
	"net/http"
	"testing"
)

func TestHasBearerToken(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		header http.Header
		token  string
		want   bool
	}{
		{"matching bearer", http.Header{"Authorization": {"Bearer orchard-4417"}}, "orchard-4417", true},
		{"different token", http.Header{"Authorization": {"Bearer orchard-4418"}}, "orchard-4417", false},
		{"prefix of the token", http.Header{"Authorization": {"Bearer orchard"}}, "orchard-4417", false},
		{"token without the scheme", http.Header{"Authorization": {"orchard-4417"}}, "orchard-4417", false},
		{"other scheme", http.Header{"Authorization": {"Basic orchard-4417"}}, "orchard-4417", false},
		{"lowercase scheme", http.Header{"Authorization": {"bearer orchard-4417"}}, "orchard-4417", false},
		{"no header", http.Header{}, "orchard-4417", false},
		{"empty token and empty bearer", http.Header{"Authorization": {"Bearer "}}, "", false},
		{"empty token and no header", http.Header{}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasBearerToken(tc.header, tc.token); got != tc.want {
				t.Errorf("HasBearerToken(%v, %q) = %v, want %v", tc.header, tc.token, got, tc.want)
			}
		})
	}
}
