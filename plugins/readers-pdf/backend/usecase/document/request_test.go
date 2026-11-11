package document

import (
	"errors"
	"testing"
)

func TestParsePageRequestReadsAGoodRequest(t *testing.T) {
	got, err := ParsePageRequest("3", "books/a.pdf", "640")
	if err != nil {
		t.Fatal(err)
	}
	if want := (PageRequest{Index: 3, Path: "books/a.pdf", PixelWidth: 640}); got != want {
		t.Errorf("request %v, want %v", got, want)
	}
}

func TestParsePageRequestRefusesWhatNamesNoPage(t *testing.T) {
	for name, args := range map[string][3]string{
		"index not a number": {"x", "a.pdf", "400"},
		"negative index":     {"-1", "a.pdf", "400"},
		"no path":            {"0", "", "400"},
		"no width":           {"0", "a.pdf", ""},
		"zero width":         {"0", "a.pdf", "0"},
		"width not a number": {"0", "a.pdf", "x"},
	} {
		_, err := ParsePageRequest(args[0], args[1], args[2])
		if failure, ok := errors.AsType[*BadRequestError](err); !ok || failure == nil {
			t.Errorf("%s: error %v, want a BadRequestError", name, err)
		}
	}
}
