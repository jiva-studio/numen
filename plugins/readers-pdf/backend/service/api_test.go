package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "api", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertContract[T any](t *testing.T, name string) {
	t.Helper()
	raw := readFixture(t, name)
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s does not parse into %T: %v", name, value, err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%T encodes as %s, fixture %s holds %s", value, encoded, name, raw)
	}
}

func TestAPIContract(t *testing.T) {
	t.Run("GetDocument request", func(t *testing.T) { assertContract[GetDocumentRequest](t, "GetDocument.request.json") })
	t.Run("GetDocument response", func(t *testing.T) { assertContract[GetDocumentResponse](t, "GetDocument.response.json") })
	t.Run("GetHighlights request", func(t *testing.T) { assertContract[GetHighlightsRequest](t, "GetHighlights.request.json") })
	t.Run("GetHighlights response", func(t *testing.T) { assertContract[GetHighlightsResponse](t, "GetHighlights.response.json") })
}
