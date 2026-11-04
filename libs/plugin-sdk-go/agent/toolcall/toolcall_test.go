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
	words := getTestWords()
	words["tag_list"] = Declaration{Title: "List tags", Kind: "read", About: "tags"}
	got := Describe(words, "note_read", "c", `{"paths":[{"path":"A.md"},{"path":"B.md"},{"path":"C.md"}]}`)
	if got.About != "A.md and 2 more" || got.Path != "" {
		t.Errorf("got %+v", got)
	}
	single := Describe(words, "note_read", "c", `{"paths":[{"path":"A.md"}]}`)
	if single.About != "A.md" {
		t.Errorf("single element = %q, want A.md", single.About)
	}
	stringsList := Describe(words, "tag_list", "c", `{"tags":["alpha","beta"]}`)
	if stringsList.About != "alpha and 1 more" {
		t.Errorf("strings collection = %q, want alpha and 1 more", stringsList.About)
	}
	for _, emptyArgs := range []string{
		`{"paths":[]}`,
		`{"paths":[{"path":""},{"path":"B.md"}]}`,
		`{"paths":[42]}`,
		`{"paths":"A.md"}`,
		`{"paths":123}`,
	} {
		res := Describe(words, "note_read", "c", emptyArgs)
		if emptyArgs == `{"paths":"A.md"}` {
			if res.About != "A.md" {
				t.Errorf("%s: got %q, want A.md", emptyArgs, res.About)
			}
			continue
		}
		if res.About != "" {
			t.Errorf("%s: got %q, want empty", emptyArgs, res.About)
		}
	}
	if noInside := Describe(words, "tag_list", "c", `{"tags":[{"":"alpha"}]}`); noInside.About != "" {
		t.Errorf("object without Element declaration = %q, want empty", noInside.About)
	}
	if partialElement := Describe(words, "note_read", "c", `{"paths":[{"path":"A.m`); partialElement.About != "A.m" {
		t.Errorf("partial element = %q, want A.m", partialElement.About)
	}
	if partialAbout := Describe(words, "note_edit", "c", `{"path":"Ideas/Pa`); partialAbout.About != "Ideas/Pa" || partialAbout.Range != nil {
		t.Errorf("partial about = %+v", partialAbout)
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
	raw := Describe(nil, "ping", "c", `{"count":1}`)
	if raw.About != `{"count":1}` || raw.Path != "" || raw.Range != nil {
		t.Errorf("raw fallback = %+v", raw)
	}
	zeroSpan := Describe(nil, "read", "c", `{"path":"a.md","start":0,"length":0}`)
	if zeroSpan.Range == nil || zeroSpan.Range.Start != 0 || zeroSpan.Range.End != 0 {
		t.Errorf("zero span = %+v", zeroSpan.Range)
	}
	maxSpan := Describe(nil, "read", "c", `{"path":"a.md","start":2147483644,"length":3}`)
	if maxSpan.Range == nil || maxSpan.Range.Start != 2147483644 || maxSpan.Range.End != 2147483647 {
		t.Errorf("max span = %+v", maxSpan.Range)
	}
	for _, noSpan := range []string{
		`{"path":"a.md","start":5}`,
		`{"path":"a.md","length":0}`,
		`{"path":"a.md","start":-1,"length":2}`,
		`{"path":"a.md","start":2,"length":-1}`,
		`{"path":"a.md","start":2147483645,"length":3}`,
	} {
		if got := Describe(nil, "read", "c", noSpan); got.About != "a.md" || got.Path != "a.md" || got.Range != nil {
			t.Errorf("%s: got = %+v, want path a.md with nil span", noSpan, got)
		}
	}
}

func TestAnUndeclaredCallIsReadAsItArrives(t *testing.T) {
	if empty := Describe(nil, "search", "c", ""); empty.About != "" || empty.Path != "" || empty.Range != nil {
		t.Errorf("empty = %+v", empty)
	}
	got := Describe(nil, "search", "c", `{"query":"alge`)
	if got.About != "alge" {
		t.Errorf("got %+v", got)
	}
	if byAbout := Describe(nil, "search", "c", `{"about":"summ`); byAbout.About != "summ" {
		t.Errorf("byAbout = %+v", byAbout)
	}
	if byTitle := Describe(nil, "search", "c", `{"title":"head`); byTitle.About != "head" {
		t.Errorf("byTitle = %+v", byTitle)
	}
	if byFile := Describe(nil, "open", "c", `{"file":"notes/f.md","length":"5`); byFile.Path != "notes/f.md" || byFile.About != "notes/f.md" || byFile.Range != nil {
		t.Errorf("byFile = %+v", byFile)
	}
	spanned := Describe(nil, "read", "c", `{"path":"a.md","start":"0","length":"0","x`)
	if spanned.Range == nil || spanned.Range.Start != 0 || spanned.Range.End != 0 {
		t.Errorf("span %+v", spanned.Range)
	}
	maxSpanned := Describe(nil, "read", "c", `{"path":"a.md","start":"2147483644","length":"3","x`)
	if maxSpanned.Range == nil || maxSpanned.Range.Start != 2147483644 || maxSpanned.Range.End != 2147483647 {
		t.Errorf("maxSpanned %+v", maxSpanned.Range)
	}
	for _, invalid := range []string{
		`{"path":"a.md","start":"19","length":"bad`,
		`{"path":"a.md","start":"-1","length":"5`,
		`{"path":"a.md","start":"1","length":"-1`,
		`{"path":"a.md","start":"2147483645","length":"3`,
	} {
		if got := Describe(nil, "read", "c", invalid); got.Path != "a.md" || got.About != "a.md" || got.Range != nil {
			t.Errorf("%s: got %+v", invalid, got)
		}
	}
}

func TestPartialFieldIsReadAtEveryTruncation(t *testing.T) {
	const whole = `{"path":"Note.md","stood":"a\u0042€🌟\"b\\c"}`
	const want = "aB€🌟\"b\\c"
	for i := 0; i <= len(whole); i++ {
		got := ReadPartialField(whole[:i], "stood")
		if !strings.HasPrefix(want, got) {
			t.Errorf("%q: %q is not a prefix of %q", whole[:i], got, want)
		}
	}
	if got := ReadPartialField(whole, "stood"); got != want {
		t.Errorf("the whole input read %q, want %q", got, want)
	}
	if got := ReadPartialField(`{"stood":"a\"`, "stood"); got != `a"` {
		t.Errorf("trailing escaped quote read %q, want a\"", got)
	}
	if got := ReadPartialField(`{"stood":"a`+"\xe2", "stood"); got != "a" {
		t.Errorf("prefix before partial rune read %q, want a", got)
	}
	if got := ReadPartialField(`{"stood":"a\u00`, "stood"); got != "a" {
		t.Errorf("prefix before partial unicode escape read %q, want a", got)
	}
	if got := ReadPartialField(`"stood" 	
: 	
"abc"`, "stood"); got != "abc" {
		t.Errorf("spaced field read %q, want abc", got)
	}
	if got := ReadPartialField(`{"stood":"`+"\xe2\x82", "stood"); got != "" {
		t.Errorf("only partial rune read %q, want empty", got)
	}
	if ReadPartialField(`{"":"abc"}`, "") != "" {
		t.Error("an empty field name was read")
	}
	if ReadPartialField(whole, "missing") != "" {
		t.Error("a missing field was read")
	}
}

func TestASpanOutsideTheTextIsNoSpan(t *testing.T) {
	validMin := Describe(getTestWords(), "note_edit", "c", `{"path":"a.md","start":0,"length":1}`)
	if validMin.Range == nil || validMin.Range.Start != 0 || validMin.Range.End != 1 {
		t.Errorf("validMin span = %+v", validMin.Range)
	}
	validMax := Describe(getTestWords(), "note_edit", "c", `{"path":"a.md","start":2147483644,"length":3}`)
	if validMax.Range == nil || validMax.Range.Start != 2147483644 || validMax.Range.End != 2147483647 {
		t.Errorf("validMax span = %+v", validMax.Range)
	}
	for _, args := range []string{
		`{"path":"a.md","start":0,"length":0}`,
		`{"path":"a.md","start":4,"length":-1}`,
		`{"path":"a.md","start":-1,"length":3}`,
		`{"path":"a.md","start":2147483645,"length":3}`,
	} {
		if span := Describe(getTestWords(), "note_edit", "c", args).Range; span != nil {
			t.Errorf("%s: span %+v", args, span)
		}
	}
}
