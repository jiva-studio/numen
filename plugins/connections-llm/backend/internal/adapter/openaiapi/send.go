package openaiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
)

// maxResponseBytes bounds the answer of a service that is read into memory.
const maxResponseBytes = 64 << 20

// endpoint is the method and the path of a request.
type endpoint struct{ method, path string }

func readTarget(ctx context.Context, call connection.Call) (*url.URL, string, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", false, err
	}
	if !servicetype.IsKnown(call.Type) {
		return nil, "", false, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	key := call.Secrets[servicetype.SecretAPIKey].Value
	base, err := address.Parse(call.Fields[servicetype.FieldBaseURL], key != "")
	if err != nil {
		return nil, "", false, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	return base, key, base.Scheme == address.SchemeHTTP && key != "", nil
}

func (g *Gateway) send(ctx context.Context, call connection.Call, limit time.Duration, target endpoint, in, out any) error {
	base, key, isLoopbackRequired, err := readTarget(ctx, call)
	if err != nil {
		return err
	}

	limited, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	limited = context.WithValue(limited, loopbackOnlyKey{}, isLoopbackRequired)

	request, err := newRequest(limited, target, base, key, in)
	if err != nil {
		return connection.NewStopError(connection.ReasonUnavailable)
	}
	response, err := g.clients.get(isLoopbackRequired).Do(request)
	if err != nil {
		return classifyTransportError(ctx, err)
	}
	defer closeBody(response.Body)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return g.mapStatus(response)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return classifyTransportError(ctx, err)
	}
	if len(body) > maxResponseBytes || json.Unmarshal(body, out) != nil {
		return connection.NewStopError(connection.ReasonUnavailable)
	}
	return nil
}

func newRequest(ctx context.Context, target endpoint, base *url.URL, key string, in any) (*http.Request, error) {
	destination := *base
	destination.RawQuery, destination.Fragment = "", ""
	destination.Path = strings.TrimSuffix(destination.Path, "/") + target.path
	destination.RawPath = ""

	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, target.method, destination.String(), body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if in != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	return request, nil
}

func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		return
	}
}

func classifyTransportError(parent context.Context, err error) error {
	switch {
	case parent.Err() != nil:
		return parent.Err()
	case address.IsError(err):
		return connection.NewStopError(connection.ReasonInvalidRequest)
	}
	return connection.NewTryNextError(connection.ReasonUnavailable, 0)
}

func (g *Gateway) mapStatus(response *http.Response) error {
	return mapStatusCode(response.StatusCode, response.Header.Get("Retry-After"), g.clock.Now())
}

func mapStatusCode(status int, retryAfter string, now time.Time) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return connection.NewTryNextError(connection.ReasonRejectedKey, 0)
	case status == http.StatusPaymentRequired:
		return connection.NewTryNextError(connection.ReasonPaymentRequired, 0)
	case status == http.StatusTooManyRequests:
		return connection.NewTryNextError(connection.ReasonRateLimited, parseRetryAfter(retryAfter, now))
	case status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout:
		return connection.NewTryNextError(connection.ReasonUnavailable, 0)
	case status >= 400 && status < 500:
		return connection.NewStopError(connection.ReasonInvalidRequest)
	}
	return connection.NewStopError(connection.ReasonUnavailable)
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 32); err == nil {
		return time.Duration(max(seconds, 0)) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(date.Sub(now), 0)
	}
	return 0
}
