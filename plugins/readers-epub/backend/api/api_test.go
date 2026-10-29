package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// assertFixture decodes the fixture into value and checks that encoding it gives the same JSON.
func assertFixture[T any](t *testing.T, name string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "api", name))
	if err != nil {
		t.Fatal(err)
	}
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("%s: %v", name, err)
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
	wantText, _ := json.Marshal(want)
	gotText, _ := json.Marshal(got)
	if string(wantText) != string(gotText) {
		t.Errorf("%s\nfixture: %s\nencoded: %s", name, wantText, gotText)
	}
}

func TestMessages_MatchTheFixtures(t *testing.T) {
	assertFixture[GetBookRequest](t, "GetBook.request.json")
	assertFixture[GetBookResponse](t, "GetBook.response.json")
	assertFixture[ReadBookMarkupRequest](t, "ReadBookMarkup.request.json")
	assertFixture[ReadBookMarkupResponse](t, "ReadBookMarkup.response.json")
	assertFixture[ReadBookEntryRequest](t, "ReadBookEntry.request.json")
	assertFixture[ReadBookEntryResponse](t, "ReadBookEntry.response.json")
	assertFixture[GetCfiRequest](t, "GetCfi.request.json")
	assertFixture[GetCfiResponse](t, "GetCfi.response.json")
	assertFixture[FindOffsetRequest](t, "FindOffset.request.json")
	assertFixture[FindOffsetResponse](t, "FindOffset.response.json")
}
