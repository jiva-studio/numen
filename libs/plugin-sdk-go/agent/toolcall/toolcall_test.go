package toolcall

import (
	"strings"
	"testing"
)

func getTestWords() map[string]Declaration {
	return IndexDeclarations([]ToolDeclaration{
		{Name: "note_edit", Title: "Edit a note", Kind: "edit", About: "path", Match: "stood", Text: "becomes"},
		{Name: "note_read", Title: "Read notes", Kind: "read", About: "paths", Element: "path"},
		{Name: "vault_list", Title: "List the vault", Kind: "read"},
	})
}

func TestADeclaredCallIsShownByItsTitleEffectAndPath(t *testing.T) {
	got := Describe(getTestWords(), "note_edit", "c1", `{"path":"Ideas/Plan.md","start":4,"length":3}`)
	if got.Tool != "Edit a note" || got.Kind != "edit" || got.Call != "c1" {
		t.Errorf("got %+v", got)
	}
	if got.About != "Ideas/Plan.md" || got.Path != "Ideas/Plan.md" {
		t.Errorf("about %q path %q", got.About, got.Path)
	}
	if got.Range.Start != 4 || got.Range.End != 7 {
		t.Errorf("span %+v", got.Range)
	}
}

func TestACollectionIsShownByItsFirstElementAndTheCountOfTheRest(t *testing.T) {
	got := Describe(getTestWords(), "note_read", "c", `{"paths":[{"path":"A.md"},{"path":"B.md"},{"path":"C.md"}]}`)
	if got.About != "A.md and 2 more" || got.Path != "" {
		t.Errorf("got %+v", got)
	}
}

func TestACallOfAToolThatIsAboutNothingCountsNothingSaved(t *testing.T) {
	got := Describe(getTestWords(), "vault_list", "c", `{"depth":2}`)
	if got.Length != 0 || got.About != "" {
		t.Errorf("got %+v", got)
	}
}

func TestAnUndeclaredCallIsReadByItsCommonArguments(t *testing.T) {
	got := Describe(nil, "search", "c", `{"query":"algebra","path":"notes/math.md","start":10,"length":5}`)
	if got.Tool != "search" || got.About != "algebra" || got.Path != "notes/math.md" {
		t.Errorf("got %+v", got)
	}
	if got.Range.Start != 10 || got.Range.End != 15 {
		t.Errorf("span %+v", got.Range)
	}

	file := Describe(nil, "open", "c", `{"file":"notes/file.md"}`)
	if file.Path != "notes/file.md" || file.About != "notes/file.md" {
		t.Errorf("got %+v", file)
	}
}

func TestAnUndeclaredCallIsReadAsItArrives(t *testing.T) {
	got := Describe(nil, "search", "c", `{"query":"alge`)
	if got.About != "alge" {
		t.Errorf("got %+v", got)
	}
	spanned := Describe(nil, "read", "c", `{"path":"a.md","start":"2","length":"3","x`)
	if spanned.Range.Start != 2 || spanned.Range.End != 5 {
		t.Errorf("span %+v", spanned.Range)
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
		if span := Describe(getTestWords(), "note_edit", "c", args).Range; span != nil {
			t.Errorf("%s: span %+v", args, span)
		}
	}
}
