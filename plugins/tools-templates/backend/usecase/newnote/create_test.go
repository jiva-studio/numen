package newnote_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/mask"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/usecase/newnote"
)

const (
	templateID = "01J0TEMPLATE0000000000000"
	meeting    = "---\nid: " + templateID + "\ntype: meeting\ntags:\n  - weekly\ncreated: \"{{ date \"YYYY-MM-DD\" }}\"\nupdated: {{ date \"DD.MM.YYYY\" }}\n---\n# {{ .title }}\n\nAttendees\n"
	book       = "# {{ .title }}\n\nAuthor\n"
)

func getToday() time.Time { return time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC) }

func newClock(at time.Time) *clocktest.Fake {
	source := new(clocktest.Fake)
	source.Set(at)
	return source
}

func newVault(templates map[string]string) *hosttest.Vault {
	vault := hosttest.New("notes")
	for path, content := range templates {
		vault.File(path, content)
	}
	return vault
}

func newUseCase(vault newnote.Vault, source newnote.Clock) newnote.CreateFromTemplate {
	return newnote.NewCreateFromTemplate(vault, source, ulid.NewGenerator(rand.Reader), func() mask.Set { return mask.Parse("") })
}

func createNote(t *testing.T, vault *hosttest.Vault, template, folder string) string {
	t.Helper()
	made, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: template, Folder: folder})
	if err != nil {
		t.Fatalf("Execute(%q, %q): %v", template, folder, err)
	}
	return made.Path
}

func getContent(t *testing.T, vault *hosttest.Vault, path string) string {
	t.Helper()
	content, isThere := vault.Content(path)
	if !isThere {
		t.Fatalf("no file at %q", path)
	}
	return content
}

func openNote(t *testing.T, content string) *markdown.Document {
	t.Helper()
	doc, err := markdown.Open([]byte(content))
	if err != nil {
		t.Fatalf("the note does not open: %v\n%s", err, content)
	}
	return doc
}

var topLevelKey = regexp.MustCompile(`(?m)^([A-Za-z][A-Za-z0-9_-]*):`)

func getKeys(t *testing.T, content string) []string {
	t.Helper()
	front, _, found := strings.Cut(strings.TrimPrefix(content, "---\n"), "\n---\n")
	if !found {
		t.Fatalf("no frontmatter in %q", content)
	}
	var keys []string
	for _, match := range topLevelKey.FindAllStringSubmatch(front, -1) {
		keys = append(keys, match[1])
	}
	return keys
}

func TestMakeTheNoteFromTheTemplate(t *testing.T) {
	t.Run("makes a note whose body is the template after substitution", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		path := createNote(t, vault, "Templates/Meeting.md", "work")

		if path != "work/Meeting.md" {
			t.Fatalf("path = %q", path)
		}
		if got := openNote(t, getContent(t, vault, path)).Body(); got != "# Meeting\n\nAttendees\n" {
			t.Errorf("body = %q", got)
		}
	})
	t.Run("keeps the frontmatter keys of the template in their order", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		path := createNote(t, vault, "Templates/Meeting.md", "work")

		if got, want := getKeys(t, getContent(t, vault, path)), []string{"id", "type", "tags", "created", "updated"}; !slices.Equal(got, want) {
			t.Errorf("keys = %v, want %v", got, want)
		}
		if !strings.Contains(getContent(t, vault, path), "type: meeting\ntags:\n  - weekly\n") {
			t.Errorf("the type and tags changed:\n%s", getContent(t, vault, path))
		}
	})
	t.Run("substitutes variables inside the frontmatter", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		path := createNote(t, vault, "Templates/Meeting.md", "work")

		content := getContent(t, vault, path)
		if !strings.Contains(content, "created: \"2026-11-14\"\n") || !strings.Contains(content, "updated: 14.11.2026\n") {
			t.Errorf("frontmatter is not substituted:\n%s", content)
		}
	})
	t.Run("makes the note at the vault root for an empty folder", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		if path := createNote(t, vault, "Templates/Meeting.md", ""); path != "Meeting.md" {
			t.Errorf("path = %q", path)
		}
	})
	t.Run("reads a template of a subfolder of the templates folder", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meetings/Weekly.md": book})

		if path := createNote(t, vault, "Templates/Meetings/Weekly.md", "work"); path != "work/Weekly.md" {
			t.Errorf("path = %q", path)
		}
	})
	t.Run("scans the new note into the index", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		path := createNote(t, vault, "Templates/Meeting.md", "work")

		if got := vault.Scans(); len(got) != 1 || !slices.Equal(got[0], []string{path}) {
			t.Errorf("scans = %v", got)
		}
	})
	t.Run("reads a template named by a mask anywhere in the vault", func(t *testing.T) {
		vault := newVault(map[string]string{"Notes/Work/Weekly.template.md": book})

		if path := createNote(t, vault, "Notes/Work/Weekly.template.md", "work"); path != "work/Weekly.md" {
			t.Errorf("path = %q", path)
		}
	})
	t.Run("follows the masks setting at each call", func(t *testing.T) {
		vault := newVault(map[string]string{"Models/Meeting.md": book})
		masks := mask.Parse("")
		useCase := newnote.NewCreateFromTemplate(vault, newClock(getToday()), ulid.NewGenerator(rand.Reader), func() mask.Set { return masks })
		request := newnote.Request{Template: "Models/Meeting.md", Folder: ""}

		_, before := useCase.Execute(t.Context(), request)
		masks = mask.Parse("Models/**")
		made, after := useCase.Execute(t.Context(), request)

		if !errors.Is(before, newnote.ErrNotATemplate) || after != nil || made.Path != "Meeting.md" {
			t.Errorf("before = %v, after = %v, path = %q", before, after, made.Path)
		}
	})
}

func TestAssignTheNoteItsOwnID(t *testing.T) {
	t.Run("assigns the new note an id other than the template's", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		path := createNote(t, vault, "Templates/Meeting.md", "work")

		id, hasID := openNote(t, getContent(t, vault, path)).Identifier()
		if !hasID || id == "" || id == templateID || len(id) != 26 {
			t.Errorf("id = %q (%v), the template's is %q", id, hasID, templateID)
		}
	})
	t.Run("assigns an id to a template without frontmatter", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Book.md": book})

		path := createNote(t, vault, "Templates/Book.md", "reading")

		doc := openNote(t, getContent(t, vault, path))
		if id, hasID := doc.Identifier(); !hasID || len(id) != 26 {
			t.Errorf("id = %q (%v)", id, hasID)
		}
		if doc.Body() != "# Book\n\nAuthor\n" {
			t.Errorf("body = %q", doc.Body())
		}
	})
	t.Run("assigns two notes from one template two ids", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

		first := createNote(t, vault, "Templates/Meeting.md", "work")
		second := createNote(t, vault, "Templates/Meeting.md", "work")

		firstID, _ := openNote(t, getContent(t, vault, first)).Identifier()
		secondID, _ := openNote(t, getContent(t, vault, second)).Identifier()
		if firstID == secondID {
			t.Errorf("both notes have the id %q", firstID)
		}
	})
}

func TestNameTheNote(t *testing.T) {
	t.Run("names the note after the template", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Book review.md": book})

		if path := createNote(t, vault, "Templates/Book review.md", "reading"); path != "reading/Book review.md" {
			t.Errorf("path = %q", path)
		}
	})
	t.Run("strips .template from the name", func(t *testing.T) {
		vault := newVault(map[string]string{"Notes/Book review.template.md": book})

		path := createNote(t, vault, "Notes/Book review.template.md", "reading")

		if path != "reading/Book review.md" {
			t.Fatalf("path = %q", path)
		}
		if got := openNote(t, getContent(t, vault, path)).Body(); got != "# Book review\n\nAuthor\n" {
			t.Errorf("body = %q", got)
		}
	})
	t.Run("keeps a name that is .template alone", func(t *testing.T) {
		vault := newVault(map[string]string{"Notes/.template.md": book})

		if path := createNote(t, vault, "Notes/.template.md", "work"); path != "work/.template.md" {
			t.Errorf("path = %q", path)
		}
	})
	t.Run("uses that name as the title variable", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Book review.md": book})

		path := createNote(t, vault, "Templates/Book review.md", "reading")

		if got := openNote(t, getContent(t, vault, path)).Body(); got != "# Book review\n\nAuthor\n" {
			t.Errorf("body = %q", got)
		}
	})
	t.Run("numbers the name when it is taken", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": book, "work/Meeting.md": "taken", "work/Meeting 2.md": "taken"})

		path := createNote(t, vault, "Templates/Meeting.md", "work")

		if path != "work/Meeting 3.md" {
			t.Fatalf("path = %q", path)
		}
		if got := openNote(t, getContent(t, vault, path)).Body(); got != "# Meeting 3\n\nAuthor\n" {
			t.Errorf("body = %q", got)
		}
		if content, _ := vault.Content("work/Meeting.md"); content != "taken" {
			t.Errorf("a taken note changed: %q", content)
		}
	})
	t.Run("tries the next name when the file appears between check and create", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": book})
		racing := &racingVault{Vault: vault, pending: []string{"work/Meeting.md"}}

		made, err := newUseCase(racing, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Meeting.md", Folder: "work"})

		if err != nil || made.Path != "work/Meeting 2.md" {
			t.Fatalf("path = %q, error = %v", made.Path, err)
		}
		if content, _ := vault.Content("work/Meeting.md"); content != "written by another window" {
			t.Errorf("the note of the other writer changed: %q", content)
		}
	})
	t.Run("uses the last number of the hundred", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": book})
		takeNames(vault, "work", "Meeting", 99)

		if path := createNote(t, vault, "Templates/Meeting.md", "work"); path != "work/Meeting 100.md" {
			t.Errorf("path = %q", path)
		}
	})
	t.Run("fails after 100 taken names", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": book})
		takeNames(vault, "work", "Meeting", 100)

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Meeting.md", Folder: "work"})

		if !errors.Is(err, newnote.ErrNoFreeName) || !strings.Contains(err.Error(), "Meeting") {
			t.Errorf("error = %v", err)
		}
		if content, isThere := vault.Content("work/Meeting 101.md"); isThere {
			t.Errorf("a 101st note was made: %q", content)
		}
		if len(vault.Scans()) != 0 {
			t.Errorf("scans = %v", vault.Scans())
		}
	})
}

func takeNames(vault *hosttest.Vault, folder, stem string, count int) {
	vault.File(folder+"/"+stem+".md", "taken")
	for number := 2; number <= count; number++ {
		vault.File(fmt.Sprintf("%s/%s %d.md", folder, stem, number), "taken")
	}
}

type racingVault struct {
	*hosttest.Vault
	pending []string
}

func (v *racingVault) Create(ctx context.Context, path string, content []byte) error {
	if i := slices.Index(v.pending, path); i >= 0 {
		v.pending = slices.Delete(v.pending, i, i+1)
		v.Vault.File(path, "written by another window")
	}
	return v.Vault.Create(ctx, path, content)
}

func TestWriteTheDateTheClockReports(t *testing.T) {
	t.Run("writes the date the clock reports", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Daily.md": "# {{ .title }} {{ date \"YYYY-MM-DD\" }} {{ time \"HH:mm\" }}\n"})
		source := newClock(time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC))
		useCase := newUseCase(vault, source)
		request := newnote.Request{Template: "Templates/Daily.md", Folder: "journal"}

		first, err := useCase.Execute(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		source.Set(time.Date(2027, 2, 3, 21, 45, 0, 0, time.UTC))
		second, err := useCase.Execute(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}

		if got := openNote(t, getContent(t, vault, first.Path)).Body(); got != "# Daily 2026-11-14 09:05\n" {
			t.Errorf("first body = %q", got)
		}
		if got := openNote(t, getContent(t, vault, second.Path)).Body(); got != "# Daily 2 2027-02-03 21:45\n" {
			t.Errorf("second body = %q", got)
		}
	})
}

func TestLeaveTheTemplateAsItWas(t *testing.T) {
	t.Run("keeps the template file as it was", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": meeting})
		before, err := vault.StatFile(t.Context(), "Templates/Meeting.md")
		if err != nil {
			t.Fatal(err)
		}

		createNote(t, vault, "Templates/Meeting.md", "work")

		if content, _ := vault.Content("Templates/Meeting.md"); content != meeting {
			t.Errorf("the template changed:\n%s", content)
		}
		if after, err := vault.StatFile(t.Context(), "Templates/Meeting.md"); err != nil || !after.IsUnchanged(before) {
			t.Errorf("the template file was written: %v", err)
		}
	})
}

func TestRefuseWhatIsNoTemplate(t *testing.T) {
	templates := map[string]string{
		"Templates/Meeting.md": book, "Notes/Meeting.md": book, "Secret.md": "secret", "Templates2/Meeting.md": book, "Notes/Meeting.template.txt": "text",
		"Templates/Meeting.txt": "text", "Templates/Meeting": "no extension",
	}
	refuse := func(t *testing.T, template string) {
		t.Helper()
		vault := newVault(templates)

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: template, Folder: "work"})

		if !errors.Is(err, newnote.ErrNotATemplate) || !strings.Contains(err.Error(), template) {
			t.Errorf("Execute(%q) error = %v", template, err)
		}
		if len(vault.Scans()) != 0 {
			t.Errorf("a note was made for %q: %v", template, vault.Scans())
		}
	}

	t.Run("refuses a note no mask names", func(t *testing.T) {
		refuse(t, "Notes/Meeting.md")
		refuse(t, "Secret.md")
		refuse(t, "Templates2/Meeting.md")
	})
	t.Run("refuses a path with a parent segment", func(t *testing.T) {
		refuse(t, "Templates/../Secret.md")
		refuse(t, "Templates/../Notes/Meeting.md")
		refuse(t, "../Templates/Meeting.md")
	})
	t.Run("refuses a path that is not a note", func(t *testing.T) {
		refuse(t, "Templates/Meeting.txt")
		refuse(t, "Templates/Meeting")
		refuse(t, "Templates")
		refuse(t, "")
	})
	t.Run("reports a missing template as not found", func(t *testing.T) {
		vault := newVault(templates)

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Nothing.md", Folder: "work"})

		if !errors.Is(err, host.ErrNotExist) || errors.Is(err, newnote.ErrNotATemplate) {
			t.Errorf("error = %v", err)
		}
	})
}

func TestMakeNoNoteFromATemplateThatIsNotText(t *testing.T) {
	t.Run("makes no note when the substituted frontmatter is invalid", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Plan: draft.md": "---\nname: {{ .title }}\n---\nbody\n"})

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Plan: draft.md", Folder: "work"})

		if !errors.Is(err, newnote.ErrInvalidNote) {
			t.Errorf("error = %v", err)
		}
		if content, isThere := vault.Content("work/Plan: draft.md"); isThere {
			t.Errorf("a note was made: %q", content)
		}
		if len(vault.Scans()) != 0 {
			t.Errorf("scans = %v", vault.Scans())
		}
	})
	t.Run("makes no note from a template that is not UTF-8 text", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Binary.md": "# {{ .title }}\n\xff\xfe\x00"})

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Binary.md", Folder: "work"})

		if !errors.Is(err, newnote.ErrInvalidNote) {
			t.Errorf("error = %v", err)
		}
		if _, isThere := vault.Content("work/Binary.md"); isThere {
			t.Error("a note was made")
		}
	})
}

func TestReportWhatTheHostRefuses(t *testing.T) {
	t.Run("reports a vault the host does not show", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": book})
		vault.Hide()

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Meeting.md", Folder: "work"})

		if !errors.Is(err, host.ErrNoVault) {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("reports a template the host denies", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Meeting.md": book})
		vault.Refuse("Templates/Meeting.md", host.ErrPermissionDenied)

		_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Meeting.md", Folder: "work"})

		if !errors.Is(err, host.ErrPermissionDenied) {
			t.Errorf("error = %v", err)
		}
	})
}

func TestFillTheFieldsOfTheTemplate(t *testing.T) {
	const asking = "---\nclient: {{ prompt \"Client\" \"Acme\" }}\n---\n# {{ select \"Kind\" \"call\" \"visit\" }} with {{ .vars.Client }} in {{ .folder }}\n"
	execute := func(vault *hosttest.Vault, values map[string]string) (newnote.Result, error) {
		return newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Visit.md", Folder: "work", Values: values})
	}

	t.Run("writes the values given", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Visit.md": asking})

		made, err := execute(vault, map[string]string{"Client": "Globex", "Kind": "visit"})

		if err != nil {
			t.Fatal(err)
		}
		if got := openNote(t, getContent(t, vault, made.Path)).Body(); got != "# visit with Globex in work\n" {
			t.Errorf("body = %q", got)
		}
	})
	t.Run("writes the defaults for no values", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Visit.md": asking})

		made, err := execute(vault, nil)

		if err != nil {
			t.Fatal(err)
		}
		if got := openNote(t, getContent(t, vault, made.Path)).Body(); got != "# call with Acme in work\n" {
			t.Errorf("body = %q", got)
		}
	})
	t.Run("makes no note for a value that is none of the options", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Visit.md": asking})

		_, err := execute(vault, map[string]string{"Kind": "nap"})

		if !errors.Is(err, engine.ErrNotAnOption) || len(vault.Scans()) != 0 {
			t.Errorf("error = %v, scans = %v", err, vault.Scans())
		}
	})
}

func TestKeepAnAnswerInsideItsFrontmatterValue(t *testing.T) {
	const quoting = "---\ntitle: {{ prompt \"Title\" | yaml }}\ntags: [x]\n---\nbody\n"
	for _, c := range []struct{ name, answer, line string }{
		{"a line break and a key", "a\nid: evil", `title: "a id: evil"`},
		{"a document marker", "---", `title: "---"`},
		{"a colon and a space", "a: b", `title: "a: b"`},
		{"a hash", "a #b", `title: "a #b"`},
		{"an opening bracket", "[a, b", `title: "[a, b"`},
		{"an opening brace", "{a: b}", `title: "{a: b}"`},
		{"a quote", `say "hi"`, `title: "say \"hi\""`},
	} {
		t.Run(c.name, func(t *testing.T) {
			vault := newVault(map[string]string{"Templates/Quoted.md": quoting})
			request := newnote.Request{Template: "Templates/Quoted.md", Folder: "work", Values: map[string]string{"Title": c.answer}}

			made, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), request)

			if err != nil {
				t.Fatal(err)
			}
			content := getContent(t, vault, made.Path)
			doc := openNote(t, content)
			if id, _ := doc.Identifier(); id == "evil" || len(id) != 26 {
				t.Errorf("id = %q", id)
			}
			if !strings.Contains(content, c.line+"\n") || !slices.Equal(getKeys(t, content), []string{"title", "tags", "id"}) {
				t.Errorf("the answer broke the frontmatter:\n%s", content)
			}
		})
	}
	t.Run("keeps the id of the note whatever the template or the answer says", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Raw.md": "---\nid: fixed\nname: {{ prompt \"Name\" | yaml }}\n---\nbody\n"})
		request := newnote.Request{Template: "Templates/Raw.md", Folder: "work", Values: map[string]string{"Name": "x\nid: other"}}

		made, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), request)

		if err != nil {
			t.Fatal(err)
		}
		if id, _ := openNote(t, getContent(t, vault, made.Path)).Identifier(); id == "fixed" || id == "other" || len(id) != 26 {
			t.Errorf("id = %q", id)
		}
	})
}

func TestMakeNoNoteFromATemplateThatIsTooLarge(t *testing.T) {
	const limit = 256 << 10
	execute := func(vault *hosttest.Vault) (newnote.Result, error) {
		return newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Big.md", Folder: "work"})
	}

	t.Run("refuses a template over the limit", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Big.md": strings.Repeat("x", limit+1)})

		_, err := execute(vault)

		if !errors.Is(err, newnote.ErrInvalidNote) || !strings.Contains(err.Error(), "256 KiB") {
			t.Errorf("error = %v", err)
		}
		if _, isThere := vault.Content("work/Big.md"); isThere {
			t.Error("a note was made")
		}
	})
	t.Run("takes a template of the size of the limit", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Big.md": strings.Repeat("x", limit)})

		if _, err := execute(vault); err != nil {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("refuses a template that runs for ever", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Big.md": "{{ range 1000000000 }}x{{ end }}"})

		_, err := execute(vault)

		if !errors.Is(err, newnote.ErrInvalidNote) || !errors.Is(err, engine.ErrOutputTooLarge) {
			t.Errorf("error = %v", err)
		}
	})
}

func TestMakeNoNoteFromATemplateThatDoesNotRun(t *testing.T) {
	for _, c := range []struct{ name, text, want string }{
		{"a syntax error", "# {{ .title }}\n{{ if }}\n", "Broken.md:2"},
		{"an unknown function", "line\nline\n{{ nope }}\n", "Broken.md:3"},
		{"an unknown name", "{{ .nothing }}\n", "Broken.md:1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			vault := newVault(map[string]string{"Templates/Broken.md": c.text})

			_, err := newUseCase(vault, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Broken.md", Folder: "work"})

			if !errors.Is(err, newnote.ErrInvalidNote) || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %v, want one with %q", err, c.want)
			}
			if _, isThere := vault.Content("work/Broken.md"); isThere || len(vault.Scans()) != 0 {
				t.Error("a note was made")
			}
		})
	}
}

func TestListTheFieldsOfTheTemplate(t *testing.T) {
	list := func(vault *hosttest.Vault, template string) ([]engine.Field, error) {
		useCase := newnote.NewListFields(vault, newClock(getToday()), func() mask.Set { return mask.Parse("") })
		return useCase.Execute(t.Context(), newnote.FieldsRequest{Template: template, Folder: "work"})
	}

	t.Run("lists what the template asks", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Visit.md": "{{ prompt \"Client\" \"Acme\" }}{{ select \"Kind\" \"call\" \"visit\" }}"})

		fields, err := list(vault, "Templates/Visit.md")

		if err != nil || len(fields) != 2 || fields[0].Label != "Client" || fields[1].Kind != engine.KindSelect {
			t.Errorf("fields = %+v, %v", fields, err)
		}
	})
	t.Run("makes no note", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Visit.md": "{{ prompt \"Client\" }}"})

		if _, err := list(vault, "Templates/Visit.md"); err != nil || len(vault.Scans()) != 0 {
			t.Errorf("error = %v, scans = %v", err, vault.Scans())
		}
	})
	t.Run("refuses a note that is no template", func(t *testing.T) {
		vault := newVault(map[string]string{"Notes/Visit.md": "x"})

		if _, err := list(vault, "Notes/Visit.md"); !errors.Is(err, newnote.ErrNotATemplate) {
			t.Errorf("error = %v", err)
		}
	})
	t.Run("reports a template that does not parse", func(t *testing.T) {
		vault := newVault(map[string]string{"Templates/Broken.md": "{{ if }}"})

		if _, err := list(vault, "Templates/Broken.md"); !errors.Is(err, newnote.ErrInvalidNote) || !strings.Contains(err.Error(), "Broken.md:1") {
			t.Errorf("error = %v", err)
		}
	})
}

func TestWriteTheIDOfTheNewNote(t *testing.T) {
	vault := newVault(map[string]string{"Templates/Ref.md": "---\nid: {{ .id }}\n---\nid is {{ .id }}\n"})

	path := createNote(t, vault, "Templates/Ref.md", "work")

	doc := openNote(t, getContent(t, vault, path))
	if id, _ := doc.Identifier(); id == "" || doc.Body() != "id is "+id+"\n" {
		t.Errorf("id = %q, body = %q", id, doc.Body())
	}
}

type unindexed struct{ *hosttest.Vault }

func (unindexed) Scan(context.Context, []string) error { return errors.New("index is down") }

func TestKeepTheNoteWhenItCannotBeIndexed(t *testing.T) {
	vault := newVault(map[string]string{"Templates/Meeting.md": meeting})

	made, err := newUseCase(unindexed{vault}, newClock(getToday())).Execute(t.Context(), newnote.Request{Template: "Templates/Meeting.md", Folder: "work"})

	if !errors.Is(err, newnote.ErrNotIndexed) || made.Path != "work/Meeting.md" {
		t.Errorf("Execute = %q, %v", made.Path, err)
	}
	if _, isThere := vault.Content("work/Meeting.md"); !isThere {
		t.Error("the note is not made")
	}
}
