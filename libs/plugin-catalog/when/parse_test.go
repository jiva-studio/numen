package when_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/when"
)

// vectors are the conditions of libs/protocol/when.json, which the TypeScript reader of the plugin
// host is tested against as well.
type vectors struct {
	Keys []struct {
		Name string `json:"name"`
	} `json:"keys"`
	Valid []struct {
		When string   `json:"when"`
		Keys []string `json:"keys"`
	} `json:"valid"`
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

// listKeys gives the keys of the tree, each once, in the order they first appear.
func listKeys(expression when.Expression) []string {
	if expression.Operator == when.OperatorKey {
		return []string{expression.Key}
	}
	var keys []string
	for _, operand := range expression.Operands {
		for _, key := range listKeys(operand) {
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func TestTheKeysAreTheOnesWhenJSONLists(t *testing.T) {
	for _, key := range readVectors(t).Keys {
		if !when.IsKey(key.Name) {
			t.Errorf("%q is not a key", key.Name)
		}
	}
	if when.IsKey("isFile") {
		t.Error("a name outside the list is taken for a key")
	}
}

func TestAValidConditionIsReadIntoATreeOfItsKeys(t *testing.T) {
	for _, one := range readVectors(t).Valid {
		expression, err := when.Parse(one.When)
		if err != nil {
			t.Errorf("%q: %v", one.When, err)
			continue
		}
		if keys := listKeys(expression); !slices.Equal(keys, one.Keys) {
			t.Errorf("%q: keys %v, want %v", one.When, keys, one.Keys)
		}
	}
}

func TestTheTreeKeepsThePrecedenceOfTheOperators(t *testing.T) {
	expression, err := when.Parse("!isRoot || isFolder && isMultiple")
	if err != nil {
		t.Fatal(err)
	}

	if expression.Operator != when.OperatorOr || expression.Operands[0].Operator != when.OperatorNot ||
		expression.Operands[1].Operator != when.OperatorAnd {
		t.Errorf("the tree came out as %+v", expression)
	}
}

func TestAnInvalidConditionIsAnError(t *testing.T) {
	for _, one := range readVectors(t).Invalid {
		if _, err := when.Parse(one.When); err == nil {
			t.Errorf("%s (%q): read without an error", one.About, one.When)
		}
	}
}
