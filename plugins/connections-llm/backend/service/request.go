package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

// maxResponseBytes bounds the answer of a service that is read into memory.
const maxResponseBytes = 64 << 20

// send makes one request to the service of the call and decodes a successful answer into out. A
// failure is a connection failure with a reason, or the error of the context of the call.
// endpoint names the operation a request is made for, which sets its time limit, and the method and
// path it uses.
type endpoint struct{ operation, method, path string }

func (s *Service) send(ctx context.Context, call connection.Call, target endpoint, in, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !isKnownType(call.Type) {
		return connection.NewStopError(connection.ReasonBadRequest)
	}
	key := call.Secrets[secretAPIKey].Value
	base, err := parseAddress(call.Fields[fieldBaseURL], key != "")
	if err != nil {
		return connection.NewStopError(connection.ReasonBadRequest)
	}
	isLoopbackRequired := base.Scheme == schemeHTTP && key != ""

	limit, cancel := context.WithTimeout(ctx, s.getLimit(call, target.operation))
	defer cancel()
	limit = context.WithValue(limit, loopbackOnlyKey{}, isLoopbackRequired)

	request, err := newRequest(limit, target, base, key, in)
	if err != nil {
		return connection.NewStopError(connection.ReasonUnknown)
	}
	response, err := s.clients.get(isLoopbackRequired).Do(request)
	if err != nil {
		return classifyTransportError(ctx, limit, err)
	}
	defer closeBody(response.Body)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return mapStatus(response)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return classifyTransportError(ctx, limit, err)
	}
	if len(body) > maxResponseBytes || json.Unmarshal(body, out) != nil {
		return connection.NewStopError(connection.ReasonUnknown)
	}
	return nil
}

func newRequest(ctx context.Context, target endpoint, base *url.URL, key string, in any) (*http.Request, error) {
	address := *base
	address.RawQuery, address.Fragment = "", ""
	address.Path = strings.TrimSuffix(address.Path, "/") + target.path
	address.RawPath = ""

	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, target.method, address.String(), body)
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

func classifyTransportError(parent, limit context.Context, err error) error {
	switch {
	case parent.Err() != nil:
		return parent.Err()
	case isAddressError(err):
		return connection.NewStopError(connection.ReasonBadRequest)
	case errors.Is(limit.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded):
		return connection.NewTryNextError(connection.ReasonTimeout, 0)
	}
	return connection.NewTryNextError(connection.ReasonUnreachable, 0)
}

func mapStatus(response *http.Response) error {
	return mapStatusCode(response.StatusCode, response.Header.Get("Retry-After"))
}

func mapStatusCode(status int, retryAfter string) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return connection.NewTryNextError(connection.ReasonKeyRejected, 0)
	case status == http.StatusPaymentRequired:
		return connection.NewTryNextError(connection.ReasonPaymentRequired, 0)
	case status == http.StatusTooManyRequests:
		return connection.NewTryNextError(connection.ReasonRateLimited, parseRetryAfter(retryAfter, time.Now()))
	case status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout:
		return connection.NewTryNextError(connection.ReasonUnreachable, 0)
	case status >= 400 && status < 500:
		return connection.NewStopError(connection.ReasonBadRequest)
	}
	return connection.NewStopError(connection.ReasonUnknown)
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

func isKnownType(typeID string) bool {
	return typeID == typeOpenRouter || typeID == typeOpenAI || typeID == typeCompatible
}
