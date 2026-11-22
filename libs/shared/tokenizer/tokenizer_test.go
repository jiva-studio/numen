package tokenizer_test

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

func loadWithError(t *testing.T, path string) (tok *tokenizer.Tokenizer, err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Errorf("Load(%s) panicked: %v", path, recovered)
		}
	}()
	return tokenizer.Load(path)
}

func setMetaspace(file map[string]any, settings map[string]any) {
	metaspace := map[string]any{"type": "Metaspace", "replacement": "▁"}
	maps.Copy(metaspace, settings)
	file["pre_tokenizer"] = metaspace
}

func TestLoadRefusesAStepItDoesNotSupportAndNamesIt(t *testing.T) {
	for _, one := range []struct {
		name   string
		change func(file map[string]any)
		named  string
	}{
		{"a Lowercase normalizer", func(file map[string]any) {
			file["normalizer"] = map[string]any{"type": "Lowercase"}
		}, "Lowercase"},
		{"an NFKC normalizer inside a Sequence", func(file map[string]any) {
			file["normalizer"] = map[string]any{"type": "Sequence", "normalizers": []any{
				map[string]any{"type": "Strip", "strip_left": true, "strip_right": true},
				map[string]any{"type": "NFKC"},
			}}
		}, "NFKC"},
		{"a ByteLevel pre-tokenizer", func(file map[string]any) {
			file["pre_tokenizer"] = map[string]any{"type": "ByteLevel", "add_prefix_space": true, "trim_offsets": true, "use_regex": true}
		}, "ByteLevel"},
		{"a Punctuation pre-tokenizer inside a Sequence", func(file map[string]any) {
			file["pre_tokenizer"] = map[string]any{"type": "Sequence", "pretokenizers": []any{
				map[string]any{"type": "WhitespaceSplit"},
				map[string]any{"type": "Punctuation", "behavior": "Isolated"},
			}}
		}, "Punctuation"},
		{"a RobertaProcessing post-processor", func(file map[string]any) {
			file["post_processor"] = map[string]any{"type": "RobertaProcessing", "sep": []any{"</s>", 2}, "cls": []any{"<s>", 0}, "trim_offsets": true, "add_prefix_space": true}
		}, "RobertaProcessing"},
		{"Metaspace with prepend_scheme never", func(file map[string]any) {
			setMetaspace(file, map[string]any{"prepend_scheme": "never", "split": true})
		}, "Metaspace"},
		{"Metaspace with prepend_scheme first", func(file map[string]any) {
			setMetaspace(file, map[string]any{"prepend_scheme": "first", "split": true})
		}, "Metaspace"},
		{"Metaspace that does not split", func(file map[string]any) {
			setMetaspace(file, map[string]any{"prepend_scheme": "always", "split": false})
		}, "Metaspace"},
		{"Metaspace with add_prefix_space false", func(file map[string]any) {
			setMetaspace(file, map[string]any{"add_prefix_space": false})
		}, "Metaspace"},
		{"byte_fallback", func(file map[string]any) {
			getModel(t, file)["byte_fallback"] = true
		}, "byte_fallback"},
		{"a Regex that RE2 cannot compile", func(file map[string]any) {
			file["normalizer"] = map[string]any{"type": "Replace", "pattern": map[string]any{"Regex": "(?<=a)b"}, "content": "c"}
		}, "Replace"},
		{"a model without unk_id", func(file map[string]any) {
			delete(getModel(t, file), "unk_id")
		}, "unk_id"},
	} {
		_, err := loadWithError(t, writeVariant(t, one.change))
		if err == nil || !strings.Contains(err.Error(), one.named) {
			t.Errorf("%s: Load error %v, want one that names %s", one.name, err, one.named)
		}
	}
}

func TestLoadReturnsTheErrorOfAMissingOrMalformedFile(t *testing.T) {
	if _, err := loadWithError(t, filepath.Join(t.TempDir(), "absent.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load of an absent file = %v, want fs.ErrNotExist", err)
	}
	for name, text := range map[string]string{
		"not JSON":             "{",
		"a vocabulary of text": `{"model": {"type": "Unigram", "unk_id": 0, "vocab": "abc"}}`,
		"a score that is text": `{"model": {"type": "Unigram", "unk_id": 0, "vocab": [["a", "high"]]}}`,
	} {
		path := filepath.Join(t.TempDir(), "tokenizer.json")
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadWithError(t, path); err == nil {
			t.Errorf("Load of %s succeeded, want an error", name)
		}
	}
}

func getFixtureTexts() []string {
	// The texts cover every path of the fixture: Viterbi, ties, unknowns, Strip, Replace and Metaspace.
	return []string{"abcd", "xy", "ghk", "mnp", "ʘʘ", "ʘhkʘ", "  a  b  ", "hello world", "hello▁world", "", "abcd xy ghk  mnp ʘ"}
}

// The tokenizer holds no state that an Encode changes: 16 goroutines get the ids one goroutine gets.
// Run under -race.
func TestEncodeIsSafeForConcurrentUse(t *testing.T) {
	tok := loadFixture(t)
	fixtureTexts := getFixtureTexts()
	want := make([][]int, len(fixtureTexts))
	for i, text := range fixtureTexts {
		want[i] = tok.Encode(text)
	}
	wantPair, err := tok.EncodePair("hello hello hello", "world world", 6)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan string, 16*(len(fixtureTexts)+1))
	for range 16 {
		wg.Go(func() {
			for range 20 {
				for i, text := range fixtureTexts {
					if got := tok.Encode(text); !slices.Equal(got, want[i]) {
						errs <- "Encode(" + text + ") changed under concurrent use"
						return
					}
				}
				got, err := tok.EncodePair("hello hello hello", "world world", 6)
				if err != nil || !slices.Equal(got.IDs, wantPair.IDs) || !slices.Equal(got.TypeIDs, wantPair.TypeIDs) {
					errs <- "EncodePair changed under concurrent use"
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for one := range errs {
		t.Error(one)
	}
}
