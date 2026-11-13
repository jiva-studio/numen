package toolbridge

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"time"
	"unicode/utf8"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// maxLoggedValue is the length in bytes an argument value is cut to in the log.
const maxLoggedValue = 200

// callRecord is one tool call as it is written to the log.
type callRecord struct {
	callerID string
	pluginID string
	call     *v1.CallToolRequest
	response *v1.CallToolResponse
	err      error
	duration time.Duration
}

func logCall(ctx context.Context, logger *slog.Logger, record callRecord) {
	attrs := []slog.Attr{
		slog.String("caller", record.callerID),
		slog.String("tool", record.call.GetName()),
		slog.String("plugin", record.pluginID),
		slog.String("connection", record.call.GetConnectionId()),
		slog.String("vault", record.call.GetVaultId()),
		getArgumentsAttr(record.call.GetArguments()),
		slog.Duration("duration", record.duration),
	}
	if code := record.response.GetError(); code != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED {
		attrs = append(attrs, slog.String("tool_error", code.String()))
	}
	if result := record.response.GetResult(); result != nil {
		attrs = append(attrs,
			slog.Int("result_text_bytes", len(result.GetText())),
			slog.Int("result_structured_bytes", len(result.GetStructured())),
			slog.Bool("result_is_error", result.GetIsError()),
		)
	}
	if record.err != nil {
		attrs = append(attrs, slog.String("error", record.err.Error()))
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "tool call", attrs...)
}

// getArgumentsAttr is a group of the arguments by name, each value as JSON cut to maxLoggedValue; arguments
// that are not a JSON object are logged whole, cut the same way.
func getArgumentsAttr(arguments []byte) slog.Attr {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &values); err != nil {
		return slog.String("arguments", cutValue(string(arguments)))
	}
	attrs := make([]any, 0, len(values))
	for _, name := range slices.Sorted(maps.Keys(values)) {
		attrs = append(attrs, slog.String(name, cutValue(string(values[name]))))
	}
	return slog.Group("arguments", attrs...)
}

func cutValue(value string) string {
	if len(value) <= maxLoggedValue {
		return value
	}
	cut := maxLoggedValue
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "…"
}
