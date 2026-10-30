package toolcall

import (
	"strings"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

var testWords = IndexDeclarations([]*v1.ToolDeclaration{
	{Name: "note_edit", Title: "Edit a note", Kind: "edit", About: "path", Match: "stood", Text: "becomes"},
	{Name: "note_read", Title: "Read notes", Kind: "read", About: "paths", Element: "path"},
	{Name: "vault_list", Title: "List the vault", Kind: "read"},
})

func TestADeclaredCallIsShownByItsTitleEffectAndPath(t *testing.T) {
	got := Describe(testWords, "note_edit", "c1", `{"path":"Ideas/Plan.md","start":4,"length":3}`)
	if got.GetTool() != "Edit a note" || got.GetKind() != "edit" || got.GetCall() != "c1" {
		t.Errorf("got %+v", got)
	}
	if got.GetAbout() != "Ideas/Plan.md" || got.GetPath() != "Ideas/Plan.md" {
		t.Errorf("about %q path %q", got.GetAbout(), got.GetPath())
	}
	if got.GetRange().GetStart() != 4 || got.GetRange().GetEnd() != 7 {
		t.Errorf("span %+v", got.GetRange())
	}
}

func TestACollectionIsShownByItsFirstElementAndTheCountOfTheRest(t *testing.T) {
	got := Describe(testWords, "note_read", "c", `{"paths":[{"path":"A.md"},{"path":"B.md"},{"path":"C.md"}]}`)
	if got.GetAbout() != "A.md and 2 more" || got.GetPath() != "" {
		t.Errorf("got %+v", got)
	}
}

func TestACallOfAToolThatIsAboutNothingCountsNothingSaved(t *testing.T) {
	got := Describe(testWords, "vault_list", "c", `{"depth":2}`)
	if got.GetLength() != 0 || got.GetAbout() != "" {
		t.Errorf("got %+v", got)
	}
}

func TestAnUndeclaredCallIsReadByItsCommonArguments(t *testing.T) {
	got := Describe(nil, "search", "c", `{"query":"algebra","path":"notes/math.md","start":10,"length":5}`)
	if got.GetTool() != "search" || got.GetAbout() != "algebra" || got.GetPath() != "notes/math.md" {
		t.Errorf("got %+v", got)
	}
	if got.GetRange().GetStart() != 10 || got.GetRange().GetEnd() != 15 {
		t.Errorf("span %+v", got.GetRange())
	}

	file := Describe(nil, "open", "c", `{"file":"notes/file.md"}`)
	if file.GetPath() != "notes/file.md" || file.GetAbout() != "notes/file.md" {
		t.Errorf("got %+v", file)
	}
}

func TestAnUndeclaredCallIsReadAsItArrives(t *testing.T) {
	got := Describe(nil, "search", "c", `{"query":"alge`)
	if got.GetAbout() != "alge" {
		t.Errorf("got %+v", got)
	}
	spanned := Describe(nil, "read", "c", `{"path":"a.md","start":"2","length":"3","x`)
	if spanned.GetRange().GetStart() != 2 || spanned.GetRange().GetEnd() != 5 {
		t.Errorf("span %+v", spanned.GetRange())
	}
}

func TestPartialFieldIsReadAtEveryTruncation(t *testing.T) {
	const whole = `{"path":"Note.md","stood":"abc"}`
	for i := 0; i <= len(whole); i++ {
		got := ReadPartialField(whole[:i], "stood")
		if !strings.HasPrefix("abc", got) {
			t.Errorf("%q: %q is not a prefix", whole[:i], got)
		}
	}
	if got := ReadPartialField(whole, "stood"); got != "abc" {
		t.Errorf("the whole input read %q", got)
	}
	if ReadPartialField(whole, "missing") != "" {
		t.Error("a missing field was read")
	}
}

func TestASpanOutsideTheTextIsNoSpan(t *testing.T) {
	for _, args := range []string{`{"path":"a.md","start":-4,"length":3}`, `{"path":"a.md","start":2147483647,"length":3}`} {
		if span := Describe(testWords, "note_edit", "c", args).GetRange(); span != nil {
			t.Errorf("%s: span %+v", args, span)
		}
	}
}
