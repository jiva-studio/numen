package progress

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestRecordIsForChecksHashAndRecipe(t *testing.T) {
	var nilRec *Record
	if nilRec.IsFor("h1", "r1") {
		t.Fatal("nil record matched")
	}
	rec := &Record{InputHash: "h1", Recipe: "r1"}
	if !rec.IsFor("h1", "r1") {
		t.Fatal("matching hash and recipe returned false")
	}
	if rec.IsFor("h2", "r1") {
		t.Fatal("mismatched hash returned true")
	}
	if rec.IsFor("h1", "r2") {
		t.Fatal("mismatched recipe returned true")
	}
}

func TestRecordGetTextPropagatesSelectorError(t *testing.T) {
	rec := &Record{
		Path:      "books/scan.pdf",
		InputHash: "h1",
		Recipe:    "r1",
		PageCount: 1,
		Pages:     [][]Part{{{Text: "bad", Rects: [][4]float32{{float32(math.NaN()), 0, 1, 1}}}}},
	}
	var unsupported *json.UnsupportedValueError
	if _, err := rec.GetTextSegments(0); !errors.As(err, &unsupported) {
		t.Fatalf("GetTextSegments on NaN rect = %v, want *json.UnsupportedValueError", err)
	}
	unsupported = nil
	if _, err := rec.GetText(); !errors.As(err, &unsupported) {
		t.Fatalf("GetText on NaN rect = %v, want *json.UnsupportedValueError", err)
	}
}

func TestRecordSurvivesEncoding(t *testing.T) {
	want := &Record{
		Path: "books/scan.pdf", InputHash: "h1", Recipe: "r1", PageCount: 2,
		Pages: [][]Part{
			{{Text: "first\n\n", Rects: [][4]float32{{0, 0.25, 1, 0.5}}, Section: "One"}},
			{{Text: "second\n\n", Rects: [][4]float32{{0.1, 0.2, 0.8, 0.9}}, Section: "Two"}},
		},
	}
	raw, err := want.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != want.Path || got.PageCount != 2 || got.Pages[0][0].Section != "One" || got.Pages[0][0].Rects[0] != want.Pages[0][0].Rects[0] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	segs0, err := got.GetTextSegments(0)
	if err != nil || len(segs0) != 1 || segs0[0].Text != "first\n\n" || segs0[0].Heading != "One" ||
		string(segs0[0].Selector) != `{"page":1,"boxes":[[0,0.25,1,0.5]]}` {
		t.Fatalf("GetTextSegments(0) = %+v, %v", segs0, err)
	}
	text, err := got.GetText()
	if err != nil {
		t.Fatal(err)
	}
	allSegs := text.GetSegments()
	if len(allSegs) != 2 || string(allSegs[1].Selector) != `{"page":2,"boxes":[[0.1,0.2,0.8,0.9]]}` {
		t.Fatalf("GetText segments = %+v", allSegs)
	}
	if _, err := ParseRecord([]byte("{invalid-json")); err == nil {
		t.Fatal("ParseRecord accepted invalid JSON")
	}
}
