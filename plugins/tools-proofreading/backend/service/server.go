// Package service is the backend of the proofreading plugin: it reads a text
// version of a file, asks a model to correct its lines, stores the corrected
// text in the plugin's folder of the vault's store, and reports it to the host
// as a version derived from the one it read.
package service

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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/ai"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	proofreading "github.com/jiva-studio/numen/plugins/tools-proofreading"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.proofreading"

// EnvMockML names the variable that has the plugin answer every question with the mock proofreader.
const EnvMockML = "NUMEN_MOCK_ML"

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// purpose is what every question to the model is for. It routes nothing.
const purpose = "proofreading"

// completer sends a question to the host and returns the answer.
type completer func(ctx context.Context, question *ai.Question) (string, error)

// Host provides the plugin's host services: the store folder for corrected
// texts, the text versions it reads and reports, and the queue it runs in.
type Host struct {
	Store host.Store
	Texts host.Texts
	Queue host.Queue
}

// openHostAt connects to the host at the endpoint.
func openHostAt(endpoint hostcall.Endpoint) Host {
	return Host{
		Store: host.NewStoreClient(endpoint),
		Texts: host.NewTextClient(endpoint),
		Queue: host.NewQueueClient(endpoint),
	}
}

type proofreadingConfig struct {
	provider        string
	baseURL         string
	model           string
	apiKey          string
	batchSize       int
	overlap         int
	inFlight        int
	maxEditDistance float64
}

func defaultProofreadingConfig() proofreadingConfig {
	return proofreadingConfig{
		provider:        "agent",
		baseURL:         "",
		model:           "openai/gpt-4o-mini",
		apiKey:          "",
		batchSize:       DefaultBatchSize,
		overlap:         DefaultOverlap,
		inFlight:        DefaultInFlight,
		maxEditDistance: MaxEditDistance,
	}
}

// mockEnabled is the value of EnvMockML that selects the mock model.
const mockEnabled = "true"

// settings is the configuration the host last handed over.
type settings struct {
	mu  sync.RWMutex
	cfg proofreadingConfig
}

func newSettings() *settings {
	return &settings{mu: sync.RWMutex{}, cfg: defaultProofreadingConfig()}
}

func (s *settings) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if provider := values["provider"]; provider != "" {
		s.cfg.provider = provider
	}
	if baseURL, ok := values["baseUrl"]; ok {
		s.cfg.baseURL = baseURL
	}
	if model := values["model"]; model != "" {
		s.cfg.model = model
	}
	if key, ok := server.ReadSecret(values, server.SecretAPIKey); ok {
		s.cfg.apiKey = key
	}
	applyIntConfig(values, "batch_size", &s.cfg.batchSize, 1)
	applyIntConfig(values, "overlap", &s.cfg.overlap, 0)
	applyIntConfig(values, "in_flight", &s.cfg.inFlight, 1)
	if v, ok := values["max_edit_distance"]; ok {
		if d, err := strconv.ParseFloat(v, 64); err == nil && d >= 0 {
			s.cfg.maxEditDistance = d
		}
	}
}

func applyIntConfig(values map[string]string, key string, target *int, minVal int) {
	if v, ok := values[key]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= minVal {
			*target = n
		}
	}
}

func (s *settings) getConfig() proofreadingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// newAskModel returns the model as a system instruction and a prompt are put to it.
func newAskModel(complete completer, getConfig ...func() proofreadingConfig) askModel {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		if len(getConfig) > 0 && getConfig[0] != nil && os.Getenv(EnvMockML) != mockEnabled {
			cfg := getConfig[0]()
			if cfg.provider == "openrouter" || cfg.provider == "openai" || cfg.provider == "custom" {
				return askDirectAPI(ctx, cfg, instruction, prompt)
			}
		}
		return complete(ctx, &ai.Question{
			Purpose: purpose,
			Messages: []ai.Message{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
		})
	}
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
}

func resolveDirectBaseURL(cfg proofreadingConfig) string {
	switch cfg.provider {
	case "openrouter":
		if cfg.baseURL != "" {
			return cfg.baseURL
		}
		return "https://openrouter.ai/api/v1"
	case "openai":
		if cfg.baseURL != "" {
			return cfg.baseURL
		}
		return "https://api.openai.com/v1"
	default:
		return cfg.baseURL
	}
}

func askDirectAPI(ctx context.Context, cfg proofreadingConfig, instruction, prompt string) (string, error) {
	baseURL := strings.TrimSuffix(resolveDirectBaseURL(cfg), "/")
	if baseURL == "" {
		return "", errors.New("service URL is not configured")
	}
	model := cfg.model
	if model == "" {
		model = "openai/gpt-4o-mini"
	}
	body, err := json.Marshal(chatCompletionRequest{
		Model: model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	}
	client := &http.Client{
		Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: 2 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read chat response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chat API status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("chat API returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// Mount registers the proofread and delete commands on s, using the
// host that the environment names. The model is the mock proofreader when the
// environment enables mock models.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	complete := completer(ai.Complete)
	if os.Getenv(EnvMockML) == mockEnabled {
		complete = completeMock
	}
	return mount(s, openHostAt(endpoint), complete, nil)
}

// NewHandler returns the handler of the plugin's backend for a host that links
// the plugin in, configured from the variables given to a launched plugin
// process. It uses the host that those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	complete := func(ctx context.Context, question *ai.Question) (string, error) {
		client, err := ai.NewClientFromVars(vars)
		if err != nil {
			return "", err
		}
		return client.Complete(ctx, question)
	}
	if vars[EnvMockML] == mockEnabled {
		complete = completeMock
	}
	hosted := openHostAt(endpoint)
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, hosted, complete, vars)
	})
}

var errIncompleteHost = errors.New("proofreading needs the store, the texts and the queue of the host")

func mount(s *server.Server, h Host, complete completer, vars map[string]string) error {
	if h.Store == nil || h.Texts == nil || h.Queue == nil {
		return errIncompleteHost
	}
	config := newSettings()
	if len(vars) > 0 {
		config.updateConfig(vars)
	}
	kinds, err := readKinds(proofreading.Manifest)
	if err != nil {
		return err
	}
	run := newProofreading(h, kinds, newAskModel(complete, config.getConfig), config.getConfig)
	commands := command.New()
	if err := commands.Add("proofread", newProofreadCommand(run)); err != nil {
		return err
	}
	if err := commands.Add("delete-proofreading", command.NewDeleteTextRun(h.Texts, h.Store, PluginID)); err != nil {
		return err
	}
	s.SetUpdateConfig(config.updateConfig)
	command.Mount(s, commands)
	return nil
}
