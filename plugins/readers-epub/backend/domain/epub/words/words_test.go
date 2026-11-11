package words_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub/words"
)

func TestHasTokenBytesWhitespaceOnly(t *testing.T) {
	if words.HasTokenBytes([]byte("   "), "pagebreak") {
		t.Errorf("HasTokenBytes on whitespace-only bytes returned true, want false")
	}
}

func TestHasTokenBytesSingleCharacter(t *testing.T) {
	if !words.HasTokenBytes([]byte("a"), "a") {
		t.Errorf("HasTokenBytes on single character returned false, want true")
	}
	for _, sep := range []string{" ", "\t", "\n", "\r"} {
		input := []byte("first" + sep + "second" + sep + "third")
		if !words.HasTokenBytes(input, "second") {
			t.Errorf("HasTokenBytes(%q, \"second\") = false, want true", input)
		}
		if words.HasTokenBytes(input, "missing") {
			t.Errorf("HasTokenBytes(%q, \"missing\") = true, want false", input)
		}
	}
}
