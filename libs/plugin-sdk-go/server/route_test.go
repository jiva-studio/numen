package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// getRoute requests path from handler with the given bearer token, or none when empty.
func getRoute(t *testing.T, handler http.Handler, method, path, token string) (int, string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorded := httptest.NewRecorder()
	handler.ServeHTTP(recorded, req)
	body, err := io.ReadAll(recorded.Body)
	if err != nil {
		t.Fatal(err)
	}
	return recorded.Code, string(body)
}

func TestRegisterRouteAnswersTheHostToken(t *testing.T) {
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "route-token"}, func(s *Server) error {
		s.RegisterRoute("pages/{page}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, "page "+r.PathValue("page")); err != nil {
				t.Error(err)
			}
		}))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if code, body := getRoute(t, handler, http.MethodGet, "/plugin/pages/3", "route-token"); code != http.StatusOK || body != "page 3" {
		t.Errorf("with the token: %d %q", code, body)
	}
	if code, _ := getRoute(t, handler, http.MethodGet, "/plugin/pages/3", "another-token"); code != http.StatusUnauthorized {
		t.Errorf("with another token: %d", code)
	}
	if code, _ := getRoute(t, handler, http.MethodGet, "/plugin/pages/3", ""); code != http.StatusUnauthorized {
		t.Errorf("without a token: %d", code)
	}
	if code, _ := getRoute(t, handler, http.MethodPost, "/plugin/pages/3", "route-token"); code != http.StatusMethodNotAllowed {
		t.Errorf("a POST: %d", code)
	}
}
