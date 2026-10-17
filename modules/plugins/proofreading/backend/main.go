package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type proofreadingServer struct {
	mu             sync.RWMutex
	modelName      string
	baseURL        string
	strict         bool
	maxSuggestions int
}

func (s *proofreadingServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := values["model"]; ok && m != "" {
		s.modelName = m
	}
	if u, ok := values["baseUrl"]; ok && u != "" {
		s.baseURL = u
	}
	if v, ok := values["strictMode"]; ok {
		s.strict = v == "true"
	}
	if v, ok := values["maxSuggestions"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			s.maxSuggestions = n
		}
	}
}

func (s *proofreadingServer) GetProofreaderInfo(
	_ context.Context,
	_ *connect.Request[numenv1.GetProofreaderInfoRequest],
) (*connect.Response[numenv1.GetProofreaderInfoResponse], error) {
	s.mu.RLock()
	name := s.modelName
	s.mu.RUnlock()
	if name == "" {
		name = "llm-proofreader"
	}
	return connect.NewResponse(&numenv1.GetProofreaderInfoResponse{
		Name: name,
	}), nil
}

const defaultBaseURL = "https://openrouter.ai/api/v1"

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func callLLM(
	ctx context.Context,
	client *http.Client,
	baseURL, key, model, instruction, prompt string,
) (string, error) {
	if model == "" {
		model = "openai/gpt-4o-mini"
	}
	endpoint := strings.TrimSuffix(baseURL, "/") + "/chat/completions"

	body, err := json.Marshal(chatRequest{
		Model:       model,
		Temperature: 0,
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no completion returned")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func main() {
	proof := &proofreadingServer{
		modelName: os.Getenv("NUMEN_PROOFREADING_MODEL"),
		baseURL:   os.Getenv("NUMEN_PROOFREADING_URL"),
	}
	s := server.New(server.WithUpdateConfig(proof.updateConfig))
	s.Register(numenv1connect.NewProofreadingServiceHandler(proof, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

// getAPIKey is the key the model is called with.
func getAPIKey() string {
	if key := os.Getenv("NUMEN_PROOFREADING_KEY"); key != "" {
		return key
	}
	return os.Getenv("OPENAI_API_KEY")
}

func (s *proofreadingServer) ProofreadText(
	ctx context.Context,
	req *connect.Request[numenv1.ProofreadTextRequest],
) (*connect.Response[numenv1.ProofreadTextResponse], error) {
	kind := req.Msg.GetKind()
	if kind != numenv1.ProofreadKind_PROOFREAD_KIND_SCAN && kind != numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown kind %s", kind))
	}
	key := getAPIKey()
	if key == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("no API key: set NUMEN_PROOFREADING_KEY or OPENAI_API_KEY"))
	}

	s.mu.RLock()
	baseURL, model := s.baseURL, s.modelName
	opts := textOptions{
		strict:         s.strict,
		maxSuggestions: s.maxSuggestions,
		maxDistance:    req.Msg.GetMaxEditDistance(),
	}
	s.mu.RUnlock()
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	client := &http.Client{Timeout: 60 * time.Second}
	ask := func(ctx context.Context, instruction, prompt string) (string, error) {
		return callLLM(ctx, client, baseURL, key, model, instruction, prompt)
	}

	var (
		result textResult
		err    error
	)
	groups := readGroups(req.Msg.GetGroups())
	if kind == numenv1.ProofreadKind_PROOFREAD_KIND_SCAN {
		result, err = proofreadScan(ctx, ask, groups, opts)
	} else {
		var lines []Line
		for _, group := range groups {
			lines = append(lines, group.Lines...)
		}
		result, err = proofreadSpeech(ctx, ask, lines,
			int(req.Msg.GetSize()), int(req.Msg.GetOverlap()), int(req.Msg.GetFromLine()), int(req.Msg.GetLineLimit()), opts)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	corrections := make([]*numenv1.ProofreadCorrection, len(result.corrections))
	for i, line := range result.corrections {
		corrections[i] = &numenv1.ProofreadCorrection{
			Number: int32(line.Number),
			Last:   int32(line.Last),
			Text:   line.Text,
		}
	}
	return connect.NewResponse(&numenv1.ProofreadTextResponse{
		Corrections: corrections,
		Uncorrected: int32(result.uncorrected),
		NextLine:    int32(result.next),
		IsDone:      result.isDone,
	}), nil
}

// readGroups returns the batches a request carries.
func readGroups(groups []*numenv1.ProofreadGroup) []Batch {
	out := make([]Batch, len(groups))
	for i, group := range groups {
		lines := make([]Line, len(group.GetLines()))
		for j, line := range group.GetLines() {
			lines[j] = Line{Number: int(line.GetNumber()), Text: line.GetText()}
		}
		out[i] = Batch{Number: int(group.GetNumber()), Lines: lines}
	}
	return out
}
