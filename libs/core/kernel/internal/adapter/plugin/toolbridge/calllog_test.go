package toolbridge_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func newCatalog(host toolbridge.Host, files *storage.Storage) *toolbridge.Catalog {
	return toolbridge.NewCatalog(host, files, testsupport.NewClock(), slog.New(slog.DiscardHandler))
}

// steppingClock moves a second forward on every reading.
type steppingClock struct{ clock *testsupport.ManualClock }

func (c steppingClock) Now() time.Time { return c.clock.Advance(time.Second) }

func callLogged(t *testing.T, call *v1.CallToolRequest) map[string]any {
	t.Helper()
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	catalog := toolbridge.NewCatalog(newTwoPlugins(t), filesIn(t, ""), steppingClock{testsupport.NewClock()}, logger)

	if _, err := catalog.CallTool(t.Context(), callerPlugin, call); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("the call wrote %d log lines: %q", len(lines), out.String())
	}
	var line map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &line); err != nil {
		t.Fatal(err)
	}
	return line
}

func TestACallIsLoggedWithItsCallerToolArgumentsResultAndDuration(t *testing.T) {
	long := strings.Repeat("pear ", 100)
	line := callLogged(t, &v1.CallToolRequest{
		Name:      "tree_count",
		Arguments: []byte(`{"row":7,"note":"` + long + `"}`),
		VaultId:   "vault-orchard",
	})

	want := map[string]any{
		"level": "INFO", "msg": "tool call", "caller": callerPlugin, "tool": "tree_count", "plugin": orchardPlugin,
		"vault": "vault-orchard", "duration": float64(time.Second), "result_text_bytes": float64(len("tree_count")),
		"result_structured_bytes": float64(0), "result_is_error": false,
	}
	for key, value := range want {
		if line[key] != value {
			t.Errorf("%s is %v, want %v", key, line[key], value)
		}
	}
	arguments, _ := line["arguments"].(map[string]any)
	if arguments["row"] != "7" {
		t.Errorf("the argument row is logged as %v", arguments["row"])
	}
	note, _ := arguments["note"].(string)
	if !strings.HasPrefix(note, `"pear pear`) || !strings.HasSuffix(note, "…") || len(note) > 210 {
		t.Errorf("the long argument is logged as %q", note)
	}
	if _, hasError := line["error"]; hasError {
		t.Errorf("a call that ran logged the error %v", line["error"])
	}
}

func TestACallOfAnUnknownToolIsLoggedWithItsToolError(t *testing.T) {
	line := callLogged(t, &v1.CallToolRequest{Name: "pond_depth", Arguments: []byte(`{}`)})

	if line["tool"] != "pond_depth" || line["plugin"] != "" {
		t.Errorf("the call is logged as tool %v on plugin %v", line["tool"], line["plugin"])
	}
	if line["tool_error"] != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL.String() {
		t.Errorf("the tool error is logged as %v", line["tool_error"])
	}
}
