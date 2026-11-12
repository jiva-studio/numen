package settings_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

func read(t *testing.T, file string) settings.Agent {
	t.Helper()
	var target settings.Agent
	if err := json.Unmarshal([]byte(file), &target); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return target
}

// A settings file that names no agent selects no agent. A file that omits the
// name defers to the default, and an empty name is an explicit choice of none.
func TestNamingNoAgentIsCached(t *testing.T) {
	var unmodified settings.Agent
	if !unmodified.IsNamed() || unmodified.GetUse() != "" {
		t.Fatalf("the defaults name %q", unmodified.GetUse())
	}

	target := read(t, `{"use": ""}`)
	if target.IsNamed() || target.GetUse() != "" {
		t.Errorf("a file naming no agent names %q", target.GetUse())
	}

	if named := read(t, `{"use": "codex"}`); named.GetUse() != "codex" || !named.IsNamed() {
		t.Errorf("the file names %q", named.GetUse())
	}
	if again := read(t, `{"use": null}`); !again.IsNamed() || again.GetUse() != "" {
		t.Errorf("a null name names %q", again.GetUse())
	}
}

// A section of the wrong type is a settings file that cannot be read.
func TestASectionOfTheWrongTypeIsRefused(t *testing.T) {
	for _, file := range []string{
		`{"claude": 5}`,
		`{"codex": "x"}`,
		`{"codex": {"command": 3}}`,
		`{"use": 5}`,
		`[]`,
	} {
		var target settings.Agent
		if err := json.Unmarshal([]byte(file), &target); err == nil {
			t.Errorf("%s was read", file)
		}
	}
}

// Every field a section names is taken over the defaults, for each program alike, whatever the
// program is called.
func TestTheFieldsASectionNamesAreTaken(t *testing.T) {
	target := read(t, `{
		"claude": {"command": ["c", "--x"], "model": "opus"},
		"anything": {"command": ["x", "y"], "model": "gpt"}
	}`)

	want := map[string]settings.AgentProgram{
		"claude":   {Command: []string{"c", "--x"}, Model: "opus"},
		"anything": {Command: []string{"x", "y"}, Model: "gpt"},
	}
	for name, program := range want {
		got := target.GetProgram(name)
		if !slices.Equal(got.Command, program.Command) || got.Model != program.Model {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

// A file read twice into one section keeps the values of the first read that the second omits.
func TestAgentUnmarshal_RepeatedReadMergesSection(t *testing.T) {
	var target settings.Agent
	for _, file := range []string{`{"claude": {"model": "opus"}}`, `{"claude": {"command": ["c"]}}`} {
		if err := json.Unmarshal([]byte(file), &target); err != nil {
			t.Fatal(err)
		}
	}
	if got := target.GetProgram("claude"); got.Model != "opus" || !slices.Equal(got.Command, []string{"c"}) {
		t.Errorf("got %+v", got)
	}
}

// The command lines the settings give are those of the programs that set one.
func TestTheCommandsAreThoseTheSettingsSet(t *testing.T) {
	target := read(t, `{"claude": {"command": ["/c"]}, "codex": {"model": "m"}, "agy": {"command": []}}`)
	got := target.GetCommands()
	if len(got) != 1 || !slices.Equal(got["claude"], []string{"/c"}) {
		t.Errorf("got %v", got)
	}
}

// The section is written with the programs next to `use`, and a choice
// left to the default is written as null, so it reads back unchanged.
func TestTheSectionIsSavedAsItIsRead(t *testing.T) {
	file := `{"use":"claude","claude":{"command":["/c"],"model":"opus"}}`
	content, err := json.Marshal(read(t, file))
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(content, &back); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(file), &want); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(back, want) {
		t.Errorf("wrote %s, read %s", content, file)
	}

	defaults, err := json.Marshal(settings.Agent{})
	if err != nil {
		t.Fatal(err)
	}
	if string(defaults) != `{"use":null}` {
		t.Errorf("the defaults are written as %s", defaults)
	}
	none, err := json.Marshal(settings.Agent{Use: new("")})
	if err != nil {
		t.Fatal(err)
	}
	if string(none) != `{"use":""}` {
		t.Errorf("no agent is written as %s", none)
	}
}

func jsonEqual(a, b map[string]any) bool {
	remaining, errLeft := json.Marshal(a)
	right, errRight := json.Marshal(b)
	return errLeft == nil && errRight == nil && string(remaining) == string(right)
}

// A section of a program holds the command and the model; what a plugin reads for itself is not kept by
// the core, and a section written back carries only the two.
func TestASectionKeepsOnlyTheCommandAndTheModel(t *testing.T) {
	target := read(t, `{"claude":{"command":["/c"],"model":"opus","reasoning":"high","reads_hooks_and_skills":true}}`)

	content, err := json.Marshal(target.GetProgram("claude"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"command":["/c"],"model":"opus"}`; string(content) != want {
		t.Errorf("wrote %s, want %s", content, want)
	}
}
