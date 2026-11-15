package href_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/href"
)

func TestResolveEmptyHref(t *testing.T) {
	if got := href.Resolve("OEBPS", ""); got != "" {
		t.Errorf("Resolve(\"OEBPS\", \"\") = %q, want \"\"", got)
	}
	if got := href.Resolve("OEBPS", "part%201/ch%201.xhtml"); got != "OEBPS/part 1/ch 1.xhtml" {
		t.Errorf("Resolve(\"OEBPS\", \"part%%201/ch%%201.xhtml\") = %q, want \"OEBPS/part 1/ch 1.xhtml\"", got)
	}
	if got := href.Resolve("OEBPS", "100%.xhtml"); got != "OEBPS/100%.xhtml" {
		t.Errorf("Resolve(\"OEBPS\", \"100%%.xhtml\") = %q, want \"OEBPS/100%%.xhtml\"", got)
	}
}

func TestResolveWithFragmentWhenTargetResolvesToEmpty(t *testing.T) {
	if got := href.ResolveWithFragment("OEBPS", "#fragment"); got != "" {
		t.Errorf("ResolveWithFragment(\"OEBPS\", \"#fragment\") = %q, want \"\"", got)
	}
	if got := href.ResolveWithFragment("OEBPS", "chapter.xhtml#sec1"); got != "OEBPS/chapter.xhtml#sec1" {
		t.Errorf("ResolveWithFragment(\"OEBPS\", \"chapter.xhtml#sec1\") = %q, want \"OEBPS/chapter.xhtml#sec1\"", got)
	}
	if got := href.ResolveWithFragment("OEBPS", "/EPUB/chapter.xhtml"); got != "EPUB/chapter.xhtml" {
		t.Errorf("ResolveWithFragment(\"OEBPS\", \"/EPUB/chapter.xhtml\") = %q, want \"EPUB/chapter.xhtml\"", got)
	}
}
