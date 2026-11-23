package fakeopenai

import (
	"context"
	"encoding/json"
	"errors"
	"math"
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

func TestTheServerAnswersARerankWithOneScoreForEachDocumentInDescendingOrder(t *testing.T) {
	fake := New(Config{})
	documents := []string{"Tide tables", "Ferry times", "Harbor map", "Lighthouse hours"}
	body, err := json.Marshal(map[string]any{"model": "m", "query": "when does the ferry leave?", "documents": documents, "top_n": len(documents)})
	if err != nil {
		t.Fatal(err)
	}
	recorder := post(t, fake, "/v1/rerank", "", string(body))
	var answer struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float32 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &answer); err != nil || recorder.Code != http.StatusOK {
		t.Fatalf("status %d, answer %s, err %v", recorder.Code, recorder.Body.String(), err)
	}
	if len(answer.Results) != len(documents) {
		t.Fatalf("answer = %s, want one result for each document", recorder.Body.String())
	}
	seen := map[int]bool{}
	for i, one := range answer.Results {
		if one.Index < 0 || one.Index >= len(documents) || seen[one.Index] {
			t.Fatalf("answer = %s: index %d is out of range or repeated", recorder.Body.String(), one.Index)
		}
		seen[one.Index] = true
		if want := fake.Score("when does the ferry leave?", documents[one.Index]); one.RelevanceScore != want {
			t.Errorf("score of document %d = %v, want %v", one.Index, one.RelevanceScore, want)
		}
		if i > 0 && one.RelevanceScore > answer.Results[i-1].RelevanceScore {
			t.Errorf("answer = %s, want the results in descending score order", recorder.Body.String())
		}
	}
}

func TestTheScoreOfAQueryAndADocumentIsTheSameEachTime(t *testing.T) {
	fake := New(Config{})
	if fake.Score("ferry", "Ferry times") != New(Config{}).Score("ferry", "Ferry times") {
		t.Error("two servers score the same pair differently")
	}
	if fake.Score("ferry", "Ferry times") == fake.Score("ferry", "Tide tables") {
		t.Error("two documents got the same score")
	}
}

func TestAFailedRerankAnswersItsStatus(t *testing.T) {
	fake := New(Config{})
	fake.Fail("/rerank", http.StatusUnauthorized, "")
	if got := post(t, fake, "/rerank", "", `{"model":"m","query":"q","documents":["a"],"top_n":1}`).Code; got != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", got)
	}
}

func TestAPathWithNoRouteAnswersNotFound(t *testing.T) {
	fake := New(Config{})
	if got := post(t, fake, "/v1/classify", "", `{"model":"m"}`).Code; got != http.StatusNotFound {
		t.Errorf("status %d, want 404", got)
	}
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("unreadable") }

func TestVectorMathAndDefaultDimensions(t *testing.T) {
	if DefaultDimensions != 8 {
		t.Fatalf("DefaultDimensions = %d, want 8", DefaultDimensions)
	}
	for _, dims := range []int{0, -1} {
		srv := New(Config{Dimensions: dims})
		if got := len(srv.BuildVector("ferry")); got != 8 {
			t.Errorf("New(Dimensions=%d).Vector len = %d, want 8", dims, got)
		}
		if got := cap(srv.Aborted()); got != 64 {
			t.Errorf("cap(Aborted) = %d, want 64", got)
		}
	}

	vec := New(Config{Dimensions: 28}).BuildVector("ferry")
	if len(vec) != 28 {
		t.Fatalf("len(vec) = %d, want 28", len(vec))
	}
	var sum float64
	for _, x := range vec {
		sum += float64(x) * float64(x)
	}
	if math.Abs(sum-1.0) > 1e-5 {
		t.Errorf("squared norm = %v, want 1.0", sum)
	}
	if !(vec[0] > 0 && vec[1] < 0 && vec[25] < 0 && vec[26] > 0 && vec[27] < 0) {
		t.Errorf("unexpected signs in vec: v0=%v v1=%v v25=%v v26=%v v27=%v", vec[0], vec[1], vec[25], vec[26], vec[27])
	}
	raw0 := float64(uint32(0x9883d8cf))/float64(math.MaxUint32) - 0.5
	for idx, rawBits := range map[int]uint32{1: 0x759ad11d, 25: 0x50886fd3, 26: 0xaae2586d, 27: 0x26cc7ecb} {
		wantRatio := (float64(rawBits)/float64(math.MaxUint32) - 0.5) / raw0
		gotRatio := float64(vec[idx]) / float64(vec[0])
		if math.Abs(gotRatio-wantRatio) > 1e-4 {
			t.Errorf("vec[%d]/vec[0] = %v, want %v", idx, gotRatio, wantRatio)
		}
	}

	score := New(Config{}).Score("ferry", "Ferry times")
	if score <= 0 || score >= 1 {
		t.Errorf("Score = %v, want in (0, 1)", score)
	}
}

func TestRoutesChatStreamAndValidationErrors(t *testing.T) {
	fake := New(Config{
		Models:          []Model{{ID: "m1", Name: "Model 1"}},
		IsLabeled:       true,
		EmbeddingModels: []Model{{ID: "e1", Name: "Embed 1"}},
	})

	// Unreadable request body -> 400.
	badBodyReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/chat/completions", brokenReader{})
	badBodyRec := httptest.NewRecorder()
	fake.ServeHTTP(badBodyRec, badBodyReq)
	if badBodyRec.Code != http.StatusBadRequest {
		t.Errorf("unreadable body status = %d, want 400", badBodyRec.Code)
	}

	// Chat validation errors -> 400.
	for _, body := range []string{`{`, `{"messages":[{"role":"user","content":"hi"}]}`, `{"model":"m"}`} {
		if got := post(t, fake, "/v1/chat/completions", "", body).Code; got != http.StatusBadRequest {
			t.Errorf("chat(%s) status = %d, want 400", body, got)
		}
	}
	// Embeddings validation errors -> 400.
	for _, body := range []string{`{`, `{"model":"m"}`} {
		if got := post(t, fake, "/v1/embeddings", "", body).Code; got != http.StatusBadRequest {
			t.Errorf("embeddings(%s) status = %d, want 400", body, got)
		}
	}
	// Rerank validation errors -> 400.
	for _, body := range []string{`{`, `{"query":"q","documents":["a"]}`, `{"model":"m","query":"q"}`} {
		if got := post(t, fake, "/v1/rerank", "", body).Code; got != http.StatusBadRequest {
			t.Errorf("rerank(%s) status = %d, want 400", body, got)
		}
	}

	// Chat non-streaming response content and Content-Type header.
	chatRec := post(t, fake, "/v1/chat/completions", "", `{"model":"m1","messages":[{"role":"user","content":"hello"}]}`)
	if chatRec.Code != http.StatusOK || chatRec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("chat status = %d, content-type = %q", chatRec.Code, chatRec.Header().Get("Content-Type"))
	}
	var chatOut struct {
		ID      string `json:"id"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(chatRec.Body.Bytes(), &chatOut); err != nil || len(chatOut.Choices) != 1 || chatOut.Choices[0].Index != 0 || chatOut.Choices[0].Message.Content != "echo: hello" {
		t.Errorf("chat response = %s (%v)", chatRec.Body.String(), err)
	}

	// Chat streaming response.
	streamRec := post(t, fake, "/v1/chat/completions", "", `{"model":"m1","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	if streamRec.Code != http.StatusOK || !strings.Contains(streamRec.Body.String(), "data: [DONE]") {
		t.Errorf("stream response = %d %q", streamRec.Code, streamRec.Body.String())
	}

	// GET routes and non-matching methods/paths.
	get := func(srv *Server, method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec
	}
	if rec := get(fake, http.MethodGet, "/v1/models"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "m1") {
		t.Errorf("GET /v1/models = %d %q", rec.Code, rec.Body.String())
	}
	if rec := get(fake, http.MethodGet, "/v1/embeddings/models"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "e1") {
		t.Errorf("GET /v1/embeddings/models (labeled) = %d %q", rec.Code, rec.Body.String())
	}
	unlabeled := New(Config{Models: []Model{{ID: "m1"}}})
	if rec := get(unlabeled, http.MethodGet, "/v1/embeddings/models"); rec.Code != http.StatusNotFound {
		t.Errorf("GET /v1/embeddings/models (unlabeled) = %d, want 404", rec.Code)
	}
	if rec := get(fake, http.MethodGet, "/v1/unknown"); rec.Code != http.StatusNotFound {
		t.Errorf("GET /v1/unknown = %d, want 404", rec.Code)
	}
	if rec := get(fake, http.MethodDelete, "/v1/models"); rec.Code != http.StatusNotFound {
		t.Errorf("DELETE /v1/models = %d, want 404", rec.Code)
	}
	if rec := get(fake, http.MethodDelete, "/v1/embeddings/models"); rec.Code != http.StatusNotFound {
		t.Errorf("DELETE /v1/embeddings/models = %d, want 404", rec.Code)
	}

	// Delay == 0 with an already-canceled context still serves the response without aborting;
	// Delay == 1ns with a live context waits and serves the response.
	canceledCtx, cancel := context.WithCancel(t.Context())
	cancel()
	canceledReq := httptest.NewRequestWithContext(canceledCtx, http.MethodGet, "/v1/models", nil)
	canceledRec := httptest.NewRecorder()
	fake.ServeHTTP(canceledRec, canceledReq)
	if canceledRec.Code != http.StatusOK {
		t.Errorf("delay=0 with canceled context = %d, want 200", canceledRec.Code)
	}
	fake.Delay("/models", time.Nanosecond)
	if rec := get(fake, http.MethodGet, "/v1/models"); rec.Code != http.StatusOK {
		t.Errorf("delay=1ns with live context = %d, want 200", rec.Code)
	}
	if len(fake.Requests()) == 0 {
		t.Error("expected recorded requests")
	}
}
