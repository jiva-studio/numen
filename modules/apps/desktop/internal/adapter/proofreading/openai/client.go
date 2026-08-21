// Package openai puts a reading right with a hosted model, over the
// /chat/completions request shape. Several services speak it, so an
// installation points at one by changing a base URL.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/proofread"
)

// ErrNoKey is a service configured without a key anywhere to find it.
var ErrNoKey = errors.New("no key in the configuration or the environment")

// inFlight is how many pages are being asked about at any moment.
const inFlight = 4

// Client is one hosted model. It asks about one page at a time.
type Client struct {
	service proofreading.Service
	http    *http.Client
}

// New builds a client from configuration. The key is never an argument: it is
// read from the configuration or the environment, where no call site can copy
// it into a log.
func New(cfg proofreading.Service) (*Client, error) {
	if cfg.Name == "" {
		return nil, errors.New("no model name for the proofreading service")
	}
	if cfg.Key() == "" {
		return nil, fmt.Errorf("%w: %s", ErrNoKey, cfg)
	}
	return &Client{service: cfg, http: &http.Client{Timeout: 120 * time.Second}}, nil
}

// Name is the model, recorded beside every correction it made.
func (c *Client) Name() string { return c.service.Name }

// Read asks about every page and answers with what came back about each, by the
// page it is about. A page nothing came back about is left out.
//
// One page that fails ends the run: the pages already answered are dropped and
// the caller asks again.
func (c *Client) Read(ctx context.Context, pages []proofread.Page) (map[int]string, error) {
	if len(pages) == 0 {
		return nil, nil
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	queue := make(chan proofread.Page)
	go func() {
		defer close(queue)
		for _, page := range pages {
			select {
			case queue <- page:
			case <-ctx.Done():
				return
			}
		}
	}()

	var (
		mu     sync.Mutex
		out    = make(map[int]string, len(pages))
		failed error
		wg     sync.WaitGroup
	)
	for range min(inFlight, len(pages)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range queue {
				reply, err := c.ask(ctx, page)
				mu.Lock()
				switch {
				case err != nil:
					if failed == nil {
						failed = err
						stop()
					}
				case reply != "":
					out[page.At] = reply
				}
				mu.Unlock()
				if err != nil {
					return
				}
			}
		}()
	}
	wg.Wait()

	if failed != nil {
		return nil, failed
	}
	return out, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature"`
	Messages    []message `json:"messages"`
}

type response struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

// ask sends one page and returns what the service said about it.
func (c *Client) ask(ctx context.Context, page proofread.Page) (string, error) {
	body, err := json.Marshal(request{
		Model:       c.service.Name,
		Temperature: 0,
		Messages: []message{
			{Role: "system", Content: proofread.Instruction},
			{Role: "user", Content: proofread.Ask(page)},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(c.service.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.service.Key())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("page %d: %w", page.At, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("page %d: %d %s: %s", page.At,
			resp.StatusCode, http.StatusText(resp.StatusCode), c.detail(body))
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("page %d: %w", page.At, err)
	}
	if len(parsed.Choices) == 0 {
		return "", nil
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// detail is a short piece of what the service said, with the key struck out of
// it. A service that quotes the request back quotes the key back.
func (c *Client) detail(body []byte) string {
	text := strings.TrimSpace(string(body))
	if key := c.service.Key(); key != "" {
		text = strings.ReplaceAll(text, key, "…")
	}
	return text
}
