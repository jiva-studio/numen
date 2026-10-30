package pdf

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"strings"
	"testing"
)

func TestOutlineWithInvalidStarts(t *testing.T) {
	doc, err := openBytes(fixture(t, "outline.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, doc)

	parts := doc.outline([]int{0})
	if len(parts) >= 3 {
		t.Errorf("got %d parts, want fewer when starts is truncated", len(parts))
	}

	emptyParts := doc.outline(nil)
	if len(emptyParts) != 0 {
		t.Errorf("got %d parts for nil starts, want 0", len(emptyParts))
	}
}

func TestRecoverPanic(t *testing.T) {
	var ans int
	var err error
	func() {
		defer recoverPanic("test operation", &ans, &err)
		panic("forced panic")
	}()
	if err == nil || !strings.Contains(err.Error(), "forced panic") {
		t.Fatalf("expected error from panic, got %v", err)
	}
}
