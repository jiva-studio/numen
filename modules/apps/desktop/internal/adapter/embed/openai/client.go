// Package openai embeds text with a hosted model, over the /v1/embeddings
// request shape. Several services speak it, so an installation points at one by
// changing a base URL.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ErrRejected is a request the service refused to interpret. The batching rule
// produced something the model will never accept, so the error is final.
var ErrRejected = errors.New("the service rejected the request")

// ErrNoKey is a service configured without a key anywhere to find it.
var ErrNoKey = errors.New("no key in the configuration or the environment")

// Client is one hosted model.
type Client struct {
	model embed.ServiceModel
	is    port.EmbeddingModel
	http  *http.Client

	// Attempts is how many times one request is sent before its error is
	// reported. A real run meets "engine overloaded" repeatedly.
	Attempts int
	// Delay is the wait before the second attempt, doubling after each.
	Delay time.Duration
}

// New builds a client from configuration. The key is never an argument: it is
// read from the configuration or the environment, where no call site can copy
// it into a log.
//
// is is the identity the vectors this client returns are kept under, which the
// settings decide.
func New(is port.EmbeddingModel, model embed.ServiceModel) (*Client, error) {
	if model.BaseURL == "" {
		return nil, errors.New("no base URL for the embedding service")
	}
	if model.Name == "" {
		return nil, errors.New("no model name for the embedding service")
	}
	if is.Dimensions <= 0 {
		return nil, fmt.Errorf("%s: dimensions must be known before a vector is stored", model.Name)
	}
	if model.Key() == "" {
		return nil, fmt.Errorf("%w: %s", ErrNoKey, model)
	}
	return &Client{
		model:    model,
		is:       is,
		http:     &http.Client{Timeout: 90 * time.Second},
		Attempts: 5,
		Delay:    time.Second,
	}, nil
}

func (c *Client) Model() port.EmbeddingModel { return c.is }

// Close releases what the model holds on this machine, which is nothing: the
// weights are the service's.
func (c *Client) Close() error { return nil }

// Embed sends the texts in as few requests as the character budget allows.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out := make([][]float32, 0, len(texts))
	for _, batch := range embedding.Batches(texts, c.model.BatchCharacters) {
		vectors, err := c.request(ctx, batch)
		if err != nil {
			return nil, err
		}
		out = append(out, vectors...)
	}
	return out, nil
}

type request struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format"`
}

type response struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// temporary is an error worth sending the same request for again, carrying what
// the service asked to be waited.
type temporary struct {
	err   error
	after time.Duration
}

func (t temporary) Error() string { return t.err.Error() }
func (t temporary) Unwrap() error { return t.err }

// request sends one batch, retrying what the service says is temporary.
func (c *Client) request(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(request{Model: c.model.Name, Input: texts, EncodingFormat: "float"})
	if err != nil {
		return nil, err
	}
	attempts := max(c.Attempts, 1)
	delay := c.Delay
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			if err := wait(ctx, delay); err != nil {
				return nil, err
			}
			delay *= 2
		}
		vectors, err := c.send(ctx, body)
		if err == nil {
			return vectors, nil
		}
		var again temporary
		if !errors.As(err, &again) {
			return nil, err
		}
		last = err
		if again.after > 0 {
			delay = again.after
		}
	}
	return nil, fmt.Errorf("%d attempts: %w", attempts, last)
}

// send makes one call.
func (c *Client) send(ctx context.Context, body []byte) ([][]float32, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(c.model.BaseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.model.Key())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, temporary{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, c.statusError(resp, detail)
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, temporary{err: err}
	}
	return c.collect(parsed)
}

func (c *Client) statusError(resp *http.Response, detail []byte) error {
	status := resp.StatusCode
	summary := strings.TrimSpace(string(detail))
	if status == http.StatusBadRequest {
		return fmt.Errorf("%w as malformed, which is the batching rule to fix: %s", ErrRejected, summary)
	}
	if status >= 500 || status == http.StatusTooManyRequests {
		return temporary{
			err:   fmt.Errorf("%d %s: %s", status, http.StatusText(status), summary),
			after: retryDelay(resp),
		}
	}
	return fmt.Errorf("%w with %d %s: %s", ErrRejected, status, http.StatusText(status), summary)
}

// collect puts the vectors in the order the texts were sent, keyed by the index
// the service returns with each one.
func (c *Client) collect(parsed response) ([][]float32, error) {
	vectors := make([][]float32, len(parsed.Data))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(vectors) {
			return nil, fmt.Errorf("vector %d of %d is out of range", item.Index, len(vectors))
		}
		if len(item.Embedding) != c.is.Dimensions {
			return nil, fmt.Errorf("%s returned %d dimensions, configured as %d",
				c.model.Name, len(item.Embedding), c.is.Dimensions)
		}
		if vectors[item.Index] != nil {
			return nil, fmt.Errorf("vector %d arrived twice", item.Index)
		}
		vectors[item.Index] = embedding.Normalise(item.Embedding)
	}
	for i, v := range vectors {
		if v == nil {
			return nil, fmt.Errorf("vector %d is missing from the answer", i)
		}
	}
	return vectors, nil
}

// retryDelay is what the service asked to be waited, or zero for the client's
// own backoff.
func retryDelay(resp *http.Response) time.Duration {
	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(header); err == nil {
		if d := time.Until(at); d > 0 {
			return d
		}
	}
	return 0
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
