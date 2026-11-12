package connection_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

func TestReadRowsReturnsTheRowsOfTheSettingInTheOrderOfUse(t *testing.T) {
	values := map[string]string{
		"connections": `[{"connection":"c1f0a9e20","model":"google/gemini-2.5-flash","reasoning":"low"},{"connection":"c77b01d40","model":"qwen2.5:14b"}]`,
	}

	rows, err := connection.ReadRows(values, "connections")

	if err != nil {
		t.Fatal(err)
	}
	want := []connection.Row{
		{Connection: "c1f0a9e20", Model: "google/gemini-2.5-flash", Reasoning: "low"},
		{Connection: "c77b01d40", Model: "qwen2.5:14b"},
	}
	if !slices.Equal(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestReadRowsReturnsNoRowForASettingThatWasNotDelivered(t *testing.T) {
	rows, err := connection.ReadRows(map[string]string{"source": "local"}, "connections")

	if err != nil || len(rows) != 0 {
		t.Errorf("rows = %v, %v, want none", rows, err)
	}
}

func TestReadRowsReturnsNoRowForAnEmptyList(t *testing.T) {
	rows, err := connection.ReadRows(map[string]string{"connections": `[]`}, "connections")

	if err != nil || len(rows) != 0 {
		t.Errorf("rows = %v, %v, want none", rows, err)
	}
}

func TestReadRowsRefusesAValueThatIsNoListOfRows(t *testing.T) {
	for what, value := range map[string]string{
		"a text":               `"c1f0a9e20"`,
		"an object":            `{"connection":"c1f0a9e20","model":"m"}`,
		"text that is no JSON": `c1f0a9e20`,
		"a row that is a text": `["c1f0a9e20"]`,
	} {
		if _, err := connection.ReadRows(map[string]string{"connections": value}, "connections"); !errors.Is(err, connection.ErrInvalidRows) {
			t.Errorf("%s: err = %v, want ErrInvalidRows", what, err)
		}
	}
}
