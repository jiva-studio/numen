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

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// The statuses a batch is in. A batch in one of the three ends answers about
// nothing.
const (
	statusCompleted = "completed"
	statusFailed    = "failed"
	statusExpired   = "expired"
	statusCancelled = "cancelled"
)

// batchRequest is a run of pages left with the queue. The service stream-parses
// the requests, so the fields stand in the order it reads them.
type batchRequest struct {
	Endpoint string           `json:"endpoint"`
	Model    string           `json:"model"`
	Requests []batchedRequest `json:"requests"`
}

// batchedRequest is one page of the run: the number what comes back is known by,
// and the body one page is asked with on its own.
type batchedRequest struct {
	CustomID string  `json:"custom_id"`
	Body     request `json:"body"`
}

// batch is what the service says about a run it holds. A completed batch
// carries its results inline.
type batch struct {
	ID      string        `json:"id"`
	Status  string        `json:"status"`
	Results []batchResult `json:"results"`
}

// batchResult is what came back about one page of the run.
type batchResult struct {
	CustomID string `json:"custom_id"`
	Response struct {
		Body response `json:"body"`
	} `json:"response"`
}

// Leave hands the pages to the queue and answers with the id they are collected
// under.
func (c *Client) Leave(ctx context.Context, pages []proofread.Batch) (string, error) {
	if c.service.BatchURL == "" {
		return "", errors.New("no batch queue for the proofreading service")
	}

	asked := make([]batchedRequest, 0, len(pages))
	for _, page := range pages {
		asked = append(asked, batchedRequest{
			CustomID: strconv.Itoa(page.Number),
			Body: request{
				Model:       c.service.Name,
				Temperature: 0,
				Messages: []message{
					{Role: "system", Content: c.instruction},
					{Role: "user", Content: proofread.Ask(page)},
				},
			},
		})
	}
	body, err := json.Marshal(batchRequest{
		Endpoint: "/v1/chat/completions",
		Model:    c.service.Name,
		Requests: asked,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.queue(""), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.service.Key())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("leaving %d pages: %w", len(pages), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("leaving %d pages: %d %s: %s", len(pages),
			resp.StatusCode, http.StatusText(resp.StatusCode), c.said(resp.Body))
	}

	var left batch
	if err := json.NewDecoder(resp.Body).Decode(&left); err != nil {
		return "", fmt.Errorf("leaving %d pages: %w", len(pages), err)
	}
	if left.ID == "" {
		return "", fmt.Errorf("leaving %d pages: the service named no batch", len(pages))
	}
	return left.ID, nil
}

// Collect is what came back about the pages left under a name, by the page it
// is about, and whether the service is done with them. A result under a name
// that is not a page number, or carrying nothing, is left out.
func (c *Client) Collect(ctx context.Context, name string) (map[int]string, bool, error) {
	if c.service.BatchURL == "" {
		return nil, false, errors.New("no batch queue for the proofreading service")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.queue("/"+name), nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.service.Key())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("batch %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("batch %s: %d %s: %s", name,
			resp.StatusCode, http.StatusText(resp.StatusCode), c.said(resp.Body))
	}

	var held batch
	if err := json.NewDecoder(resp.Body).Decode(&held); err != nil {
		return nil, false, fmt.Errorf("batch %s: %w", name, err)
	}

	switch held.Status {
	case statusFailed, statusExpired, statusCancelled:
		return nil, false, fmt.Errorf("batch %s: %s", name, held.Status)
	case statusCompleted:
	default:
		return nil, false, nil
	}

	out := make(map[int]string, len(held.Results))
	for _, result := range held.Results {
		at, err := strconv.Atoi(result.CustomID)
		if err != nil {
			continue
		}
		if len(result.Response.Body.Choices) == 0 {
			continue
		}
		if reply := strings.TrimSpace(result.Response.Body.Choices[0].Message.Content); reply != "" {
			out[at] = reply
		}
	}
	return out, true, nil
}

// queue is the batch service, with a path appended.
func (c *Client) queue(path string) string {
	return strings.TrimSuffix(c.service.BatchURL, "/") + path
}

// said is a short piece of what the service answered, with the key struck out
// of it.
func (c *Client) said(body io.Reader) string {
	read, _ := io.ReadAll(io.LimitReader(body, 512))
	return c.detail(read)
}
