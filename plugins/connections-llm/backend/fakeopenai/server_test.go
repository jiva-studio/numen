package fakeopenai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func post(t *testing.T, handler http.Handler, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestTheServerRejectsARequestWithoutTheKey(t *testing.T) {
	fake := New(Config{APIKey: "k"})
	body := `{"model":"m","messages":[{"role":"user","content":"hi"}]}`
	if got := post(t, fake, "/v1/chat/completions", "", body).Code; got != http.StatusUnauthorized {
		t.Errorf("no key: status %d", got)
	}
	if got := post(t, fake, "/v1/chat/completions", "k", body).Code; got != http.StatusOK {
		t.Errorf("right key: status %d", got)
	}
}

func TestTheServerAnswersAVectorOfTheWidthItWasGiven(t *testing.T) {
	fake := New(Config{Dimensions: 5})
	recorder := post(t, fake, "/embeddings", "", `{"model":"m","input":["a","b"]}`)
	var answer struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &answer); err != nil {
		t.Fatal(err)
	}
	if len(answer.Data) != 2 || len(answer.Data[0].Embedding) != 5 {
		t.Errorf("answer = %s", recorder.Body.String())
	}
}

func TestTheServerRefusesARequestWithoutAModel(t *testing.T) {
	fake := New(Config{})
	if got := post(t, fake, "/embeddings", "", `{"input":["a"]}`).Code; got != http.StatusBadRequest {
		t.Errorf("status %d", got)
	}
}

func TestAFailedPathAnswersItsStatusWithTheRetryAfter(t *testing.T) {
	fake := New(Config{})
	fake.Fail("/embeddings", http.StatusTooManyRequests, "7")
	recorder := post(t, fake, "/embeddings", "", `{"model":"m","input":["a"]}`)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "7" {
		t.Errorf("status %d, Retry-After %q", recorder.Code, recorder.Header().Get("Retry-After"))
	}
	fake.Fail("/embeddings", 0, "")
	if got := post(t, fake, "/embeddings", "", `{"model":"m","input":["a"]}`).Code; got != http.StatusOK {
		t.Errorf("after clearing: status %d", got)
	}
}

func TestADelayedRequestIsCountedAsAbortedWhenTheClientDisconnects(t *testing.T) {
	fake := New(Config{})
	fake.Delay("/models", time.Minute)
	ts := httptest.NewServer(fake)
	defer ts.Close()
	client := &http.Client{Timeout: 50 * time.Millisecond}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err == nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("want a timeout")
	}
	select {
	case <-fake.Aborted():
	case <-time.After(5 * time.Second):
		t.Fatal("the abort was not seen")
	}
}
