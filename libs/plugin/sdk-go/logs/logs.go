// Package logs lets a plugin backend record structured log entries in the host.
//
// Each entry carries a severity level, a human-readable message, and optional key-value properties.
// The host stamps the entry with the plugin's own id as its source, drops entries below the active
// logging.level setting, buffers accepted entries for the live log stream, and appends them to the
// daily JSONL log file.
package logs

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/internal/transport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Level is the severity of a log entry.
type Level int

const (
	// Debug is diagnostic detail for troubleshooting.
	Debug Level = iota + 1
	// Info is normal operational progress.
	Info
	// Warning is an unexpected condition that did not stop the operation.
	Warning
	// Error is a failure of an operation.
	Error
)

// Entry is one structured log record reported by a plugin backend.
//
//numen:api
type Entry struct {
	// Level is the severity of the entry; zero defaults to Info.
	Level Level
	// Message is the human-readable summary line.
	Message string
	// Properties are optional structured attributes; values are JSON-encoded strings or arbitrary values marshaled to JSON.
	Properties map[string]any
}

// Client sends structured log entries of the plugin to the host.
type Client struct {
	rpc numenv1connect.LogServiceClient
}

// NewClient returns the client of the host at the endpoint, sending its calls with the default HTTP client.
func NewClient(endpoint hostcall.Endpoint) *Client {
	return newClientOver(transport.GetHTTPClient(), endpoint)
}

func newClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewLogServiceClient(httpClient, endpoint.BaseURL, transport.BuildOptions(endpoint)...)}
}

// NewClientFromEnv returns the client of the host the process was started for. It returns
// hostcall.ErrNoHost where the environment names none.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.ReadEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(endpoint), nil
}

// Write sends entry to the host.
func (c *Client) Write(ctx context.Context, entry Entry) error {
	_, err := c.rpc.WriteLog(ctx, connect.NewRequest(&v1.WriteLogRequest{
		Level:          writeLevel(entry.Level),
		Message:        entry.Message,
		PropertiesJson: encodeProperties(entry.Properties),
	}))
	return err
}

func writeLevel(level Level) v1.LogLevel {
	switch level {
	case Debug:
		return v1.LogLevel_LOG_LEVEL_DEBUG
	case Info:
		return v1.LogLevel_LOG_LEVEL_INFO
	case Warning:
		return v1.LogLevel_LOG_LEVEL_WARNING
	case Error:
		return v1.LogLevel_LOG_LEVEL_ERROR
	}
	return v1.LogLevel_LOG_LEVEL_INFO
}

func encodeProperties(in map[string]any) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if k == "" || v == nil {
			continue
		}
		if s, ok := v.(string); ok && json.Valid([]byte(s)) {
			out[k] = s
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			continue
		}
		out[k] = string(raw)
	}
	return out
}
