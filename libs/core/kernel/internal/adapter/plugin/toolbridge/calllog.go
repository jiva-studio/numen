package toolbridge

import (
	"encoding/json"
	"io"
	"os"
	"slices"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// maxLoggedArguments is how many argument names the line of a call lists, in name order. The values are never
// written: an argument can hold the text of a note.
const maxLoggedArguments = 20

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

// callLine is the JSON line of one tool call. Duration is in nanoseconds. ArgumentBytes is the size of the
// JSON value of each argument; UnparsedArgumentBytes is the size of arguments that are not a JSON object.
// ResultBytes is the size of the text and of the structured value of the result.
type callLine struct {
	Time                  time.Time      `json:"time"`
	Message               string         `json:"msg"`
	Caller                string         `json:"caller"`
	Tool                  string         `json:"tool"`
	Plugin                string         `json:"plugin"`
	Connection            string         `json:"connection"`
	Vault                 string         `json:"vault"`
	ArgumentBytes         map[string]int `json:"argumentBytes,omitempty"`
	ArgumentsLeftOut      int            `json:"argumentsLeftOut,omitempty"`
	UnparsedArgumentBytes *int           `json:"unparsedArgumentBytes,omitempty"`
	Duration              time.Duration  `json:"duration"`
	ToolError             string         `json:"toolError,omitempty"`
	ResultBytes           map[string]int `json:"resultBytes,omitempty"`
	ResultIsError         *bool          `json:"resultIsError,omitempty"`
	Error                 string         `json:"error,omitempty"`
}

func writeCall(out io.Writer, record callRecord) {
	line := callLine{
		Time: record.started, Message: "tool call", Caller: record.callerID, Tool: record.call.GetName(),
		Plugin: record.pluginID, Connection: record.call.GetConnectionId(), Vault: record.call.GetVaultId(),
		Duration: record.duration,
	}
	setArguments(&line, record.call.GetArguments())
	if code := record.response.GetError(); code != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED {
		line.ToolError = code.String()
	}
	if result := record.response.GetResult(); result != nil {
		isError := result.GetIsError()
		line.ResultBytes = map[string]int{"text": len(result.GetText()), "structured": len(result.GetStructured())}
		line.ResultIsError = &isError
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

func setArguments(line *callLine, arguments []byte) {
	if len(arguments) == 0 {
		return
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &values); err != nil {
		size := len(arguments)
		line.UnparsedArgumentBytes = &size
		return
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) > maxLoggedArguments {
		line.ArgumentsLeftOut = len(names) - maxLoggedArguments
		names = names[:maxLoggedArguments]
	}
	line.ArgumentBytes = make(map[string]int, len(names))
	for _, name := range names {
		line.ArgumentBytes[name] = len(values[name])
	}
}
