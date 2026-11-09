// Package plugintest holds the checks that plugin tests share: the request and response messages of a
// method, the fixtures a plugin and its frontend agree on, and the manifest as the host reads it.
package plugintest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// MessageContract creates an empty request message and an empty response message of one method.
type MessageContract struct {
	Request  func() any
	Response func() any
}

// CheckMessageFixtures fails where, for a method of contracts, the file <method>.request.json or
// <method>.response.json of dir does not decode into its message with no unknown key, or the message
// does not encode back to the same JSON.
func CheckMessageFixtures(t *testing.T, dir string, contracts map[string]MessageContract) {
	t.Helper()
	for method, contract := range contracts {
		for _, kind := range []struct {
			suffix string
			create func() any
		}{{"request", contract.Request}, {"response", contract.Response}} {
			t.Run(method+"."+kind.suffix, func(t *testing.T) {
				raw, err := os.ReadFile(filepath.Join(dir, method+"."+kind.suffix+".json"))
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
				checkSameJSON(t, raw, encoded)
			})
		}
	}
}

// CheckFixturePairs fails where dir does not hold exactly one request fixture and one response
// fixture for each method of contracts.
func CheckFixturePairs(t *testing.T, dir string, contracts map[string]MessageContract) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2*len(contracts) {
		t.Errorf("%d fixtures, want %d", len(entries), 2*len(contracts))
	}
}

func checkSameJSON(t *testing.T, want, got []byte) {
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
