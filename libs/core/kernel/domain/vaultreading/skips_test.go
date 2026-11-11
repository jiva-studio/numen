package vaultreading_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
)

// A cancel is bound to the content hash the next pass sees, and a different
// hash ends it.
func TestCancelledSourceIsClearedByAnotherHash(t *testing.T) {
	skips := vaultreading.NewSkips()
	skips.AddVectors("Book.pdf")

	if !skips.IsVectorSkipped("Book.pdf", "aa") {
		t.Fatal("a canceled source was embedded on the next pass")
	}
	if !skips.IsVectorSkipped("Book.pdf", "aa") {
		t.Error("a canceled source with the same hash was embedded")
	}
	if skips.IsVectorSkipped("Other.pdf", "aa") {
		t.Error("a source nobody canceled was skipped")
	}
	if skips.IsVectorSkipped("Book.pdf", "bb") {
		t.Error("a canceled source whose content changed was skipped")
	}
	if skips.IsVectorSkipped("Book.pdf", "aa") {
		t.Error("a cancel outlived the change of its source")
	}

	skips.AddTexts("Scan.pdf")
	skips.AddVectors("Scan.pdf")
	skips.Clear([]string{"Scan.pdf"})
	if skips.IsTextSkipped("Scan.pdf") || skips.IsVectorSkipped("Scan.pdf", "aa") {
		t.Error("a reported change left the source canceled")
	}
}

// The count of skipped chunks is the one of the list it was counted for: a change of the list forgets
// it, and a pass that started before the change records nothing.
func TestSkippedChunksAreForgottenWhenTheListChanges(t *testing.T) {
	skips := vaultreading.NewSkips()
	skips.AddVectors("Book.pdf")
	generation := skips.GetGeneration()
	skips.SetSkippedChunks(12, generation)
	if got := skips.GetSkippedChunks(); got != 12 {
		t.Fatalf("skipped chunks = %d, want 12", got)
	}

	skips.Clear([]string{"Book.pdf"})
	if got := skips.GetSkippedChunks(); got != 0 {
		t.Errorf("skipped chunks after a clear = %d, want 0", got)
	}
	skips.SetSkippedChunks(12, generation)
	if got := skips.GetSkippedChunks(); got != 0 {
		t.Errorf("a pass that started before the clear recorded %d skipped chunks", got)
	}
}
