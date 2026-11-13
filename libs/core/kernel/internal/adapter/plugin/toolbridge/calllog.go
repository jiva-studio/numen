package toolbridge

import (
	"encoding/json"
	"io"
	"os"
	"time"
	"unicode/utf8"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// maxLoggedValue is the length in bytes each argument value is cut to in the line of a call. Arguments that
// are not a JSON object are written whole as UnparsedArguments, cut the same way.
const maxLoggedValue = 200

// GetErrorOutput returns the error output of the process, which the catalog of the plugin host writes its
// calls to.
func GetErrorOutput() io.Writer { return os.Stderr }

// callRecord is one tool call as it is written to the output.
type callRecord struct {
	started  time.Time
	callerID string
	pluginID string
	call     *v1.CallToolRequest
	response *v1.CallToolResponse
	err      error
	duration time.Duration
}

// callLine is the JSON line of one tool call. Duration is in nanoseconds.
type callLine struct {
	Time                  time.Time         `json:"time"`
	Message               string            `json:"msg"`
	Caller                string            `json:"caller"`
	Tool                  string            `json:"tool"`
	Plugin                string            `json:"plugin"`
	Connection            string            `json:"connection"`
	Vault                 string            `json:"vault"`
	Arguments             map[string]string `json:"arguments,omitempty"`
	UnparsedArguments     string            `json:"unparsedArguments,omitempty"`
	Duration              time.Duration     `json:"duration"`
	ToolError             string            `json:"toolError,omitempty"`
	ResultTextBytes       *int              `json:"resultTextBytes,omitempty"`
	ResultStructuredBytes *int              `json:"resultStructuredBytes,omitempty"`
	ResultIsError         *bool             `json:"resultIsError,omitempty"`
	Error                 string            `json:"error,omitempty"`
}

func writeCall(out io.Writer, record callRecord) {
	line := callLine{
		Time: record.started, Message: "tool call", Caller: record.callerID, Tool: record.call.GetName(),
		Plugin: record.pluginID, Connection: record.call.GetConnectionId(), Vault: record.call.GetVaultId(),
		Duration: record.duration,
	}
	line.Arguments, line.UnparsedArguments = getArguments(record.call.GetArguments())
	if code := record.response.GetError(); code != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED {
		line.ToolError = code.String()
	}
	if result := record.response.GetResult(); result != nil {
		textBytes, structuredBytes, isError := len(result.GetText()), len(result.GetStructured()), result.GetIsError()
		line.ResultTextBytes, line.ResultStructuredBytes, line.ResultIsError = &textBytes, &structuredBytes, &isError
	}
	if record.err != nil {
		line.Error = record.err.Error()
	}
	encoded, err := json.Marshal(line)
	if err != nil {
		return
	}
	if _, err := out.Write(append(encoded, '\n')); err != nil {
		return
	}
}

func getArguments(arguments []byte) (map[string]string, string) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &values); err != nil {
		return nil, cutValue(string(arguments))
	}
	cut := make(map[string]string, len(values))
	for name, value := range values {
		cut[name] = cutValue(string(value))
	}
	return cut, ""
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
