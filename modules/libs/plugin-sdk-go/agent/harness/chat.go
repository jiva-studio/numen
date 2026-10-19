package harness

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
	errDowngrade = errors.New("the chat API redirected a request that carries a key to http")
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
}

// chat is an OpenAI-compatible chat API: its base URL and the key presented to it, empty for an
// endpoint that asks for none.
type chat struct {
	baseURL string
	key     string
	client  *http.Client
}

// newText is a pointer to the text, for the content of a message.
func newText(text string) *string { return &text }

// send asks the model for the next message of the conversation, offering it the tools. It returns
// the message and why the model stopped writing it, 'length' for a message cut off at its limit.
func (c chat) send(ctx context.Context, model string, messages []message, tools []functionTool) (message, string, error) {
	body, err := json.Marshal(struct {
		Model    string         `json:"model"`
		Messages []message      `json:"messages"`
		Tools    []functionTool `json:"tools,omitempty"`
	}{Model: model, Messages: messages, Tools: tools})
	if err != nil {
		return message{}, "", fmt.Errorf("marshal chat request: %w", err)
	}

	var reply struct {
		Choices []struct {
			Message      message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := c.do(ctx, http.MethodPost, "/chat/completions", body, &reply); err != nil {
		return message{}, "", err
	}
	if len(reply.Choices) == 0 {
		return message{}, "", errNoChoice
	}
	return reply.Choices[0].Message, reply.Choices[0].FinishReason, nil
}

// listModels asks the chat API for the models it serves.
func (c chat) listModels(ctx context.Context) ([]model, error) {
	var listing struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/models", nil, &listing); err != nil {
		return nil, err
	}
	models := make([]model, 0, len(listing.Data))
	for _, one := range listing.Data {
		if one.ID != "" {
			models = append(models, model{Name: one.ID, Title: one.Name})
		}
	}
	return models, nil
}

// do sends one request to the chat API and decodes its answer into out.
func (c chat) do(ctx context.Context, method, path string, body []byte, out any) error {
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
		return fmt.Errorf("create chat request: %w", err)
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
			return errPlainKey
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		guarded := *client
		guarded.CheckRedirect = refuseDowngrade
		client = &guarded
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()

	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read chat response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the chat API answered %s: %s", resp.Status, strings.TrimSpace(string(answer)))
	}
	if err := json.Unmarshal(answer, out); err != nil {
		return fmt.Errorf("decode chat response: %w", err)
	}
	return nil
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
	host := address.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// refuseDowngrade follows a redirect only while what is sent stays private, and at most ten times.
func refuseDowngrade(req *http.Request, via []*http.Request) error {
	if !isPrivate(req.URL) {
		return errDowngrade
	}
	if len(via) >= 10 {
		return errors.New("the chat API redirected ten times")
	}
	return nil
}
