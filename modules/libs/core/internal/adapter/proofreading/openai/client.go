// Package openai puts a text right with a hosted model, over the
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
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/retry"
)

// ErrNoKey is a service configured without a key anywhere to find it.
var ErrNoKey = errors.New("no key in the configuration or the environment")

// maxDrain is how much of a response body is read and discarded before the
// body is closed.
const maxDrain = 64 << 10

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
		sleep:       retry.Wait,
		now:         time.Now,
		jitter:      func(ceiling time.Duration) time.Duration { return rand.N(ceiling + 1) },
	}, nil
}

// GetName is the model, recorded beside every correction it made.
func (c *Client) GetName() string { return c.service.Name }

// getInFlight is how many batches this service is asked about at once.
func (c *Client) getInFlight() int {
	return c.service.GetInFlight(0)
}

// Proofread asks about every page and answers with what came back about each,
// by the page it is about. A page nothing came back about is left out.
//
// One page that fails ends the run: the pages already answered are dropped and
// the caller asks again.
func (c *Client) Proofread(ctx context.Context, pages []domain.ProofreadBatch) (map[int]string, error) {
	if len(pages) == 0 {
		return nil, nil
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	queue := make(chan domain.ProofreadBatch)
	go queuePages(ctx, queue, pages)

	run := &proofreadRun{out: make(map[int]string, len(pages)), stop: stop}
	var wg sync.WaitGroup
	for range min(c.getInFlight(), len(pages)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.work(ctx, queue, run)
		}()
	}
	wg.Wait()

	if run.failed != nil {
		return nil, run.failed
	}
	return run.out, nil
}

// queuePages hands the pages on until they are all taken or the context ends,
// and closes the queue.
func queuePages(ctx context.Context, queue chan<- domain.ProofreadBatch, pages []domain.ProofreadBatch) {
	defer close(queue)
	for _, page := range pages {
		select {
		case queue <- page:
		case <-ctx.Done():
			return
		}
	}
}

// proofreadRun is what the workers of one run share: the replies by page, and
// the first failure, which stops the run.
type proofreadRun struct {
	mu     sync.Mutex
	out    map[int]string
	failed error
	stop   context.CancelFunc
}

// record keeps what came back about a page. The first failure is kept and stops
// the run.
func (r *proofreadRun) record(page domain.ProofreadBatch, reply string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case err != nil:
		if r.failed == nil {
			r.failed = err
			r.stop()
		}
	case reply != "":
		r.out[page.Number] = reply
	}
}

// work asks about the pages of the queue one at a time, and ends at the first
// page that fails.
func (c *Client) work(ctx context.Context, queue <-chan domain.ProofreadBatch, run *proofreadRun) {
	for page := range queue {
		reply, err := c.sendWithRetry(ctx, page)
		run.record(page, reply, err)
		if err != nil {
			return
		}
	}
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

// sendWithRetry sends one page and returns what the service said about it. A 429, a 408,
// a 5xx or a timeout is sent again after a wait, up to the profile's attempts;
// any other failure is returned at once.
func (c *Client) sendWithRetry(ctx context.Context, page domain.ProofreadBatch) (string, error) {
	body, err := json.Marshal(request{
		Model:       c.service.Name,
		Temperature: 0,
		Messages: []message{
			{Role: "system", Content: c.instruction},
			{Role: "user", Content: page.Prompt},
		},
	})
	if err != nil {
		return "", err
	}

	for attempt := 1; ; attempt++ {
		reply, next, err := c.send(ctx, page, body)
		if err == nil {
			return reply, nil
		}
		if !next.ok || attempt >= c.getAttempts() || ctx.Err() != nil {
			return "", err
		}
		if err := c.sleep(ctx, c.getBackoff(attempt, next.after)); err != nil {
			return "", fmt.Errorf("page %d: %w", page.Number, err)
		}
	}
}

// retryDecision says whether a failed request may be sent retryDecision, and how long the
// service asked to wait first.
type retryDecision struct {
	ok    bool
	after time.Duration
}

// send makes one request. The response body is drained and closed on every
// path.
func (c *Client) send(ctx context.Context, page domain.ProofreadBatch, body []byte) (string, retryDecision, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(c.service.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", retryDecision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.service.Key())

	resp, err := c.http.Do(req)
	if err != nil {
		var netErr net.Error
		timedOut := errors.As(err, &netErr) && netErr.Timeout() && ctx.Err() == nil
		return "", retryDecision{ok: timedOut}, fmt.Errorf("page %d: %w", page.Number, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		next := retryDecision{ok: isTransient(resp.StatusCode), after: retry.ParseAfter(resp.Header.Get("Retry-After"), c.now())}
		return "", next, fmt.Errorf("page %d: %d %s: %s", page.Number,
			resp.StatusCode, http.StatusText(resp.StatusCode), c.getErrorDetail(detail))
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", retryDecision{}, fmt.Errorf("page %d: %w", page.Number, err)
	}
	if len(parsed.Choices) == 0 {
		return "", retryDecision{}, nil
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), retryDecision{}, nil
}

// isTransient is whether a status is one the service recovers from.
func isTransient(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

// getAttempts is how many times one page is sent.
func (c *Client) getAttempts() int {
	if c.service.Attempts <= 0 {
		return proofreading.DefaultAttempts
	}
	return c.service.Attempts
}

// getDelay is the ceiling of the wait after the first failure.
func (c *Client) getDelay() time.Duration {
	if c.service.DelayMS <= 0 {
		return proofreading.DefaultDelayMS * time.Millisecond
	}
	return time.Duration(c.service.DelayMS) * time.Millisecond
}

// getMaxDelay is the longest any wait is.
func (c *Client) getMaxDelay() time.Duration {
	if c.service.MaxDelayMS <= 0 {
		return proofreading.DefaultMaxDelayMS * time.Millisecond
	}
	return time.Duration(c.service.MaxDelayMS) * time.Millisecond
}

// getBackoff is the wait after the given failed attempt. A wait the service asked
// for is used up to c.maxDelay(); otherwise it is drawn below a ceiling that
// doubles with each failure.
func (c *Client) getBackoff(attempt int, asked time.Duration) time.Duration {
	if asked > 0 {
		return min(asked, c.getMaxDelay())
	}
	ceiling := c.getDelay()
	for i := 1; i < attempt && ceiling < c.getMaxDelay(); i++ {
		ceiling *= 2
	}
	return c.jitter(min(ceiling, c.getMaxDelay()))
}

// getErrorDetail is a short piece of what the service said, with the key struck out of
// it. A service that quotes the request back quotes the key back.
func (c *Client) getErrorDetail(body []byte) string {
	text := strings.TrimSpace(string(body))
	if key := c.service.Key(); key != "" {
		text = strings.ReplaceAll(text, key, "…")
	}
	return text
}
