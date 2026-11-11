package books

import "testing"

func TestParseBaselineReadsSizeAndTime(t *testing.T) {
	got := ParseBaseline("1200", "1700000000123")

	if got == nil || got.Size != 1200 || got.Mtime != 1700000000123 {
		t.Errorf("baseline %+v, want size 1200 and mtime 1700000000123", got)
	}
}

func TestParseBaselineOfBadMtimeHasZeroMtime(t *testing.T) {
	got := ParseBaseline("1200", "soon")

	if got == nil || got.Size != 1200 || got.Mtime != 0 {
		t.Errorf("baseline %+v, want size 1200 and mtime 0", got)
	}
}

func TestParseBaselineOfNoSizeIsNil(t *testing.T) {
	for _, size := range []string{"", "many"} {
		if got := ParseBaseline(size, "5"); got != nil {
			t.Errorf("size %q: baseline %+v, want none", size, got)
		}
	}
}
