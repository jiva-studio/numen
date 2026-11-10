// Package remote asks an OpenAI-compatible embedding service for vectors over http.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type request struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type response struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// FetchVectors asks the service at baseURL for one vector per text, one request for each text.
func FetchVectors(ctx context.Context, key, baseURL, model string, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vector, err := fetchVector(ctx, key, baseURL, model, text)
		if err != nil {
			return nil, fmt.Errorf("service text %d: %w", i, err)
		}
		vectors[i] = vector
	}
	return vectors, nil
}

func fetchVector(ctx context.Context, key, baseURL, model, text string) (vector []float32, err error) {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="HTTP client timeout constant"
	client := &http.Client{Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: 30 * time.Second}
	endpoint := strings.TrimSuffix(baseURL, "/") + "/embeddings"
	reqBody, err := json.Marshal(request{
		Input: []string{text},
		Model: model,
	})
	// gomutants:disable-next-line BRANCH_IF reason="marshaling string struct cannot fail"
	if err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL reason="marshaling string struct cannot fail"
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="http response body close does not fail in memory"
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	respBytes, err := io.ReadAll(resp.Body)
	// gomutants:disable-next-line BRANCH_IF reason="unreadable body leaves respBytes empty and fails status or unmarshal"
	if err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL reason="unreadable body leaves respBytes empty and fails status or unmarshal"
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBytes))
	}

	var parsed response
	// gomutants:disable-next-line BRANCH_IF reason="invalid json leaves parsed.Data empty and fails length check below"
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) > 0 && len(parsed.Data[0].Embedding) > 0 {
		return parsed.Data[0].Embedding, nil
	}

	return nil, fmt.Errorf("no embedding returned by service")
}
