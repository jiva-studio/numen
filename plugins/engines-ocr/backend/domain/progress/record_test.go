package progress

import (
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
	if _, err := rec.GetTextSegments(0); err == nil {
		t.Fatal("GetTextSegments succeeded on NaN rect")
	}
	if _, err := rec.GetText(); err == nil {
		t.Fatal("GetText succeeded on NaN rect")
	}
}

func TestRecordSurvivesEncoding(t *testing.T) {
	want := &Record{
		Path: "books/scan.pdf", InputHash: "h1", Recipe: "r1", PageCount: 2,
		Pages: [][]Part{{{Text: "first\n\n", Rects: [][4]float32{{0, 0.25, 1, 0.5}}, Section: "One"}}},
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
	if _, err := ParseRecord([]byte("{invalid-json")); err == nil {
		t.Fatal("ParseRecord accepted invalid JSON")
	}
}
