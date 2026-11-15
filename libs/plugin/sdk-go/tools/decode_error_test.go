package tools

import (
	"context"
	"encoding/json"
	"strings"
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
	nines1000 := strings.Repeat("9", 1000)
	zeros999 := strings.Repeat("0", 999)
	zeros1000 := strings.Repeat("0", 1000)
	for literal, want := range map[string]string{
		"20": "20", "20.0": "20", "2e1": "20", "2E+1": "20", "200e-1": "20", "-0": "0", "0.0": "0",
		"-3.00": "-3", "1.5e1": "15", "0": "0", "0e-5": "0",
		"1e999": "1" + zeros999, "0.1e1000": "1" + zeros999,
		"1" + zeros1000 + "e-1000": "1", zeros1000 + "5": "5",
		nines1000: nines1000, "-" + nines1000: "-" + nines1000,
	} {
		got, isInteger := readInteger(json.Number(literal))
		if !isInteger || got == nil || got.String() != want {
			t.Errorf("readInteger(%s) = %v, %v, want %s", literal, got, isInteger, want)
		}
	}
	for _, literal := range []string{
		"20.5", "0.5", "2e-1", "1e1000", "10e999", "1e1001", "0.01e1001",
		"1" + strings.Repeat("0", 1001) + "e-1001",
		"1e9223372036854775807", "1e-9223372036854775808",
		"-1.25", "1e+bad", strings.Repeat("9", 1001),
	} {
		if got, isInteger := readInteger(json.Number(literal)); isInteger || got != nil {
			t.Errorf("readInteger(%s) = %v, %v, want nil, false", literal, got, isInteger)
		}
	}
	if got := formatLimit(1.5); got != "1.5" {
		t.Errorf("formatLimit(1.5) = %q, want 1.5", got)
	}
	if got := describeInvalid("", "too short").Error(); got != "the arguments are invalid: too short" {
		t.Errorf("describeInvalid empty path = %q", got)
	}
	if got := describeDecodeError([]byte("{}"), &json.UnmarshalTypeError{Offset: 1}).Error(); got != "the arguments are invalid" {
		t.Errorf("describeDecodeError root = %q", got)
	}
	if got := describeDecodeError([]byte("{}"), &json.UnmarshalTypeError{Offset: 99}).Error(); got != "the arguments are invalid" {
		t.Errorf("describeDecodeError past EOF = %q", got)
	}
	if path, ok := findPathAt([]byte(`{"a":1}`), 4); ok || path != "" {
		t.Errorf("findPathAt key = %q, %v, want empty, false", path, ok)
	}
}
