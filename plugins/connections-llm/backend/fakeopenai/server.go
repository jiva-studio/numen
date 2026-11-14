package fakeopenai

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// DefaultDimensions is the width of a vector when Config sets none.
const DefaultDimensions = 8

const (
	modelKey   = "model"
	objectKey  = "object"
	indexKey   = "index"
	contentKey = "content"
)

// Model is a model the server lists. Name is the label OpenRouter gives; an empty Name leaves the
// label out.
type Model struct {
	ID   string
	Name string
}

// Config sets what the server answers. A zero Config serves no key check, no models and vectors of
// DefaultDimensions. When APIKey is set, every request must carry it as a bearer token. With IsLabeled,
// Models carry the architecture and the supported parameters OpenRouter gives, and EmbeddingModels are
// listed at /embeddings/models, which OpenRouter serves.
type Config struct {
	APIKey          string
	Dimensions      int
	Models          []Model
	IsLabeled       bool
	EmbeddingModels []Model
}

// Request is one request the server received.
type Request struct {
	Method string
	Path   string
	Header http.Header
	Body   []byte
}

// Stream is one scripted answer to a chat request that asks for a stream. Chunks are the data of the
// events, in order; "[DONE]" is the end of the stream. When IsBroken is set the server closes the
// connection after the chunks, with no end. Pause is the wait after each chunk. A script that does not end with "[DONE]" stalls until the
// client goes away. A request that finds no script gets an echo.
type Stream struct {
	Chunks   []string
	IsBroken bool
	Pause    time.Duration
}

type behavior struct {
	status     int
	retryAfter string
	delay      time.Duration
}

// Server answers the OpenAI API. It is safe for concurrent use.
type Server struct {
	config   Config
	mu       sync.Mutex
	requests []Request
	behavior map[string]behavior
	aborted  chan struct{}
	streams  []Stream
}

// New creates a server that answers as config says.
func New(config Config) *Server {
	if config.Dimensions <= 0 {
		config.Dimensions = DefaultDimensions
	}
	return &Server{config: config, mu: sync.Mutex{}, requests: nil, behavior: map[string]behavior{}, aborted: make(chan struct{}, 64), streams: nil}
}

// Fail makes a path answer status with an empty body and the header Retry-After. A status of 0 clears it.
// The path is the API path, for example /chat/completions.
func (s *Server) Fail(path string, status int, retryAfter string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	one := s.behavior[path]
	one.status, one.retryAfter = status, retryAfter
	s.behavior[path] = one
}

// QueueStream adds scripted answers for the chat requests that ask for a stream, used in order.
func (s *Server) QueueStream(streams ...Stream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams = append(s.streams, streams...)
}

// Delay makes a path wait before it answers. The wait ends when the client disconnects.
func (s *Server) Delay(path string, wait time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	one := s.behavior[path]
	one.delay = wait
	s.behavior[path] = one
}

// Requests returns the requests received, oldest first.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// Aborted receives one value for every delayed request whose client went away before the delay ended.
func (s *Server) Aborted() <-chan struct{} { return s.aborted }

// ServeHTTP answers the paths under /v1 or at the root.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unreadable body", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.requests = append(s.requests, Request{Method: r.Method, Path: path, Header: r.Header.Clone(), Body: body})
	one := s.behavior[path]
	s.mu.Unlock()

	if one.delay > 0 && !s.wait(r, one.delay) {
		return
	}
	if one.status != 0 {
		writeStatus(w, one)
		return
	}
	if !s.isAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "invalid key")
		return
	}
	s.route(w, r, path, body)
}

func writeStatus(w http.ResponseWriter, one behavior) {
	if one.retryAfter != "" {
		w.Header().Set("Retry-After", one.retryAfter)
	}
	w.WriteHeader(one.status)
}

func (s *Server) isAuthorized(r *http.Request) bool {
	return s.config.APIKey == "" || r.Header.Get("Authorization") == "Bearer "+s.config.APIKey
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"message": message}})
}

func (s *Server) route(w http.ResponseWriter, r *http.Request, path string, body []byte) {
	if r.Method == http.MethodPost {
		s.routePost(w, r, path, body)
		return
	}
	switch {
	case r.Method == http.MethodGet && path == "/models":
		s.serveModels(w, s.config.Models, s.config.IsLabeled, false)
	case r.Method == http.MethodGet && path == "/embeddings/models" && s.config.IsLabeled:
		s.serveModels(w, s.config.EmbeddingModels, true, true)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) routePost(w http.ResponseWriter, r *http.Request, path string, body []byte) {
	switch path {
	case "/chat/completions":
		s.serveChat(w, r, body)
	case "/embeddings":
		s.serveEmbeddings(w, body)
	case "/rerank":
		s.serveRerank(w, body)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) wait(r *http.Request, wait time.Duration) bool {
	waiting, stop := context.WithTimeout(r.Context(), wait)
	defer stop()
	<-waiting.Done()
	if r.Context().Err() == nil {
		return true
	}
	select {
	case s.aborted <- struct{}{}:
	default:
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		return
	}
}

type chatRequest struct {
	Model    string `json:"model"`
	Stream   bool   `json:"stream"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (s *Server) serveChat(w http.ResponseWriter, r *http.Request, body []byte) {
	var request chatRequest
	if err := json.Unmarshal(body, &request); err != nil || request.Model == "" || len(request.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "model and messages are required")
		return
	}
	last := request.Messages[len(request.Messages)-1].Content
	if request.Stream {
		s.serveStream(w, r, "echo: "+last)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     "chatcmpl-fake",
		modelKey: request.Model,
		"choices": []any{map[string]any{
			indexKey:        0,
			"message":       map[string]any{"role": "assistant", contentKey: "echo: " + last},
			"finish_reason": "stop",
		}},
	})
}

func (s *Server) takeScript(echo string) Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.streams) == 0 {
		return Stream{Chunks: echoChunks(echo), IsBroken: false, Pause: 0}
	}
	script := s.streams[0]
	s.streams = s.streams[1:]
	return script
}

func (s *Server) serveStream(w http.ResponseWriter, r *http.Request, echo string) {
	script := s.takeScript(echo)
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	controller := http.NewResponseController(w)
	if !s.writeChunks(w, r, controller, script) {
		return
	}
	if script.IsBroken {
		closeConnection(controller)
		return
	}
	if n := len(script.Chunks); n == 0 || script.Chunks[n-1] != "[DONE]" {
		<-r.Context().Done()
	}
}

func (s *Server) writeChunks(w http.ResponseWriter, r *http.Request, controller *http.ResponseController, script Stream) bool {
	for _, chunk := range script.Chunks {
		if _, err := io.WriteString(w, "data: "+chunk+"\n\n"); err != nil {
			return false
		}
		if err := controller.Flush(); err != nil {
			return false
		}
		if script.Pause > 0 && !s.wait(r, script.Pause) {
			return false
		}
	}
	return true
}

func echoChunks(echo string) []string {
	runes := []rune(echo)
	return []string{
		encodeDelta(map[string]any{"role": "assistant", contentKey: string(runes[:len(runes)/2])}, ""),
		encodeDelta(map[string]any{contentKey: string(runes[len(runes)/2:])}, ""),
		encodeDelta(map[string]any{}, "stop"),
		"[DONE]",
	}
}

func encodeDelta(delta map[string]any, finishReason string) string {
	choice := map[string]any{indexKey: 0, "delta": delta}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	}
	encoded, err := json.Marshal(map[string]any{"choices": []any{choice}})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

func (s *Server) serveEmbeddings(w http.ResponseWriter, body []byte) {
	var request embeddingsRequest
	if err := json.Unmarshal(body, &request); err != nil || request.Model == "" || len(request.Input) == 0 {
		writeError(w, http.StatusBadRequest, "model and input are required")
		return
	}
	data := make([]any, len(request.Input))
	for i, text := range request.Input {
		data[i] = map[string]any{objectKey: "embedding", indexKey: i, "embedding": s.Vector(text)}
	}
	writeJSON(w, http.StatusOK, map[string]any{objectKey: "list", modelKey: request.Model, "data": data})
}

// Vector returns the vector the server answers for a text: unit length and the same for the same text.
func (s *Server) Vector(text string) []float32 {
	vector := make([]float32, s.config.Dimensions)
	var sum float64
	for i := range vector {
		digest := sha256.Sum256([]byte(text + "\x00" + string(rune('a'+i%26)) + string(rune('0'+i/26))))
		value := float64(binary.BigEndian.Uint32(digest[:4]))/float64(math.MaxUint32) - 0.5
		vector[i] = float32(value)
		sum += value * value
	}
	norm := float32(math.Sqrt(sum))
	for i := range vector {
		vector[i] /= norm
	}
	return vector
}

func (s *Server) serveModels(w http.ResponseWriter, models []Model, isLabeled, isEmbedding bool) {
	data := make([]any, len(models))
	for i, model := range models {
		entry := map[string]any{"id": model.ID, objectKey: "model"}
		if isLabeled {
			entry["name"] = model.Name
			output := "text"
			if isEmbedding {
				output = "embeddings"
			}
			entry["architecture"] = map[string]any{"input_modalities": []string{"text"}, "output_modalities": []string{output}}
			entry["context_length"] = 128000
			entry["supported_parameters"] = []string{"tools", "reasoning"}
		}
		data[i] = entry
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func closeConnection(controller *http.ResponseController) {
	conn, _, err := controller.Hijack()
	if err != nil {
		return
	}
	if err := conn.Close(); err != nil {
		return
	}
}

type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

func (s *Server) serveRerank(w http.ResponseWriter, body []byte) {
	var request rerankRequest
	if err := json.Unmarshal(body, &request); err != nil || request.Model == "" || len(request.Documents) == 0 {
		writeError(w, http.StatusBadRequest, "model and documents are required")
		return
	}
	order := make([]int, len(request.Documents))
	for i := range order {
		order[i] = i
	}
	scores := make([]float32, len(request.Documents))
	for i, document := range request.Documents {
		scores[i] = s.Score(request.Query, document)
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(scores[b], scores[a]) })
	results := make([]any, len(order))
	for i, index := range order {
		results[i] = map[string]any{indexKey: index, "relevance_score": scores[index]}
	}
	writeJSON(w, http.StatusOK, map[string]any{modelKey: request.Model, "results": results})
}

// Score returns the relevance score the server answers at /rerank for a query and a document: between 0 and
// 1 and the same for the same pair. The results of /rerank come in descending score order, as Cohere and
// Jina list them.
func (s *Server) Score(query, document string) float32 {
	digest := sha256.Sum256([]byte(query + "\x00" + document))
	return float32(float64(binary.BigEndian.Uint32(digest[:4])) / float64(math.MaxUint32))
}
