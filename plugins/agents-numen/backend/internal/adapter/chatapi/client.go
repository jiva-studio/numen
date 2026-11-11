// Package chatapi reaches an OpenAI-compatible chat API over http: it asks for a message, streamed
// or whole, and lists the models.
package chatapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/endpoint"
)

// DefaultBaseURL is the chat API base URL used when the settings specify none.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// requestLimit is how long one request to the chat API may take.
const requestLimit = 2 * time.Minute

// ErrPlainKey is a chat API the key would reach unencrypted.
var ErrPlainKey = errors.New("a key is sent only over https, or over http to this machine")

var (
	errNoChoice = errors.New("the chat API returned no message")
	errNoKey    = errors.New("API key is not set: add it in Settings → Keys")
)

// Client is the chat API of the settings it is given on each call.
type Client struct {
	http *http.Client
}

// Option changes how a Client reaches the chat API.
type Option func(*Client)

// WithHTTPClient makes the client send its requests with c, in place of one that gives up after two
// minutes.
func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) { client.http = c }
}

// New returns a client of the chat API.
func New(opts ...Option) *Client {
	client := &Client{http: nil}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// Send asks for the next message of the conversation and returns it whole.
func (c *Client) Send(ctx context.Context, cfg settings.Settings, turn conversation.Turn) (conversation.Reply, error) {
	if err := checkKey(cfg); err != nil {
		return conversation.Reply{}, err
	}
	body, err := json.Marshal(newChatRequest(cfg, turn, false))
	if err != nil {
		return conversation.Reply{}, fmt.Errorf("marshal chat request: %w", err)
	}
	answer, err := c.do(ctx, cfg, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return conversation.Reply{}, err
	}
	return readReply(answer)
}

func readReply(answer []byte) (conversation.Reply, error) {
	var reply struct {
		Choices []struct {
			Message      conversation.Message `json:"message"`
			FinishReason string               `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(answer, &reply); err != nil {
		return conversation.Reply{}, fmt.Errorf("decode chat response: %w", err)
	}
	if len(reply.Choices) == 0 {
		return conversation.Reply{}, errNoChoice
	}
	return conversation.Reply{Message: reply.Choices[0].Message, FinishReason: reply.Choices[0].FinishReason}, nil
}

// ListModels lists the models the chat API offers, with the reasoning depths each takes.
func (c *Client) ListModels(ctx context.Context, cfg settings.Settings) ([]conversation.Model, error) {
	answer, err := c.do(ctx, cfg, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	var listing struct {
		Data []struct {
			ID        string                         `json:"id"`
			Name      string                         `json:"name"`
			Supported []string                       `json:"supported_parameters"`
			Reasoning *conversation.ReasoningListing `json:"reasoning"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answer, &listing); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	models := make([]conversation.Model, 0, len(listing.Data))
	for _, one := range listing.Data {
		if one.ID != "" {
			models = append(models, conversation.NewModel(one.ID, one.Name, one.Supported, one.Reasoning))
		}
	}
	return models, nil
}

func (c *Client) do(ctx context.Context, cfg settings.Settings, method, path string, body []byte) (answer []byte, err error) {
	resp, err := c.open(ctx, cfg, method, path, body)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	answer, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newStatusError(resp, answer)
	}
	return answer, nil
}

func getBaseURL(cfg settings.Settings) string {
	base := strings.TrimSuffix(cfg.BaseURL, "/")
	if base == "" {
		return DefaultBaseURL
	}
	return base
}

func (c *Client) open(ctx context.Context, cfg settings.Settings, method, path string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, getBaseURL(cfg)+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.http
	if client == nil {
		client = &http.Client{Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: requestLimit}
	}
	if cfg.Key != "" {
		if !endpoint.IsPrivate(req.URL) {
			return nil, ErrPlainKey
		}
		req.Header.Set("Authorization", "Bearer "+cfg.Key)
		guarded := *client
		guarded.CheckRedirect = endpoint.RefuseDowngrade
		client = &guarded
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send chat request: %w", err)
	}
	return resp, nil
}

func newStatusError(resp *http.Response, answer []byte) error {
	text := fmt.Sprintf("the chat API answered %s: %s", resp.Status, strings.TrimSpace(string(answer)))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		text += " (check the API key in Settings → Keys)"
	}
	return errors.New(text)
}

func checkKey(cfg settings.Settings) error {
	if cfg.Key != "" {
		return nil
	}
	base := getBaseURL(cfg)
	address, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("the chat API address %q is not valid: %w", base, err)
	}
	if endpoint.IsLocalHost(address.Hostname()) {
		return nil
	}
	return errNoKey
}
