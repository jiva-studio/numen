package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"
)

// DefaultBaseURL is the chat API asked where the settings name none.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// defaultModel is the model asked where neither the task nor the settings name one.
const defaultModel = "openai/gpt-4o-mini"

// requestLimit is how long one request to the chat API may take.
const requestLimit = 2 * time.Minute

var (
	errNoChoice  = errors.New("the chat API returned no message")
	errPlainKey  = errors.New("a key is sent only over https, or over http to this machine")
	errDowngrade = errors.New("a redirect would send a key over plain http")
	errNoKey     = errors.New("API key is not set: add it in Settings → Keys")
)

// message is one message of a conversation with the chat API.
type message struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// toolCall is a call of a function the model asks for.
type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// functionTool is a function offered to the model.
type functionTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

// model is one model the chat API lists.
type model struct {
	Name  string
	Title string
	// ReasoningField is the field of a request through which the chat API takes the depth of reasoning
	// for the model, empty where the API does not list one.
	ReasoningField string
	// ReasoningLevels are the depths the chat API lists for the model: those of its reasoning object,
	// else the common ones where it lists only the field.
	ReasoningLevels []reasoning.Level
	// DefaultReasoning is the depth the chat API lists the model working at unasked, empty where it lists none.
	DefaultReasoning string
}

// chatRequest is what the chat API is asked for the next message of a conversation.
type chatRequest struct {
	Model       string         `json:"model"`
	Messages    []message      `json:"messages"`
	Tools       []functionTool `json:"tools,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	MaxTokens   *int           `json:"max_tokens,omitempty"`
	// ReasoningEffort and Reasoning are the two forms in which a chat API takes the depth of reasoning.
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	Reasoning       *reasoningRequest `json:"reasoning,omitempty"`
}

// chat is an OpenAI-compatible chat API: its base URL and the key presented to it, empty for an
// endpoint that asks for none.
type chat struct {
	baseURL     string
	key         string
	client      *http.Client
	temperature *float64
	maxTokens   *int
}

// newText is a pointer to the text, for the content of a message.
func newText(text string) *string { return &text }

// send asks the model for the next message of the conversation, offering it the tools. It returns
// the message and why the model stopped writing it, 'length' for a message cut off at its limit.
func (c chat) send(ctx context.Context, model string, effort effort, messages []message, tools []functionTool) (message, string, error) {
	if err := c.checkKey(); err != nil {
		return message{}, "", err
	}
	req := newChatRequest(model, effort, messages, tools, false)
	req.Temperature = c.temperature
	req.MaxTokens = c.maxTokens
	body, err := json.Marshal(req)
	if err != nil {
		return message{}, "", fmt.Errorf("marshal chat request: %w", err)
	}
	answer, err := c.do(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return message{}, "", err
	}
	return readReply(answer)
}

// readReply decodes a whole answer of the chat API into its first message and why the model
// stopped writing it.
func readReply(answer []byte) (message, string, error) {
	var reply struct {
		Choices []struct {
			Message      message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(answer, &reply); err != nil {
		return message{}, "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(reply.Choices) == 0 {
		return message{}, "", errNoChoice
	}
	return reply.Choices[0].Message, reply.Choices[0].FinishReason, nil
}

// listModels asks the chat API for the models it serves.
func (c chat) listModels(ctx context.Context) ([]model, error) {
	answer, err := c.do(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	var listing struct {
		Data []struct {
			ID        string            `json:"id"`
			Name      string            `json:"name"`
			Supported []string          `json:"supported_parameters"`
			Reasoning *reasoningListing `json:"reasoning"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answer, &listing); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	models := make([]model, 0, len(listing.Data))
	for _, one := range listing.Data {
		if one.ID != "" {
			models = append(models, newModel(one.ID, one.Name, one.Supported, one.Reasoning))
		}
	}
	return models, nil
}

// do sends one request to the chat API and returns its answer. An answer other than 200 OK is an
// error.
func (c chat) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	resp, err := c.open(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newStatusError(resp, answer)
	}
	return answer, nil
}

// open sends one request to the chat API and returns the response as soon as it begins. A key is
// sent only where isPrivate allows, and no redirect takes it anywhere else.
func (c chat) open(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	base := strings.TrimSuffix(c.baseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.client
	if client == nil {
		client = &http.Client{Timeout: requestLimit}
	}
	if c.key != "" {
		if !isPrivate(req.URL) {
			return nil, errPlainKey
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		guarded := *client
		guarded.CheckRedirect = refuseDowngrade
		client = &guarded
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send chat request: %w", err)
	}
	return resp, nil
}

// newStatusError is an answer of the chat API other than 200 OK: its status and what it said. A
func newStatusError(resp *http.Response, answer []byte) error {
	text := fmt.Sprintf("the chat API answered %s: %s", resp.Status, strings.TrimSpace(string(answer)))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		text += " (check the API key in Settings → Keys)"
	}
	return errors.New(text)
}

// checkKey is errNoKey for a chat API that asks for a key when none is set. An endpoint on this
// machine asks for none.
func (c chat) checkKey() error {
	if c.key != "" {
		return nil
	}
	base := strings.TrimSuffix(c.baseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	address, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("the chat API address %q is not valid: %w", base, err)
	}
	if isLocalHost(address.Hostname()) {
		return nil
	}
	return errNoKey
}

// isLocalHost reports whether the host name is this machine.
func isLocalHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isPrivate reports whether what is sent to the address stays private: over https, or over http to
// this machine.
func isPrivate(address *url.URL) bool {
	if address.Scheme == "https" {
		return true
	}
	if address.Scheme != "http" {
		return false
	}
	return isLocalHost(address.Hostname())
}

// refuseDowngrade follows a redirect only while what is sent stays private, and at most ten times.
func refuseDowngrade(req *http.Request, via []*http.Request) error {
	if !isPrivate(req.URL) {
		return errDowngrade
	}
	if len(via) >= 10 {
		return errors.New("stopped after ten redirects")
	}
	return nil
}
