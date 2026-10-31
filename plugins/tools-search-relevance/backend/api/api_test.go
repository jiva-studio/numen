package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

// contracts pairs each method with a decoder of its request and response messages.
var contracts = map[string]struct {
	request  func() any
	response func() any
}{
	MethodGetFloor:       {func() any { return &GetFloorRequest{} }, func() any { return &GetFloorResponse{} }},
	MethodCalibrateFloor: {func() any { return &CalibrateFloorRequest{} }, func() any { return &CalibrateFloorResponse{} }},
	MethodSetFloor:       {func() any { return &SetFloorRequest{} }, func() any { return &SetFloorResponse{} }},
	MethodProbeFloor:     {func() any { return &ProbeFloorRequest{} }, func() any { return &ProbeFloorResponse{} }},
}

// Each fixture decodes into its Go message with no unknown key, and the message encodes back to the
// same JSON.
func TestMessagesMatchFixtures(t *testing.T) {
	for method, contract := range contracts {
		for _, kind := range []struct {
			suffix string
			create func() any
		}{{"request", contract.request}, {"response", contract.response}} {
			t.Run(method+"."+kind.suffix, func(t *testing.T) {
				raw, err := os.ReadFile(filepath.Join(fixtureDir, method+"."+kind.suffix+".json"))
				if err != nil {
					t.Fatal(err)
				}
				message := kind.create()
				decoder := json.NewDecoder(bytes.NewReader(raw))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(message); err != nil {
					t.Fatal(err)
				}
				encoded, err := json.Marshal(message)
				if err != nil {
					t.Fatal(err)
				}
				assertSameJSON(t, raw, encoded)
			})
		}
	}
}

// Every method has one request fixture and one response fixture, and nothing else is in the folder.
func TestEveryMethodHasItsFixturePair(t *testing.T) {
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2*len(contracts) {
		t.Errorf("%d fixtures, want %d", len(entries), 2*len(contracts))
	}
}

func assertSameJSON(t *testing.T, want, got []byte) {
	t.Helper()
	var wantValue, gotValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	wantText, err := json.Marshal(wantValue)
	if err != nil {
		t.Fatal(err)
	}
	gotText, err := json.Marshal(gotValue)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wantText, gotText) {
		t.Errorf("encoded %s, fixture %s", gotText, wantText)
	}
}
