package service

import (
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

// initializeFixture is the answer of Claude Code 2.1.303 to the initialize request, cut to its models.
const initializeFixture = "../model/testdata/initialize-response.json"

func readFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(initializeFixture)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The program receives the request over stdin and its answer carries the request id.
func TestModels_QueriesProgramForModels(t *testing.T) {
	request := t.TempDir() + "/asked"
	answer := strings.Replace(readFixture(t), `"r1"`, `"numen-models"`, 1)
	answer = strings.ReplaceAll(answer, "'", "")
	script := writeScript(t, "claude", "read line\nprintf '%s\\n' \"$line\" > "+request+"\ncat <<'EOF'\n"+answer+"EOF\nsleep 30\n")
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	client := startAgent(t, newTestAgent(t))

	got, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	list := got[0].Models
	if len(list) != 13 {
		t.Fatalf("the models are %v", list)
	}
	raw, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"subtype":"initialize"`) {
		t.Errorf("the program was asked %s", raw)
	}
}

func TestGetModels_FallbackWhenListFails(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", "exit 1\n"))
	client := startAgent(t, newTestAgent(t))

	got, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for place, one := range got[0].Models {
		if one.Name != model.GetFallback()[place].Value || len(one.ReasoningLevels) != 0 {
			t.Errorf("the model %v is not a fallback alias", one)
		}
	}
}
