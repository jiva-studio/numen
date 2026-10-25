package words_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/words"
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
}
