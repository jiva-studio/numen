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
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// ErrNoKey is a service configured without a key anywhere to find it.
var ErrNoKey = errors.New("no key in the configuration or the environment")

// inFlight is how many batches are being asked about at any moment where the
// profile names no number.
const inFlight = 4

const (
	// maxAttempts is how many times one page is sent before the last failure is
	// returned.
	maxAttempts = 4
	// baseBackoff is the ceiling of the wait after the first failure. The
	// ceiling doubles with each failure up to maxBackoff, and the wait is drawn
	// at random below it.
	baseBackoff = 500 * time.Millisecond
	// maxBackoff bounds every wait, including one the service asks for with
	// Retry-After.
	maxBackoff = 30 * time.Second
	// maxDrain is how much of a response body is read and discarded before the
	// body is closed.
	maxDrain = 64 << 10
)

// Client is one hosted model, asked about several pages at once.
type Client struct {
	service proofreading.Profile
	// instruction is what the model is told it is doing. A scan and speech are
	// corrected for different mistakes, and the caller says which.
	instruction string
	http        *http.Client
	// sleep waits for a duration or until the context ends.
	sleep func(context.Context, time.Duration) error
	// now is the instant a Retry-After date is measured from.
	now func() time.Time
	// jitter picks the wait below a ceiling.
	jitter func(ceiling time.Duration) time.Duration
}

// New builds a client from configuration. The key is never an argument: it is
// read from the configuration or the environment, where no call site can copy
// it into a log.
func New(cfg proofreading.Profile, instruction string) (*Client, error) {
	if cfg.Name == "" {
		return nil, errors.New("no model name for the proofreading service")
	}
	if instruction == "" {
		return nil, errors.New("no instruction for the proofreading service")
	}
	if cfg.Key() == "" {
		return nil, fmt.Errorf("%w: %s", ErrNoKey, cfg)
	}
	return &Client{
		service:     cfg,
		instruction: instruction,
		http:        &http.Client{Timeout: 120 * time.Second},
		sleep:       sleep,
		now:         time.Now,
		jitter:      func(ceiling time.Duration) time.Duration { return rand.N(ceiling + 1) },
	}, nil
}

// GetName is the model, recorded beside every correction it made.
func (c *Client) GetName() string { return c.service.Name }

// inFlight is how many batches this service is asked about at once.
func (c *Client) inFlight() int {
	if c.service.InFlight <= 0 {
		return inFlight
	}
	return c.service.InFlight
}

// Proofread asks about every page and answers with what came back about each,
// by the page it is about. A page nothing came back about is left out.
//
// One page that fails ends the run: the pages already answered are dropped and
// the caller asks again.
func (c *Client) Proofread(ctx context.Context, pages []proofread.Batch) (map[int]string, error) {
	if len(pages) == 0 {
		return nil, nil
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	queue := make(chan proofread.Batch)
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
	for range min(c.inFlight(), len(pages)) {
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
					out[page.Number] = reply
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

// ask sends one page and returns what the service said about it. A 429, a 408,
// a 5xx or a timeout is sent again after a wait, up to maxAttempts; any other
// failure is returned at once.
func (c *Client) ask(ctx context.Context, page proofread.Batch) (string, error) {
	body, err := json.Marshal(request{
		Model:       c.service.Name,
		Temperature: 0,
		Messages: []message{
			{Role: "system", Content: c.instruction},
			{Role: "user", Content: proofread.Ask(page)},
		},
	})
	if err != nil {
		return "", err
	}

	for attempt := 1; ; attempt++ {
		reply, again, err := c.send(ctx, page, body)
		if err == nil {
			return reply, nil
		}
		if !again.ok || attempt == maxAttempts || ctx.Err() != nil {
			return "", err
		}
		if err := c.sleep(ctx, c.backoff(attempt, again.after)); err != nil {
			return "", fmt.Errorf("page %d: %w", page.Number, err)
		}
	}
}

// retry says whether a failed request may be sent again, and how long the
// service asked to wait first.
type retry struct {
	ok    bool
	after time.Duration
}

// send makes one request. The response body is drained and closed on every
// path.
func (c *Client) send(ctx context.Context, page proofread.Batch, body []byte) (string, retry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(c.service.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", retry{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.service.Key())

	resp, err := c.http.Do(req)
	if err != nil {
		var netErr net.Error
		timedOut := errors.As(err, &netErr) && netErr.Timeout() && ctx.Err() == nil
		return "", retry{ok: timedOut}, fmt.Errorf("page %d: %w", page.Number, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		again := retry{ok: isTransient(resp.StatusCode), after: c.parseRetryAfter(resp.Header.Get("Retry-After"))}
		return "", again, fmt.Errorf("page %d: %d %s: %s", page.Number,
			resp.StatusCode, http.StatusText(resp.StatusCode), c.detail(detail))
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", retry{}, fmt.Errorf("page %d: %w", page.Number, err)
	}
	if len(parsed.Choices) == 0 {
		return "", retry{}, nil
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), retry{}, nil
}

// isTransient is whether a status is one the service recovers from.
func isTransient(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

// parseRetryAfter reads a Retry-After value, in seconds or as an HTTP date,
// into a wait. A value that is absent or unreadable is zero.
func (c *Client) parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil {
		return max(0, time.Duration(seconds)*time.Second)
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(0, at.Sub(c.now()))
	}
	return 0
}

// backoff is the wait after the given failed attempt. A wait the service asked
// for is used up to maxBackoff; otherwise it is drawn below a ceiling that
// doubles with each failure.
func (c *Client) backoff(attempt int, asked time.Duration) time.Duration {
	if asked > 0 {
		return min(asked, maxBackoff)
	}
	ceiling := baseBackoff
	for i := 1; i < attempt && ceiling < maxBackoff; i++ {
		ceiling *= 2
	}
	return c.jitter(min(ceiling, maxBackoff))
}

// sleep waits for d, or returns the context's error as soon as it ends.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
