package operationresult_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
)

const (
	operation = "llm.rerank"
	recipe    = "recipe-one"
)

func getStart() time.Time { return time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC) }

func openIndex(t *testing.T) *index.DB {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.GetPath(t), index.DisableSync())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

func TestASaveAboveTheLimitRemovesTheLeastRecentlyUsedAndAFindKeepsARow(t *testing.T) {
	results := openIndex(t).GetOperationResults(3)
	for i, hash := range []string{"a", "b", "c"} {
		if err := results.SaveResults(t.Context(), operation, recipe, map[string][]byte{hash: {byte(i)}}, getStart().Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	found, err := results.FindResults(t.Context(), operation, recipe, []string{"a"}, getStart().Add(time.Hour))
	if err != nil || len(found) != 1 {
		t.Fatalf("FindResults = %v, %v", found, err)
	}

	if err := results.SaveResults(t.Context(), operation, recipe, map[string][]byte{"d": {3}}, getStart().Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}

	left, err := results.FindResults(t.Context(), operation, recipe, []string{"a", "b", "c", "d"}, getStart().Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if keys := slices.Sorted(maps.Keys(left)); !slices.Equal(keys, []string{"a", "c", "d"}) {
		t.Errorf("the rows left are %v, want a (found lately), c and d", keys)
	}
}

func TestAResultIsFoundForItsOwnRecipeAndOperationOnly(t *testing.T) {
	results := openIndex(t).GetOperationResults(100)
	if err := results.SaveResults(t.Context(), operation, recipe, map[string][]byte{"a": {7}}, getStart()); err != nil {
		t.Fatal(err)
	}
	if found, err := results.FindResults(t.Context(), operation, "recipe-two", []string{"a"}, getStart()); err != nil || len(found) != 0 {
		t.Errorf("another recipe found %v, %v", found, err)
	}
	if found, err := results.FindResults(t.Context(), "llm.embed", recipe, []string{"a"}, getStart()); err != nil || len(found) != 0 {
		t.Errorf("another operation found %v, %v", found, err)
	}
	if err := results.SaveResults(t.Context(), operation, recipe, map[string][]byte{"a": {8}}, getStart()); err != nil {
		t.Fatal(err)
	}
	if found, err := results.FindResults(t.Context(), operation, recipe, []string{"a"}, getStart()); err != nil || !slices.Equal(found["a"], []byte{8}) {
		t.Errorf("a save of an equal key left %v, %v", found, err)
	}
}

func TestFindResultsReadsMoreHashesThanOneBatch(t *testing.T) {
	results := openIndex(t).GetOperationResults(5000)
	outputs := map[string][]byte{}
	var hashes []string
	for i := range 1200 {
		hash := time.Duration(i).String()
		outputs[hash] = []byte{1}
		hashes = append(hashes, hash)
	}
	if err := results.SaveResults(t.Context(), operation, recipe, outputs, getStart()); err != nil {
		t.Fatal(err)
	}
	found, err := results.FindResults(t.Context(), operation, recipe, hashes, getStart())
	if err != nil || len(found) != len(hashes) {
		t.Errorf("found %d of %d, %v", len(found), len(hashes), err)
	}
}

func TestClearResultsRemovesEveryRowAndCountsThem(t *testing.T) {
	results := openIndex(t).GetOperationResults(100)
	if err := results.SaveResults(t.Context(), operation, recipe, map[string][]byte{"a": {1}, "b": {2}}, getStart()); err != nil {
		t.Fatal(err)
	}
	removed, err := results.ClearResults(t.Context())
	if err != nil || removed != 2 {
		t.Fatalf("ClearResults = %d, %v, want 2", removed, err)
	}
	if found, err := results.FindResults(t.Context(), operation, recipe, []string{"a", "b"}, getStart()); err != nil || len(found) != 0 {
		t.Errorf("after the clear %v, %v", found, err)
	}
}
