package engine_test

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
)

func getSaturday() time.Time { return time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC) }

func getData() engine.Data {
	return engine.Data{Title: "Plan", Folder: "work", ID: "01J0NEW", Now: getSaturday(), Values: nil}
}

func render(t *testing.T, text string) string {
	t.Helper()
	return renderWith(t, text, getData())
}

func renderWith(t *testing.T, text string, data engine.Data) string {
	t.Helper()
	got, err := engine.New(nil).Render(t.Context(), "T.md", text, data)
	if err != nil {
		t.Fatalf("Render(%q): %v", text, err)
	}
	return got.Text
}

type textCase struct{ name, in, want string }

func runTextCases(t *testing.T, data engine.Data, cases []textCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := renderWith(t, c.in, data); got != c.want {
				t.Errorf("Render(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestExposeTheData(t *testing.T) {
	data := getData()
	data.Values = map[string]string{"Client": "Acme"}
	runTextCases(t, data, []textCase{
		{"the title", "# {{ .title }}", "# Plan"},
		{"the folder", "{{ .folder }}", "work"},
		{"the id", "{{ .id }}", "01J0NEW"},
		{"the instant", "{{ .now.Year }}", "2026"},
		{"a value", `{{ $_ := prompt "Client" }}{{ .vars.Client }}`, "Acme"},
		{"plain text", "# Heading\n\n- item\n日本語 😀", "# Heading\n\n- item\n日本語 😀"},
		{"single braces", "a } b { c { title }", "a } b { c { title }"},
	})
}

func TestFormatTheDateAndTheTime(t *testing.T) {
	afternoon := getData()
	afternoon.Now = time.Date(2026, 11, 14, 15, 7, 9, 0, time.UTC)
	spring := getData()
	spring.Now = time.Date(2026, 3, 5, 1, 2, 3, 0, time.UTC)

	t.Run("applies each token of the format", func(t *testing.T) {
		runTextCases(t, afternoon, []textCase{
			{"YYYY", `{{ date "YYYY" }}`, "2026"},
			{"YY", `{{ date "YY" }}`, "26"},
			{"MMMM", `{{ date "MMMM" }}`, "November"},
			{"MMM", `{{ date "MMM" }}`, "Nov"},
			{"MM", `{{ date "MM" }}`, "11"},
			{"M", `{{ date "M" }}`, "11"},
			{"DD", `{{ date "DD" }}`, "14"},
			{"D", `{{ date "D" }}`, "14"},
			{"dddd", `{{ date "dddd" }}`, "Saturday"},
			{"ddd", `{{ date "ddd" }}`, "Sat"},
			{"HH", `{{ time "HH" }}`, "15"},
			{"H", `{{ time "H" }}`, "15"},
			{"hh", `{{ time "hh" }}`, "03"},
			{"h", `{{ time "h" }}`, "3"},
			{"mm", `{{ time "mm" }}`, "07"},
			{"m", `{{ time "m" }}`, "7"},
			{"ss", `{{ time "ss" }}`, "09"},
			{"s", `{{ time "s" }}`, "9"},
			{"A", `{{ time "A" }}`, "PM"},
			{"a", `{{ time "a" }}`, "pm"},
			{"day, month and year together", `{{ date "DD.MM.YYYY" }}`, "14.11.2026"},
			{"month name with day", `{{ date "dddd, D MMMM YYYY" }}`, "Saturday, 14 November 2026"},
		})
	})
	t.Run("writes one digit tokens without padding and two digit tokens with it", func(t *testing.T) {
		runTextCases(t, spring, []textCase{
			{"M", `{{ date "M" }}`, "3"},
			{"MM", `{{ date "MM" }}`, "03"},
			{"D", `{{ date "D" }}`, "5"},
			{"DD", `{{ date "DD" }}`, "05"},
			{"HH", `{{ time "HH" }}`, "01"},
			{"dddd", `{{ date "dddd" }}`, "Thursday"},
			{"A", `{{ time "A" }}`, "AM"},
		})
	})
	t.Run("copies bracketed text and letters that are not tokens", func(t *testing.T) {
		runTextCases(t, getData(), []textCase{
			{"a word", `{{ date "[Year] YYYY" }}`, "Year 2026"},
			{"tokens inside", `{{ date "[YYYY]" }}`, "YYYY"},
			{"a bracket that is never closed", `{{ date "[YYYY" }}`, "[2026"},
			{"a token followed by a letter", `{{ date "Do" }}`, "14o"},
			{"punctuation", `{{ date "YYYY/MM/DD - HH.mm" }}`, "2026/11/14 - 09.05"},
		})
	})
	t.Run("uses the default format when the format is empty", func(t *testing.T) {
		runTextCases(t, getData(), []textCase{
			{"date", `{{ date "" }}`, "2026-11-14"},
			{"time", `{{ time "" }}`, "09:05"},
		})
	})
	t.Run("writes 12 for midnight and noon in the 12-hour clock", func(t *testing.T) {
		midnight, noon := getData(), getData()
		midnight.Now = time.Date(2026, 11, 14, 0, 30, 0, 0, time.UTC)
		noon.Now = time.Date(2026, 11, 14, 12, 30, 0, 0, time.UTC)

		if got := renderWith(t, `{{ time "hh:mm A" }}`, midnight); got != "12:30 AM" {
			t.Errorf("midnight = %q", got)
		}
		if got := renderWith(t, `{{ time "hh:mm A" }}`, noon); got != "12:30 PM" {
			t.Errorf("noon = %q", got)
		}
	})
	t.Run("formats the instant in its own time zone", func(t *testing.T) {
		data := getData()
		data.Now = time.Date(2026, 11, 14, 23, 30, 0, 0, time.FixedZone("UTC+5", 5*60*60))

		if got := renderWith(t, `{{ date "YYYY-MM-DD" }} {{ time "HH:mm" }}`, data); got != "2026-11-14 23:30" {
			t.Errorf("text = %q", got)
		}
	})
}

func TestMoveTheInstant(t *testing.T) {
	end := getData()
	end.Now = time.Date(2026, 1, 31, 9, 5, 0, 0, time.UTC)
	runTextCases(t, getData(), []textCase{
		{"a day on", `{{ addDays 1 | date "YYYY-MM-DD" }}`, "2026-11-15"},
		{"a week back", `{{ addDays -7 | date "YYYY-MM-DD" }}`, "2026-11-07"},
		{"a month on", `{{ addMonths 1 | date "YYYY-MM" }}`, "2026-12"},
		{"a month across the year", `{{ addMonths 2 | date "YYYY-MM" }}`, "2027-01"},
		{"the weekday of now", `{{ weekday }}`, "Saturday"},
		{"the weekday of another day", `{{ addDays 2 | weekday }}`, "Monday"},
		{"a time of another day", `{{ addDays 1 | time "HH:mm" }}`, "09:05"},
		{"an explicit instant", `{{ date "YYYY-MM-DD" (addDays 3) }}`, "2026-11-17"},
		{"the instant itself", `{{ date "YYYY" .now }}`, "2026"},
		{"chained moves", `{{ addDays 1 | addMonths 1 | date "YYYY-MM-DD" }}`, "2026-12-15"},
	})
	if got := renderWith(t, `{{ addMonths 1 | date "YYYY-MM-DD" }}`, end); got != "2026-03-03" {
		t.Errorf("a month from the 31st of January = %q", got)
	}
}

func TestTransformText(t *testing.T) {
	data := getData()
	data.Title = "Book review: Dune"
	data.Values = map[string]string{"Empty": ""}
	runTextCases(t, data, []textCase{
		{"upper", `{{ .title | upper }}`, "BOOK REVIEW: DUNE"},
		{"lower", `{{ .title | lower }}`, "book review: dune"},
		{"slug", `{{ .title | slug }}`, "book-review-dune"},
		{"slug of accents and capitals", `{{ slug "  Ärger  mit   Öl!! " }}`, "ärger-mit-öl"},
		{"title", `{{ title "book review of dune" }}`, "Book Review Of Dune"},
		{"default of an empty value", `{{ $_ := prompt "Empty" }}{{ .vars.Empty | default "none" }}`, "none"},
		{"default of a value", `{{ default "none" "some" }}`, "some"},
		{"trunc", `{{ trunc 4 "abcdefg" }}`, "abcd"},
		{"trunc of a short text", `{{ trunc 40 "abc" }}`, "abc"},
		{"trunc counts characters", `{{ trunc 2 "日本語" }}`, "日本"},
		{"join", `{{ join ", " (list "a" "b" "c") }}`, "a, b, c"},
		{"split then join", `{{ split "," "a,b,c" | join "-" }}`, "a-b-c"},
		{"split in a range", `{{ range split " " "x y" }}[{{ . }}]{{ end }}`, "[x][y]"},
		{"replace", `{{ replace "a" "b" "banana" }}`, "bbnbnb"},
		{"trim", `{{ trim "  x  " }}`, "x"},
	})
}

func TestUseTheControlOfTextTemplate(t *testing.T) {
	runTextCases(t, getData(), []textCase{
		{"if and else", `{{ if eq .folder "work" }}W{{ else }}H{{ end }}`, "W"},
		{"a comparison of weekdays", `{{ if eq weekday "Saturday" }}weekend{{ end }}`, "weekend"},
		{"a variable", `{{ $d := addDays 1 }}{{ date "DD" $d }}`, "15"},
		{"range over a list", `{{ range list "a" "b" }}-{{ . }}{{ end }}`, "-a-b"},
		{"with", `{{ with .folder }}in {{ . }}{{ end }}`, "in work"},
		{"a dict", `{{ $m := dict "k" "v" }}{{ $m.k }}`, "v"},
		{"and, or and not", `{{ if and (not false) (or false true) }}yes{{ end }}`, "yes"},
		{"lt", `{{ if lt 1 2 }}yes{{ end }}`, "yes"},
	})
}

func TestMakeAnID(t *testing.T) {
	first := render(t, "{{ uuid }}")
	second := render(t, "{{ uuid }}")

	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(first) || first == second {
		t.Errorf("uuids = %q and %q", first, second)
	}
}

func TestKnowNoCursorFunction(t *testing.T) {
	_, err := engine.New(nil).Render(t.Context(), "T.md", "a{{ cursor }}b", getData())

	if !errors.Is(err, engine.ErrTemplate) || !strings.Contains(err.Error(), "cursor") {
		t.Errorf("error = %v", err)
	}
}

func TestQuoteAValueForYAML(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"plain text", "Acme", `"Acme"`},
		{"a colon", "a: b", `"a: b"`},
		{"a hash", "a # b", `"a # b"`},
		{"a bracket", "[a]", `"[a]"`},
		{"a brace", "{a: 1}", `"{a: 1}"`},
		{"a document marker", "---", `"---"`},
		{"a quote and a backslash", `say "hi" \ now`, `"say \"hi\" \\ now"`},
		{"a line break", "a\nid: x", `"a\nid: x"`},
		{"markup characters", "<a & b>", `"<a & b>"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			data := getData()
			data.Title = c.in

			if got := renderWith(t, `{{ yaml .title }}`, data); got != c.want {
				t.Errorf("yaml(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestKeepAnAnswerOnOneLine(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"a line feed", "a\nid: x", "a id: x"},
		{"a carriage return and a line feed", "a\r\nb", "a b"},
		{"a lone carriage return", "a\rb", "a b"},
		{"a run of breaks", "a\n\n\nb", "a b"},
		{"the unicode line separators", "a b c\u0085d", "a b c d"},
		{"a control character", "a\x00b\x07c", "abc"},
		{"the old cursor marker", "a\x00cursor\x00b", "acursorb"},
		{"a tab", "a\tb", "a\tb"},
		{"text with no break", "---: [x] #y {z}", "---: [x] #y {z}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			data := getData()
			data.Values = map[string]string{"V": c.in}

			if got := renderWith(t, `{{ prompt "V" }}`, data); got != c.want {
				t.Errorf("answer %q gives %q, want %q", c.in, got, c.want)
			}
		})
	}
	t.Run("keeps an answer that is only breaks empty", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"V": "\n\x00"}

		if got := renderWith(t, `{{ prompt "V" "Acme" }}`, data); got != "" {
			t.Errorf("text = %q", got)
		}
	})
}

func TestFailClearly(t *testing.T) {
	for _, c := range []struct {
		name, in string
		want     []string
	}{
		{"an unclosed action", "line one\n{{ .title", []string{"T.md:2"}},
		{"an unknown function", "a\nb\n{{ nope }}", []string{"T.md:3", "nope"}},
		{"an unknown key", "x\n{{ .tittle }}", []string{"T.md:2", "tittle"}},
		{"an unknown value", "{{ .vars.Missing }}", []string{"Missing"}},
		{"a function that fails", `{{ dict "a" }}`, []string{"T.md:1", "dict"}},
		{"an unclosed block", "{{ if true }}x", []string{"T.md"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := engine.New(nil).Render(t.Context(), "T.md", c.in, getData())

			if !errors.Is(err, engine.ErrTemplate) {
				t.Fatalf("error = %v", err)
			}
			for _, part := range c.want {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q lacks %q", err, part)
				}
			}
		})
	}
	t.Run("fails the same way when the fields are listed", func(t *testing.T) {
		_, err := engine.New(nil).ListFields(t.Context(), "T.md", "{{ nope }}", getData())

		if !errors.Is(err, engine.ErrTemplate) {
			t.Errorf("error = %v", err)
		}
	})
}

func TestReachNothingButTheFuncsAndTheData(t *testing.T) {
	for _, in := range []string{`{{ template "x" }}`, `{{ call .title }}`, `{{ os.Getenv "HOME" }}`, `{{ .nothing }}`} {
		if _, err := engine.New(nil).Render(t.Context(), "T.md", in, getData()); !errors.Is(err, engine.ErrTemplate) {
			t.Errorf("Render(%q) error = %v", in, err)
		}
	}
}

func TestTakeExtraFunctionsThroughOneSeam(t *testing.T) {
	extra := template.FuncMap{"shout": func(s string) string { return strings.ToUpper(s) + "!" }}

	got, err := engine.New(extra).Render(t.Context(), "T.md", `{{ shout .title }}`, getData())

	if err != nil || got.Text != "PLAN!" {
		t.Errorf("result = %+v, %v", got, err)
	}
}

func TestListTheFields(t *testing.T) {
	text := `{{ $c := prompt "Client" "Acme" }}{{ $k := select "Kind" "call" "visit" }}{{ prompt "Notes" }}{{ prompt "Client" "again" }}`

	fields, err := engine.New(nil).ListFields(t.Context(), "T.md", text, getData())

	want := []engine.Field{
		{Kind: "text", Label: "Client", Options: []string{}, Default: "Acme"},
		{Kind: "select", Label: "Kind", Options: []string{"call", "visit"}, Default: "call"},
		{Kind: "text", Label: "Notes", Options: []string{}, Default: ""},
	}
	if err != nil || !slices.EqualFunc(fields, want, func(a, b engine.Field) bool {
		return a.Kind == b.Kind && a.Label == b.Label && a.Default == b.Default && slices.Equal(a.Options, b.Options)
	}) {
		t.Errorf("fields = %+v, %v", fields, err)
	}
	t.Run("lists no field for a template that asks nothing", func(t *testing.T) {
		fields, err := engine.New(nil).ListFields(t.Context(), "T.md", "# {{ .title }}", getData())

		if err != nil || len(fields) != 0 {
			t.Errorf("fields = %+v, %v", fields, err)
		}
	})
	t.Run("lists the fields a branch reaches", func(t *testing.T) {
		fields, err := engine.New(nil).ListFields(t.Context(), "T.md", `{{ if eq (select "A" "x" "y") "y" }}{{ prompt "B" }}{{ else }}{{ prompt "C" }}{{ end }}`, getData())

		if err != nil || len(fields) != 2 || fields[1].Label != "C" {
			t.Errorf("fields = %+v, %v", fields, err)
		}
	})
}

func TestFillTheFieldsWithTheValues(t *testing.T) {
	text := `{{ prompt "Client" "Acme" }}/{{ select "Kind" "call" "visit" }}/{{ .vars.Kind }}`

	t.Run("writes the values given", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Client": "Globex", "Kind": "visit"}

		if got := renderWith(t, text, data); got != "Globex/visit/visit" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("writes the default of a field with no value", func(t *testing.T) {
		if got := render(t, text); got != "Acme/call/call" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("ignores a value no field asks for", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Other": "x"}

		if got := renderWith(t, text, data); got != "Acme/call/call" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("refuses a value that is none of the options", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Kind": "nap"}

		_, err := engine.New(nil).Render(t.Context(), "T.md", text, data)

		if !errors.Is(err, engine.ErrNotAnOption) || !strings.Contains(err.Error(), "Kind") {
			t.Errorf("error = %v", err)
		}
	})
}

func FuzzRenderNeverPanics(f *testing.F) {
	for _, seed := range []string{"", "{{", "}}", "{{ . }}", `{{ date "[" }}`, "{{{{{{", "\xff😀"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		if _, err := engine.New(nil).Render(t.Context(), "T.md", in, getData()); err != nil && !errors.Is(err, engine.ErrTemplate) {
			t.Errorf("Render(%q) error = %v", in, err)
		}
	})
}

func TestAskOnceForTheSameField(t *testing.T) {
	text := `{{ prompt "Client" }}|{{ prompt "Client" }}|{{ select "Kind" "a" "b" }}|{{ select "Kind" "a" "b" }}`

	fields, err := engine.New(nil).ListFields(t.Context(), "T.md", text, getData())

	if err != nil || len(fields) != 2 {
		t.Fatalf("fields = %+v, %v", fields, err)
	}
	t.Run("writes the one answer everywhere", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Client": "Globex", "Kind": "b"}

		got, err := engine.New(nil).Render(t.Context(), "T.md", text, data)

		if err != nil || got.Text != "Globex|Globex|b|b" {
			t.Errorf("text = %q, %v", got.Text, err)
		}
	})
}

func TestUseAFieldAsAVariable(t *testing.T) {
	text := `{{ $name := prompt "Имя" }}{{ if eq $name "" }}anon{{ else }}{{ $name | upper }}{{ end }}/{{ $name | default "none" }}/{{ $name }}`

	t.Run("runs the collecting pass over the placeholder", func(t *testing.T) {
		fields, err := engine.New(nil).ListFields(t.Context(), "T.md", text, getData())

		if err != nil || len(fields) != 1 || fields[0].Label != "Имя" {
			t.Errorf("fields = %+v, %v", fields, err)
		}
	})
	t.Run("renders the value given", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Имя": "Ann"}

		if got := renderWith(t, text, data); got != "ANN/Ann/Ann" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("ranges over the placeholder", func(t *testing.T) {
		got := render(t, `{{ range split "," (prompt "Tags" "a,b") }}#{{ . }} {{ end }}`)

		if got != "#a #b " {
			t.Errorf("text = %q", got)
		}
	})
}

func TestTakeTheDefaultOnlyForAMissingAnswer(t *testing.T) {
	t.Run("keeps a text the person emptied", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Client": ""}

		if got := renderWith(t, `{{ prompt "Client" "Acme" }}`, data); got != "" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("takes the default of an answer not given", func(t *testing.T) {
		if got := render(t, `{{ prompt "Client" "Acme" }}`); got != "Acme" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("keeps the default on one line", func(t *testing.T) {
		if got := render(t, "{{ prompt \"Client\" \"a\\nb\" }}"); got != "a b" {
			t.Errorf("text = %q", got)
		}
	})
}

func TestRenderTheExamplesOfTheDocs(t *testing.T) {
	t.Run("meeting note with prompts", func(t *testing.T) {
		data := getData()
		data.Values = map[string]string{"Client": "Globex", "Topic": "Budget"}
		text := "{{ $client := prompt \"Client\" \"Acme\" }}# Meeting with {{ $client }}\n\nDate: {{ date \"YYYY-MM-DD\" }}\nTopic: {{ prompt \"Topic\" }}\nNotes on {{ $client }}: \n"

		got, err := engine.New(nil).Render(t.Context(), "T.md", text, data)

		want := "# Meeting with Globex\n\nDate: 2026-11-14\nTopic: Budget\nNotes on Globex: \n"
		if err != nil || got.Text != want {
			t.Errorf("result = %+v, %v", got, err)
		}
	})
	t.Run("daily note with date functions", func(t *testing.T) {
		text := "# {{ date \"dddd, D MMMM YYYY\" }}\n\nYesterday: [[{{ addDays -1 | date \"YYYY-MM-DD\" }}]]\nTomorrow: [[{{ addDays 1 | date \"YYYY-MM-DD\" }}]]\n"

		want := "# Saturday, 14 November 2026\n\nYesterday: [[2026-11-13]]\nTomorrow: [[2026-11-15]]\n"
		if got := render(t, text); got != want {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("note by select", func(t *testing.T) {
		text := "# {{ .title }}\n{{ $kind := select \"Kind\" \"call\" \"visit\" }}Kind: {{ $kind }}\n{{ $where := prompt \"Address or phone\" }}{{ if eq $kind \"visit\" }}Address: {{ $where }}{{ else }}Phone: {{ $where }}{{ end }}\n"
		visit := getData()
		visit.Values = map[string]string{"Kind": "visit", "Address or phone": "Main St"}

		if got := render(t, text); got != "# Plan\nKind: call\nPhone: \n" {
			t.Errorf("call = %q", got)
		}
		if got := renderWith(t, text, visit); got != "# Plan\nKind: visit\nAddress: Main St\n" {
			t.Errorf("visit = %q", got)
		}
	})
}
