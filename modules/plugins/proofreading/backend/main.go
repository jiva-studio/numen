package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
)

type proofreadingServer struct {
	mu        sync.RWMutex
	modelName string
}

func (s *proofreadingServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := values["model"]; ok && m != "" {
		s.modelName = m
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

func (s *proofreadingServer) Proofread(
	ctx context.Context,
	req *connect.Request[numenv1.ProofreadRequest],
) (*connect.Response[numenv1.ProofreadResponse], error) {
	batches := req.Msg.GetBatches()
	if len(batches) == 0 {
		return connect.NewResponse(&numenv1.ProofreadResponse{
			Replies: nil,
		}), nil
	}

	key := os.Getenv("NUMEN_PROOFREADING_KEY")
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}

	baseURL := os.Getenv("NUMEN_PROOFREADING_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	replies := make(map[int32]string, len(batches))

	s.mu.RLock()
	model := s.modelName
	s.mu.RUnlock()

	if key != "" {
		client := &http.Client{Timeout: 60 * time.Second}
		for _, b := range batches {
			reply, err := callLLM(ctx, client, baseURL, key, model, req.Msg.GetInstruction(), b.GetPrompt())
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("batch %d: %w", b.GetNumber(), err))
			}
			replies[b.GetNumber()] = reply
		}
	} else {
		for _, b := range batches {
			replies[b.GetNumber()] = b.GetPrompt()
		}
	}

	return connect.NewResponse(&numenv1.ProofreadResponse{
		Replies: replies,
	}), nil
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
	proof := &proofreadingServer{modelName: os.Getenv("NUMEN_PROOFREADING_MODEL")}
	s := server.New(server.WithUpdateConfig(proof.updateConfig))
	s.Register(numenv1connect.NewProofreadingServiceHandler(proof, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

