package pdf

import (
	"strings"
	"testing"
)

func TestOutlineWithInvalidStarts(t *testing.T) {
	doc, err := newWorkers(t).openBytes(fixture(t, "outline.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, doc)

	parts := doc.outline([]int{0})
	if len(parts) >= 3 {
		t.Errorf("got %d parts, want fewer when starts is truncated", len(parts))
	}

	emptyParts := doc.outline(nil)
	if len(emptyParts) != 0 {
		t.Errorf("got %d parts for nil starts, want 0", len(emptyParts))
	}
}

func pickElement(values []int, index int) int { return values[index] }

func TestRecoverPanic(t *testing.T) {
	var ans int
	var err error
	func() {
		defer recoverPanic("test operation", &ans, &err)
		ans = pickElement(nil, 3)
	}()
	if err == nil || !strings.Contains(err.Error(), "index out of range") {
		t.Fatalf("expected error from panic, got %v", err)
	}
}
