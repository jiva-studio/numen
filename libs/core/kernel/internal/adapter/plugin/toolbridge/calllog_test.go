package toolbridge_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		"vault": "vault-orchard", "duration": float64(time.Second), "resultIsError": false,
	}
	for key, value := range want {
		if line[key] != value {
			t.Errorf("%s is %v, want %v", key, line[key], value)
		}
	}
	result, isMap := line["resultBytes"].(map[string]any)
	if !isMap || result["text"] != float64(len("tree_count")) || result["structured"] != float64(0) {
		t.Errorf("the result sizes are logged as %v", line["resultBytes"])
	}
	sizes := readArgumentBytes(t, line)
	if sizes["row"] != float64(1) || sizes["note"] != float64(len(long)+2) {
		t.Errorf("the argument sizes are logged as %v", sizes)
	}
	if strings.Contains(fmt.Sprint(line), "pear") {
		t.Errorf("the line of the call holds the text of an argument: %v", line)
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

func TestACallWithManyArgumentsLogsTheSizesOfTheFirstTwentyByName(t *testing.T) {
	fields := make([]string, 0, 25)
	for i := range 25 {
		fields = append(fields, fmt.Sprintf(`"field%02d":"plum"`, i))
	}
	line := callLogged(t, &v1.CallToolRequest{Name: "tree_count", Arguments: []byte("{" + strings.Join(fields, ",") + "}")})

	sizes := readArgumentBytes(t, line)
	if len(sizes) != 20 || sizes["field00"] != float64(len(`"plum"`)) || sizes["field20"] != nil {
		t.Errorf("the argument sizes are logged as %v", sizes)
	}
	if line["argumentsLeftOut"] != float64(5) {
		t.Errorf("the arguments left out are logged as %v", line["argumentsLeftOut"])
	}
}

func TestArgumentsThatAreNotAnObjectAreLoggedByTheirSizeAlone(t *testing.T) {
	line := callLogged(t, &v1.CallToolRequest{Name: "tree_count", Arguments: []byte(`["pear orchard"]`)})

	if line["unparsedArgumentBytes"] != float64(len(`["pear orchard"]`)) || strings.Contains(fmt.Sprint(line), "pear") {
		t.Errorf("the arguments are logged as %v", line)
	}
}

func readArgumentBytes(t *testing.T, line map[string]any) map[string]any {
	t.Helper()
	sizes, isObject := line["argumentBytes"].(map[string]any)
	if !isObject {
		t.Fatalf("the line of the call has no argument sizes: %v", line)
	}
	return sizes
}
