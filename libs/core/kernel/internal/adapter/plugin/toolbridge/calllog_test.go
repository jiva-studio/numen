package toolbridge_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func newCatalog(host *pluginHost, files *storage.Storage) *toolbridge.Catalog {
	return toolbridge.NewCatalog(host, host, files, testsupport.NewClock(), io.Discard)
}

// steppingClock moves a second forward on every reading.
type steppingClock struct{ clock *testsupport.ManualClock }

func (c steppingClock) Now() time.Time { return c.clock.Advance(time.Second) }

func callLogged(t *testing.T, call *v1.CallToolRequest) map[string]any {
	t.Helper()
	var out bytes.Buffer
	host := newTwoPlugins(t)
	catalog := toolbridge.NewCatalog(host, host, filesIn(t, ""), steppingClock{testsupport.NewClock()}, &out)

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
		"msg": "tool call", "caller": callerPlugin, "tool": "tree_count", "plugin": orchardPlugin,
		"vault": "vault-orchard", "duration": float64(time.Second), "resultTextBytes": float64(len("tree_count")),
		"resultStructuredBytes": float64(0), "resultIsError": false,
	}
	for key, value := range want {
		if line[key] != value {
			t.Errorf("%s is %v, want %v", key, line[key], value)
		}
	}
	arguments, isObject := line["arguments"].(map[string]any)
	if !isObject {
		t.Fatalf("the arguments are logged as %v", line["arguments"])
	}
	if arguments["row"] != "7" {
		t.Errorf("the argument row is logged as %v", arguments["row"])
	}
	note, isString := arguments["note"].(string)
	if !isString || !strings.HasPrefix(note, `"pear pear`) || !strings.HasSuffix(note, "…") || len(note) > 210 {
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
	if line["toolError"] != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL.String() {
		t.Errorf("the tool error is logged as %v", line["toolError"])
	}
}

func TestACallForAConnectionIsLoggedWithTheConnection(t *testing.T) {
	line := callLogged(t, &v1.CallToolRequest{Name: "tree_count", Arguments: []byte(`{}`), ConnectionId: "conn-orchard-keeper"})

	if line["connection"] != "conn-orchard-keeper" {
		t.Errorf("the connection is logged as %v", line["connection"])
	}
}
