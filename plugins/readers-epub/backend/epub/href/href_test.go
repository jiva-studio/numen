package href_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/href"
)

func TestResolveEmptyHref(t *testing.T) {
	if got := href.Resolve("OEBPS", ""); got != "" {
		t.Errorf("Resolve(\"OEBPS\", \"\") = %q, want \"\"", got)
	}
}

func TestResolveWithFragmentWhenTargetResolvesToEmpty(t *testing.T) {
	if got := href.ResolveWithFragment("OEBPS", "#fragment"); got != "" {
		t.Errorf("ResolveWithFragment(\"OEBPS\", \"#fragment\") = %q, want \"\"", got)
	}
}
