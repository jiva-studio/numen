package cardtools

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards/deckedit"
)

func TestCardToolsHelpersAndBounds(t *testing.T) {
	if maxCardModels != 50 || maxCards != 50 {
		t.Fatalf("unexpected limits: %d, %d", maxCardModels, maxCards)
	}
	if got := countBytes([]FieldValue{{Field: "ab", Text: "cde"}, {Field: "f", Text: "gh"}}); got != 8 {
		t.Fatalf("expected 8 bytes, got %d", got)
	}
	half := maxBytes / 2
	if err := checkCardSize([]FieldValue{{Field: strings.Repeat("a", half), Text: strings.Repeat("b", maxBytes-half)}}); err != nil {
		t.Fatalf("expected exact maxBytes to pass, got %v", err)
	}
	if err := checkCardSize([]FieldValue{{Field: "a", Text: strings.Repeat("b", maxBytes)}}); err == nil {
		t.Fatal("expected maxBytes+1 to fail")
	}

	stat := domain.FileStat{Size: 123, ModTime: time.Unix(0, 456)}
	out := newWriteOutcome("Deck.md", deckedit.EditResult{FileStat: stat, Cards: 7})
	if out.Path != "Deck.md" || out.Total != 7 || out.Fingerprint != "123-456" {
		t.Fatalf("unexpected WriteOutcome: %+v", out)
	}

	parsed, err := parseFingerprint("123-456")
	if err != nil || parsed.Size != 123 || parsed.ModTime.UnixNano() != 456 {
		t.Fatalf("unexpected parsed stat: %+v, err: %v", parsed, err)
	}
	for _, invalid := range []string{"", "nohyphen", "bad-456", "123-bad"} {
		if _, err := parseFingerprint(invalid); err == nil {
			t.Fatalf("expected error for fingerprint %q", invalid)
		}
	}

	if code, ok := convertOutcomeToErrorCode(edit.Ok); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("expected Ok -> Unspecified/false, got %v/%v", code, ok)
	}
	if code, ok := convertOutcomeToErrorCode(edit.ReadOutcome("unknown")); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("expected unknown -> Unspecified/false, got %v/%v", code, ok)
	}
	if got := describeCode(api.ErrorCode("CUSTOM")); got != "CUSTOM" {
		t.Fatalf("expected fallback description CUSTOM, got %q", got)
	}
}
