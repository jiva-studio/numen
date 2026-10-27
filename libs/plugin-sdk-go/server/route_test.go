package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// getRoute asks handler for path with the bearer token given, empty for none.
func getRoute(t *testing.T, handler http.Handler, method, path, token string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorded := httptest.NewRecorder()
	handler.ServeHTTP(recorded, req)
	body, _ := io.ReadAll(recorded.Body)
	return recorded.Code, string(body)
}

func TestRegisterRouteAnswersTheHostToken(t *testing.T) {
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "route-token"}, func(s *Server) error {
		s.RegisterRoute("pages/{page}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "page "+r.PathValue("page"))
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
