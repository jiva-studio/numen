package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type echoRequest struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

type echoResponse struct {
	Text string `json:"text"`
}

func createAPIHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "api-token"}, func(s *Server) error {
		api := s.NewAPI("org.example.echo")
		Handle(api, "Echo", func(_ context.Context, req echoRequest) (echoResponse, error) {
			if req.Text == "missing" {
				return echoResponse{}, NewAPIError(CodeNotFound, "no such text")
			}
			if req.Text == "broken" {
				return echoResponse{}, errors.New("disk failure")
			}
			return echoResponse{Text: strings.Repeat(req.Text, max(req.Count, 1))}, nil
		})
		Stream(api, "Count", func(_ context.Context, req echoRequest, send func(echoResponse) error) error {
			for range req.Count {
				if err := send(echoResponse{Text: req.Text}); err != nil {
					return err
				}
			}
			if req.Text == "fail" {
				return NewAPIError(CodeUnavailable, "stopped")
			}
			return nil
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func postAPI(t *testing.T, handler http.Handler, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorded := httptest.NewRecorder()
	handler.ServeHTTP(recorded, req)
	return recorded
}

func TestHandle_AnswersJSONWithTheHostToken(t *testing.T) {
	handler := createAPIHandler(t)

	res := postAPI(t, handler, "/org.example.echo/Echo", `{"text":"ab","count":2,"unknown":true}`, "api-token")

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", res.Code, res.Body)
	}
	var got echoResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil || got.Text != "abab" {
		t.Errorf("response = %q, %v", res.Body, err)
	}
}

func TestHandle_AcceptsAnEmptyBody(t *testing.T) {
	res := postAPI(t, createAPIHandler(t), "/org.example.echo/Echo", "", "api-token")

	if res.Code != http.StatusOK {
		t.Errorf("status = %d: %s", res.Code, res.Body)
	}
}

func TestHandle_RejectsARequestWithoutTheToken(t *testing.T) {
	handler := createAPIHandler(t)

	if res := postAPI(t, handler, "/org.example.echo/Echo", "{}", ""); res.Code != http.StatusUnauthorized {
		t.Errorf("without a token: %d", res.Code)
	}
	if res := postAPI(t, handler, "/org.example.echo/Echo", "{}", "other"); res.Code != http.StatusUnauthorized {
		t.Errorf("with another token: %d", res.Code)
	}
}

func TestHandle_MapsFailuresToCodes(t *testing.T) {
	handler := createAPIHandler(t)
	for _, c := range []struct {
		body   string
		status int
		code   ErrorCode
	}{
		{`{"text":"missing"}`, http.StatusNotFound, CodeNotFound},
		{`{"text":"broken"}`, http.StatusInternalServerError, CodeInternal},
		{`{"text":`, http.StatusBadRequest, CodeInvalidArgument},
	} {
		res := postAPI(t, handler, "/org.example.echo/Echo", c.body, "api-token")
		var failure APIError
		if err := json.Unmarshal(res.Body.Bytes(), &failure); err != nil {
			t.Fatalf("%s: %v", c.body, err)
		}
		if res.Code != c.status || failure.Code != c.code {
			t.Errorf("%s: status %d code %q, want %d %q", c.body, res.Code, failure.Code, c.status, c.code)
		}
	}
}

func readEvents(t *testing.T, body string) []string {
	t.Helper()
	var events []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			events = append(events, line)
		}
	}
	return events
}

func TestStream_SendsResponsesAndEnds(t *testing.T) {
	res := postAPI(t, createAPIHandler(t), "/org.example.echo/Count", `{"text":"x","count":2}`, "api-token")

	want := []string{`data: {"text":"x"}`, `data: {"text":"x"}`, "event: end", "data: {}"}
	got := readEvents(t, res.Body.String())
	if len(got) != len(want) {
		t.Fatalf("events = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %q, want %q", i, got[i], want[i])
		}
	}
	if contentType := res.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Errorf("Content-Type = %q", contentType)
	}
}

func TestStream_EndsWithAnErrorEvent(t *testing.T) {
	res := postAPI(t, createAPIHandler(t), "/org.example.echo/Count", `{"text":"fail","count":1}`, "api-token")

	events := readEvents(t, res.Body.String())
	last := events[len(events)-2:]
	if last[0] != "event: error" || last[1] != `data: {"code":"unavailable","message":"stopped"}` {
		t.Errorf("events = %q", events)
	}
}

func TestNewAPI_ReportsAnInvalidNameWhenServing(t *testing.T) {
	_, err := NewLinkedHandler(map[string]string{tokenVar: "api-token"}, func(s *Server) error {
		Handle(s.NewAPI("a/b"), "Echo", func(context.Context, echoRequest) (echoResponse, error) {
			return echoResponse{}, nil
		})
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "invalid API service name") {
		t.Fatalf("a service name with a slash gave %v", err)
	}
}

func TestHandle_ExposesRequestHeaders(t *testing.T) {
	handler, err := NewLinkedHandler(map[string]string{tokenVar: "api-token"}, func(s *Server) error {
		Handle(s.NewAPI("org.example.echo"), "Vault", func(ctx context.Context, _ echoRequest) (echoResponse, error) {
			return echoResponse{Text: GetHeader(ctx).Get("Numen-Vault")}, nil
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/org.example.echo/Vault", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer api-token")
	req.Header.Set("Numen-Vault", "vault-7")
	recorded := httptest.NewRecorder()

	handler.ServeHTTP(recorded, req)

	var got echoResponse
	if err := json.Unmarshal(recorded.Body.Bytes(), &got); err != nil || got.Text != "vault-7" {
		t.Errorf("response = %q, %v", recorded.Body, err)
	}
}
