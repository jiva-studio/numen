package licensing

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

const (
	defaultHTTPTimeout  = 10 * time.Second
	maxResponseBodySize = 1 << 20
)

func getDefaultHTTPClient() HTTPDoer {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func sendJSON(ctx context.Context, client HTTPDoer, endpoint string, headers map[string]string, payload any) (int, []byte, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: build request: %w", ErrInvalidArgument, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, nil, err
		}
		return 0, nil, fmt.Errorf("%w: %w", ErrProviderUnavailable, err)
	}
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	closeErr := resp.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return 0, nil, fmt.Errorf("%w: read response: %w", ErrProviderUnavailable, err)
	}
	return resp.StatusCode, raw, nil
}

func parseOptionalTime(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return nil, err
		}
	}
	utc := parsed.UTC()
	return &utc, nil
}

func mapStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "active":
		return StatusActive
	case "expired":
		return StatusExpired
	case "disabled", "revoked":
		return StatusDisabled
	default:
		return StatusInactive
	}
}

func getStatusError(status Status) error {
	switch status {
	case StatusActive:
		return nil
	case StatusExpired:
		return ErrLicenseExpired
	case StatusDisabled:
		return ErrLicenseDisabled
	case StatusInactive:
		return ErrLicenseDisabled
	}
	return ErrLicenseDisabled
}
