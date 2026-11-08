package when_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/when"
)

// vectors are libs/protocol/when.json, which the TypeScript evaluator of the plugin host runs as well.
type vectors struct {
	Variables []struct {
		Name string `json:"name"`
	} `json:"variables"`
	Examples []struct {
		When   string `json:"when"`
		Values []struct {
			Context map[string]any `json:"context"`
			Value   bool           `json:"value"`
		} `json:"values"`
	} `json:"examples"`
	Invalid []struct {
		When  string `json:"when"`
		About string `json:"about"`
	} `json:"invalid"`
}

func readVectors(t *testing.T) vectors {
	t.Helper()
	text, err := os.ReadFile("../../protocol/when.json")
	if err != nil {
		t.Fatal(err)
	}
	var out vectors
	if err := json.Unmarshal(text, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func parse(t *testing.T, text string) when.Condition {
	t.Helper()
	condition, err := when.Parse(text)
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return condition
}

func TestEveryVariableOfWhenJSONMayBeNamed(t *testing.T) {
	for _, variable := range readVectors(t).Variables {
		text := variable.Name + " == " + variable.Name
		if variable.Name == "config" {
			text = "config.editor.lineNumbers == true"
		}
		if _, err := when.Parse(text); err != nil {
			t.Errorf("%s: %v", variable.Name, err)
		}
	}
}

func TestTheExamplesHaveTheirValues(t *testing.T) {
	for _, example := range readVectors(t).Examples {
		condition := parse(t, example.When)
		for _, one := range example.Values {
			want := when.False
			if one.Value {
				want = when.True
			}
			if got := condition.Evaluate(one.Context); got != want {
				t.Errorf("%q over %v = %v, want %v", example.When, one.Context, got, want)
			}
		}
	}
}

func TestAnInvalidConditionIsAnError(t *testing.T) {
	for _, one := range readVectors(t).Invalid {
		if _, err := when.Parse(one.When); err == nil {
			t.Errorf("%s (%q): read without an error", one.About, one.When)
		}
	}
}

func TestAnEmptyConditionIsAlwaysTrueAndNamesNothing(t *testing.T) {
	condition := parse(t, "")

	if condition.Evaluate(nil) != when.True || len(condition.ListNames()) != 0 || condition.HasItems() {
		t.Errorf("the empty condition came out as %+v", condition)
	}
}

func TestAConditionNamesItsVariables(t *testing.T) {
	condition := parse(t, "target.isNote && window.isVaultReady || config.editor.lineNumbers == true")

	if names := condition.ListNames(); !slices.Equal(names, []string{"config", "target.isNote", "window.isVaultReady"}) {
		t.Errorf("names = %v", names)
	}
	if condition.HasItems() || !condition.HasTargetVariable() {
		t.Errorf("a condition over the target has items or names no target: %+v", condition)
	}
}

func TestAConditionOverAVersionHasItems(t *testing.T) {
	if !parse(t, "version.kind == 'ocr'").HasItems() {
		t.Error("a condition over the version has no items")
	}
}

func TestAConditionOverAValueNotGivenIsUnknown(t *testing.T) {
	condition := parse(t, "target.isNote && window.isTabOpen")

	if got := condition.Evaluate(map[string]any{"target.isNote": true}); got != when.Unknown {
		t.Errorf("with the window not given = %v, want unknown", got)
	}
	if got := condition.Evaluate(map[string]any{"target.isNote": false}); got != when.False {
		t.Errorf("false whatever the window = %v, want false", got)
	}
}

func TestAConditionIsCompiledOnceForItsText(t *testing.T) {
	const text = "target.isNote && !target.versions.exists(v, v.kind == 'recognition' && v.isLive)"
	parse(t, text)

	allocations := testing.AllocsPerRun(20, func() {
		if _, err := when.Parse(text); err != nil {
			t.Fatal(err)
		}
	})

	if allocations != 0 {
		t.Errorf("reading a condition read before allocates %v times, want it taken as compiled", allocations)
	}
}
