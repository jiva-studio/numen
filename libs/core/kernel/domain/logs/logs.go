// Package logs holds the structured log entry model, severity levels and redaction rules of the core.
package logs

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Level is the severity of a log entry.
type Level int

const (
	// LevelDebug records detailed trace entries, including SQL queries and RPC calls.
	LevelDebug Level = iota + 1
	// LevelInfo records normal work of the application and its plugins.
	LevelInfo
	// LevelWarning records unexpected conditions that do not stop an operation.
	LevelWarning
	// LevelError records failures of an operation, a query, a call or a plugin.
	LevelError
)

// DefaultLevel is the level in force when the settings name none.
const DefaultLevel = LevelInfo

// MaxPropertyBytes is the longest JSON value kept for a single property key.
const MaxPropertyBytes = 4096

// RedactedValue is the JSON string stored in place of a secret property value.
const RedactedValue = `"[REDACTED]"`

// Sources used by the core and the windows.
const (
	SourceCore   = "core"
	SourceSQL    = "sql"
	SourceRPC    = "rpc"
	SourceWindow = "window"
)

// ErrEmptyMessage is returned when a log entry carries an empty message.
var ErrEmptyMessage = errors.New("the log entry has no message")

// String returns the canonical setting name of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarning:
		return "warning"
	case LevelError:
		return "error"
	default:
		return DefaultLevel.String()
	}
}

// ParseLevel returns the level the word names, and false for an unknown word.
func ParseLevel(word string) (Level, bool) {
	switch strings.ToLower(strings.TrimSpace(word)) {
	case "debug":
		return LevelDebug, true
	case "info":
		return LevelInfo, true
	case "warning", "warn":
		return LevelWarning, true
	case "error":
		return LevelError, true
	default:
		return 0, false
	}
}

// ResolveLevel returns the level the word names, or DefaultLevel when the word is empty or unknown.
func ResolveLevel(word string) Level {
	if level, ok := ParseLevel(word); ok {
		return level
	}
	return DefaultLevel
}

// CanRecord reports whether a log configured at l records an entry of entryLevel.
func (l Level) CanRecord(entryLevel Level) bool {
	threshold := l
	if threshold < LevelDebug || threshold > LevelError {
		threshold = DefaultLevel
	}
	target := entryLevel
	if target < LevelDebug || target > LevelError {
		target = DefaultLevel
	}
	return target >= threshold
}

// Entry is one structured log record. Each value in Properties is a JSON-encoded value.
type Entry struct {
	Time       time.Time
	Level      Level
	Source     string
	Message    string
	Properties map[string]string
}

// Normalize validates and sanitizes an entry, filling default source and level and redacting or truncating properties.
func Normalize(entry Entry) (Entry, error) {
	msg := strings.TrimSpace(entry.Message)
	if msg == "" {
		return Entry{}, ErrEmptyMessage
	}
	level := entry.Level
	if level < LevelDebug || level > LevelError {
		level = DefaultLevel
	}
	source := strings.TrimSpace(entry.Source)
	if source == "" {
		source = SourceCore
	}
	return Entry{
		Time:       entry.Time,
		Level:      level,
		Source:     source,
		Message:    msg,
		Properties: SanitizeProperties(entry.Properties),
	}, nil
}

// SanitizeProperties returns a copy of props with empty keys removed, secret keys redacted, and oversized values truncated.
func SanitizeProperties(props map[string]string) map[string]string {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]string, len(props))
	for key, val := range props {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" {
			continue
		}
		if IsSecretKey(cleanKey) {
			out[cleanKey] = RedactedValue
			continue
		}
		out[cleanKey] = truncatePropertyValue(val)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// IsSecretKey reports whether key names a secret credential or token.
func IsSecretKey(key string) bool {
	lower := strings.ToLower(key)
	for _, word := range []string{"apikey", "api_key", "api-key", "secret", "password", "token", "authorization"} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func truncatePropertyValue(val string) string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return `""`
	}
	if len(trimmed) <= MaxPropertyBytes {
		return trimmed
	}
	prefix := trimmed[:MaxPropertyBytes]
	if strings.HasPrefix(prefix, `"`) {
		if unquoted, err := strconv.Unquote(prefix + `"`); err == nil {
			return strconv.Quote(unquoted + "... [truncated]")
		}
		prefix = strings.TrimPrefix(prefix, `"`)
	}
	return strconv.Quote(prefix + "... [truncated]")
}
