package service_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

const mountToken = "mount-token"

func newMountedHandler(t *testing.T, notes map[string]string) http.Handler {
	t.Helper()
	env := vaulttest.New(t, notes)
	backend := service.New(service.Config{Vault: env.Host, Store: env.Store})
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": mountToken}, backend.Mount)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func postMethod(handler http.Handler, method, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/"+service.ServiceName+"/"+method, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorded := httptest.NewRecorder()
	handler.ServeHTTP(recorded, req)
	return recorded
}

func TestTheServiceAnswersAMethodAsJSON(t *testing.T) {
	handler := newMountedHandler(t, map[string]string{"Animals.md": threeCards})

	recorded := postMethod(handler, "ReadDeck", `{"path":"Animals.md"}`, mountToken)

	if recorded.Code != http.StatusOK {
		t.Fatalf("status %d: %s", recorded.Code, recorded.Body)
	}
	var response api.ReadDeckResponse
	if err := json.Unmarshal(recorded.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Deck == nil || len(response.Deck.Cards) != 3 {
		t.Fatalf("response %s", recorded.Body)
	}
	if response.At == nil || response.At.Mtime == 0 {
		t.Errorf("the response names no version of the file: %s", recorded.Body)
	}
}

func TestTheServiceAnswersACollectionAsAnArray(t *testing.T) {
	handler := newMountedHandler(t, map[string]string{})

	recorded := postMethod(handler, "ListPresets", `{}`, mountToken)

	if !strings.Contains(recorded.Body.String(), `"presets":[]`) {
		t.Errorf("response %s", recorded.Body)
	}
}

func TestTheServiceRejectsARequestWithoutTheHostToken(t *testing.T) {
	handler := newMountedHandler(t, map[string]string{})

	if recorded := postMethod(handler, "ListPresets", `{}`, ""); recorded.Code == http.StatusOK {
		t.Errorf("status %d without a token", recorded.Code)
	}
}

func TestTheServiceAnswersAFailureWithItsCode(t *testing.T) {
	handler := newMountedHandler(t, map[string]string{})

	recorded := postMethod(handler, "StartSession", `{"vault":"nothing"}`, mountToken)

	if recorded.Code != http.StatusNotFound || !strings.Contains(recorded.Body.String(), `"code":"not_found"`) {
		t.Errorf("status %d: %s", recorded.Code, recorded.Body)
	}
}
