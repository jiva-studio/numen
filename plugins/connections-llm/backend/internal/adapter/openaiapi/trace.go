package openaiapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/logs"
)

const maxLoggedBodyBytes = 2 << 10

const redacted = "[redacted]"

func keptHeaders() []string {
	return []string{"Content-Type", "X-Request-Id", "Cf-Ray", "Retry-After", "X-Ratelimit-Limit-Requests",
		"X-Ratelimit-Remaining-Requests", "X-Ratelimit-Reset-Requests", "X-Ratelimit-Limit-Tokens",
		"X-Ratelimit-Remaining-Tokens", "X-Ratelimit-Reset-Tokens"}
}

var bearerPattern = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`)

// Tracer receives one entry for a request that is sent and one for the answer or failure it gets.
type Tracer func(ctx context.Context, level logs.Level, message string, properties map[string]any)

// SetTracer sets the receiver of the request and response entries.
func (g *Gateway) SetTracer(trace Tracer) {
	g.tracer = trace
}

func (g *Gateway) trace(ctx context.Context, level logs.Level, message string, properties map[string]any) {
	if g.tracer != nil {
		g.tracer(ctx, level, message, properties)
	}
}

func (g *Gateway) traceRequest(ctx context.Context, method, path, model, key string, body []byte) {
	g.trace(ctx, logs.Info, "Connection request sent", map[string]any{
		"method": method, "path": path, "model": model, "request_bytes": len(body),
	})
	if len(body) > 0 {
		g.trace(ctx, logs.Debug, "Connection request body", map[string]any{"path": path, "body": loggedBody(body, key)})
	}
}

func (g *Gateway) traceResponse(
	ctx context.Context, path string, response *http.Response, size int, body []byte, key string, started time.Time, limit time.Duration,
) {
	properties := map[string]any{
		"path": path, "status": response.StatusCode, "response_bytes": size,
		"elapsed_ms": time.Since(started).Milliseconds(), "limit_ms": limit.Milliseconds(),
	}
	for _, name := range keptHeaders() {
		if value := response.Header.Get(name); value != "" {
			properties[strings.ToLower(name)] = redact(value, key)
		}
	}
	g.trace(ctx, logs.Info, "Connection response received", properties)
	if len(body) > 0 {
		g.trace(ctx, logs.Debug, "Connection response body", map[string]any{"path": path, "body": loggedBody(body, key)})
	}
}

func (g *Gateway) traceFailure(ctx context.Context, path, stage string, err error, status int, started time.Time, limit time.Duration) {
	g.trace(ctx, logs.Warning, "Connection request failed to complete", map[string]any{
		"path": path, "stage": stage, "failure": classifyFailure(err), "status": status,
		"elapsed_ms": time.Since(started).Milliseconds(), "limit_ms": limit.Milliseconds(),
	})
}

func classifyFailure(err error) string {
	var timeout interface{ Timeout() bool }
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &timeout) && timeout.Timeout():
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, syscall.ECONNRESET):
		return "connection_reset"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "unexpected_eof"
	case errors.Is(err, io.EOF):
		return "eof"
	}
	return "other"
}

func loggedBody(body []byte, key string) string {
	truncated := len(body) > maxLoggedBodyBytes
	if truncated {
		body = body[:maxLoggedBodyBytes]
	}
	text := redact(string(body), key)
	if truncated {
		text += "…"
	}
	return text
}

func redact(text, key string) string {
	if key != "" {
		text = strings.ReplaceAll(text, key, redacted)
	}
	return bearerPattern.ReplaceAllString(text, "${1}"+redacted)
}

func sentBody(request *http.Request) []byte {
	if request.GetBody == nil {
		return nil
	}
	reader, err := request.GetBody()
	if err != nil {
		return nil
	}
	defer closeBody(reader)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil
	}
	return data
}
