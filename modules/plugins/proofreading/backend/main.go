package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type lifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownTrigger chan struct{}
	proof           *proofreadingServer
}

func (s *lifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[numenv1.PingRequest],
) (*connect.Response[numenv1.PingResponse], error) {
	return connect.NewResponse(&numenv1.PingResponse{
		Status: "ok",
	}), nil
}

func (s *lifecycleServer) Configure(
	_ context.Context,
	req *connect.Request[numenv1.ConfigureRequest],
) (*connect.Response[numenv1.ConfigureResponse], error) {
	if s.proof != nil {
		s.proof.updateConfig(req.Msg.GetValues())
	}
	return connect.NewResponse(&numenv1.ConfigureResponse{
		Success: true,
	}), nil
}

func (s *lifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[numenv1.ShutdownRequest],
) (*connect.Response[numenv1.ShutdownResponse], error) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(s.shutdownTrigger)
	}()
	return connect.NewResponse(&numenv1.ShutdownResponse{
		Success: true,
	}), nil
}

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

func authCheck(expectedToken string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if expectedToken != "" {
				authHeader := req.Header().Get("Authorization")
				expected := "Bearer " + expectedToken
				if authHeader != expected {
					return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
				}
			}
			return next(ctx, req)
		}
	}
}

func main() {
	var portFlag int
	var tokenFlag string

	flag.IntVar(&portFlag, "port", 0, "Port to listen on")
	flag.StringVar(&tokenFlag, "token", "", "Authentication token")
	flag.Parse()

	if portFlag == 0 {
		if envPort := os.Getenv("NUMEN_PLUGIN_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				portFlag = p
			}
		}
	}
	if tokenFlag == "" {
		tokenFlag = os.Getenv("NUMEN_PLUGIN_TOKEN")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port

	shutdownCh := make(chan struct{})
	proof := &proofreadingServer{modelName: os.Getenv("NUMEN_PROOFREADING_MODEL")}
	life := &lifecycleServer{shutdownTrigger: shutdownCh, proof: proof}

	interceptors := connect.WithInterceptors(authCheck(tokenFlag))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life, interceptors))
	mux.Handle(numenv1connect.NewProofreadingServiceHandler(proof, interceptors))

	handler := h2c.NewHandler(mux, &http2.Server{})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	// Output ready handshake for plugin host
	fmt.Printf("NUMEN_READY port=%d\n", actualPort)
	_ = os.Stdout.Sync()

	stdinEOF := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(stdinEOF)
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-shutdownCh:
	case <-stdinEOF:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
