package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type tallyEntry struct {
	Count int8 `json:"count"`
	Total int  `json:"total,omitempty"`
}

type tallyIn struct {
	Entries []tallyEntry `json:"entries"`
	Total   int          `json:"total,omitempty"`
	Exact   int64        `json:"exact,omitempty"`
}

func newTallyRegistry(t *testing.T, ran *tallyIn) *Registry {
	t.Helper()
	registry := New()
	if err := Add(registry, Spec{Name: "tally", Effect: ReadOnly}, func(_ context.Context, in tallyIn) (countIn, error) {
		*ran = in
		return countIn{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestAnIntegerTooLargeForItsFieldNamesTheArgument(t *testing.T) {
	var ran tallyIn
	registry := newTallyRegistry(t, &ran)

	for arguments, want := range map[string]string{
		`{"entries":[],"total":1e30}`:                                            "tally: the argument total is invalid",
		`{"entries":[{"count":1},{"count":2,"total":100000000000000000000000}]}`: "tally: the argument entries[1].total is invalid",
		`{"entries":[{"count":1},{"count":200}]}`:                                "tally: the argument entries[1].count is invalid: it must be at most 127",
	} {
		res := invoke(t, registry, "tally", []byte(arguments))
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != want {
			t.Errorf("the arguments %s were returned %+v, want the error %q", arguments, res.GetResult(), want)
		}
	}
}

func TestAnIntegerAbove2To53KeepsItsExactValue(t *testing.T) {
	var ran tallyIn
	registry := newTallyRegistry(t, &ran)

	res := invoke(t, registry, "tally", []byte(`{"entries":[],"exact":9007199254740993.0}`))

	if res.GetResult().GetIsError() || ran.Exact != 9007199254740993 {
		t.Errorf("the call returned %+v and ran with %d, want 9007199254740993", res.GetResult(), ran.Exact)
	}
}

func TestReadIntegerReadsWholeNumberLiterals(t *testing.T) {
	for literal, want := range map[string]string{
		"20": "20", "20.0": "20", "2e1": "20", "2E+1": "20", "200e-1": "20", "-0": "0", "0.0": "0",
		"-3.00": "-3", "1.5e1": "15", "0": "0",
	} {
		got, isInteger := readInteger(json.Number(literal))
		if !isInteger || got.String() != want {
			t.Errorf("readInteger(%s) = %v, %v, want %s", literal, got, isInteger, want)
		}
	}
	for _, literal := range []string{"20.5", "0.5", "2e-1", "1e1001", "-1.25"} {
		if got, isInteger := readInteger(json.Number(literal)); isInteger {
			t.Errorf("readInteger(%s) = %v, want no integer", literal, got)
		}
	}
}
