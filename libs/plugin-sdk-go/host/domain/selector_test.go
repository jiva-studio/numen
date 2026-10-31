package domain

import "testing"

func TestAPageSelectorSurvivesEncodingAndDecoding(t *testing.T) {
	want := PageSelector{Page: 3, Boxes: [][4]float32{{0.1, 0.2, 0.3, 0.4}}}

	raw, err := EncodeSelector(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeSelector[PageSelector](raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Page != want.Page || len(got.Boxes) != 1 || got.Boxes[0] != want.Boxes[0] {
		t.Errorf("decoded %v, want %v", got, want)
	}
}

func TestATimeSelectorSurvivesEncodingAndDecoding(t *testing.T) {
	raw, err := EncodeSelector(TimeSelector{T: []float64{1.5, 2.5}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeSelector[TimeSelector](raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.T) != 2 || got.T[0] != 1.5 || got.T[1] != 2.5 {
		t.Errorf("decoded %v", got)
	}
}

func TestACFISelectorEncodesTheFragment(t *testing.T) {
	raw, err := EncodeSelector(CFISelector{CFI: "epubcfi(/6/4)"})
	if err != nil {
		t.Fatal(err)
	}
	if raw != `{"cfi":"epubcfi(/6/4)"}` {
		t.Errorf("encoded %s", raw)
	}
}

func TestAnEmptySelectorDecodesToNoKeys(t *testing.T) {
	got, err := DecodeSelector[TimeSelector]("")
	if err != nil || got.T != nil {
		t.Errorf("decoded %v, %v", got, err)
	}
}

func TestAMalformedSelectorFailsToDecode(t *testing.T) {
	if _, err := DecodeSelector[PageSelector]("{"); err == nil {
		t.Error("expected an error")
	}
}
