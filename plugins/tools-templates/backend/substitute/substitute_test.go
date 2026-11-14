package substitute_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/substitute"
)

// getSaturday is Saturday 2026-11-14 09:05, the instant of the grammar table.
func getSaturday() time.Time { return time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC) }

func getVariables() substitute.Variables {
	return substitute.Variables{Title: "Plan", Now: getSaturday()}
}

type textCase struct {
	name string
	in   string
	want string
}

func runTextCases(t *testing.T, vars substitute.Variables, cases []textCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := substitute.Apply(c.in, vars).Text; got != c.want {
				t.Errorf("Apply(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestReplaceTheTitle(t *testing.T) {
	t.Run("replaces the title variable with the title", func(t *testing.T) {
		got := substitute.Apply("# {{title}}", getVariables()).Text

		if got != "# Plan" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("replaces every title variable", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{{"twice in a row", "{{title}}{{title}}", "PlanPlan"}})
	})
}

func TestReplaceTheDateAndTheTime(t *testing.T) {
	t.Run("writes the date of the instant given as YYYY-MM-DD", func(t *testing.T) {
		if got := substitute.Apply("{{date}}", getVariables()).Text; got != "2026-11-14" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("follows the instant it is given", func(t *testing.T) {
		later := substitute.Variables{Title: "Plan", Now: time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)}

		first := substitute.Apply("{{date}} {{time}}", getVariables()).Text
		second := substitute.Apply("{{date}} {{time}}", later).Text

		if first != "2026-11-14 09:05" || second != "2027-01-02 03:04" {
			t.Errorf("texts = %q and %q", first, second)
		}
	})
	t.Run("writes the time of the instant given as HH:mm", func(t *testing.T) {
		if got := substitute.Apply("{{time}}", getVariables()).Text; got != "09:05" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("formats the instant in its own time zone", func(t *testing.T) {
		zone := time.FixedZone("UTC+5", 5*60*60)
		vars := substitute.Variables{Title: "Plan", Now: time.Date(2026, 11, 14, 23, 30, 0, 0, zone)}

		if got := substitute.Apply("{{date}} {{time}}", vars).Text; got != "2026-11-14 23:30" {
			t.Errorf("text = %q", got)
		}
	})
}

func TestApplyTheFormat(t *testing.T) {
	afternoon := substitute.Variables{Title: "Plan", Now: time.Date(2026, 11, 14, 15, 7, 9, 0, time.UTC)}
	spring := substitute.Variables{Title: "Plan", Now: time.Date(2026, 3, 5, 1, 2, 3, 0, time.UTC)}

	t.Run("applies each token of the format", func(t *testing.T) {
		runTextCases(t, afternoon, []textCase{
			{"YYYY", "{{date:YYYY}}", "2026"},
			{"YY", "{{date:YY}}", "26"},
			{"MMMM", "{{date:MMMM}}", "November"},
			{"MMM", "{{date:MMM}}", "Nov"},
			{"MM", "{{date:MM}}", "11"},
			{"M", "{{date:M}}", "11"},
			{"DD", "{{date:DD}}", "14"},
			{"D", "{{date:D}}", "14"},
			{"dddd", "{{date:dddd}}", "Saturday"},
			{"ddd", "{{date:ddd}}", "Sat"},
			{"HH", "{{time:HH}}", "15"},
			{"H", "{{time:H}}", "15"},
			{"hh", "{{time:hh}}", "03"},
			{"h", "{{time:h}}", "3"},
			{"mm", "{{time:mm}}", "07"},
			{"m", "{{time:m}}", "7"},
			{"ss", "{{time:ss}}", "09"},
			{"s", "{{time:s}}", "9"},
			{"A", "{{time:A}}", "PM"},
			{"a", "{{time:a}}", "pm"},
			{"day, month and year together", "{{date:DD.MM.YYYY}}", "14.11.2026"},
			{"month name with day", "{{date:dddd, D MMMM YYYY}}", "Saturday, 14 November 2026"},
		})
	})
	t.Run("writes one digit tokens without padding and two digit tokens with it", func(t *testing.T) {
		runTextCases(t, spring, []textCase{
			{"M", "{{date:M}}", "3"},
			{"MM", "{{date:MM}}", "03"},
			{"D", "{{date:D}}", "5"},
			{"DD", "{{date:DD}}", "05"},
			{"H", "{{time:H}}", "1"},
			{"HH", "{{time:HH}}", "01"},
			{"MMM", "{{date:MMM}}", "Mar"},
			{"dddd", "{{date:dddd}}", "Thursday"},
			{"ddd", "{{date:ddd}}", "Thu"},
			{"A", "{{time:A}}", "AM"},
			{"a", "{{time:a}}", "am"},
		})
	})
	t.Run("copies bracketed text", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"a word", "{{date:[Year] YYYY}}", "Year 2026"},
			{"tokens inside", "{{date:[YYYY]}}", "YYYY"},
			{"before a time", "{{time:[at] HH:mm}}", "at 09:05"},
			{"a single closing brace inside", "{{date:[}]YYYY}}", "}2026"},
			{"a bracket that is never closed", "{{date:[YYYY}}", "[2026"},
		})
	})
	t.Run("reads everything after the first colon as the format", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"time with seconds", "{{time:HH:mm:ss}}", "09:05:00"},
			{"date with colons", "{{date:YYYY:MM:DD}}", "2026:11:14"},
		})
	})
	t.Run("uses the default format when the format is empty", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"date", "{{date:}}", "2026-11-14"},
			{"time", "{{time:}}", "09:05"},
		})
	})
	t.Run("accepts a time format on the date variable", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"date", "{{date:HH:mm}}", "09:05"},
			{"time", "{{time:YYYY-MM-DD}}", "2026-11-14"},
		})
	})
	t.Run("writes 12 for midnight and noon in the 12-hour clock", func(t *testing.T) {
		for _, c := range []struct {
			name string
			at   time.Time
			want string
		}{
			{"midnight", time.Date(2026, 11, 14, 0, 30, 0, 0, time.UTC), "12:30 AM 12 12"},
			{"noon", time.Date(2026, 11, 14, 12, 30, 0, 0, time.UTC), "12:30 PM 12 12"},
			{"one in the afternoon", time.Date(2026, 11, 14, 13, 30, 0, 0, time.UTC), "01:30 PM 1 01"},
			{"eleven at night", time.Date(2026, 11, 14, 23, 30, 0, 0, time.UTC), "11:30 PM 11 11"},
			{"nine in the morning", time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC), "09:30 AM 9 09"},
		} {
			t.Run(c.name, func(t *testing.T) {
				vars := substitute.Variables{Title: "Plan", Now: c.at}

				if got := substitute.Apply("{{time:hh:mm A}} {{time:h}} {{time:hh}}", vars).Text; got != c.want {
					t.Errorf("text = %q, want %q", got, c.want)
				}
			})
		}
	})
	t.Run("copies letters that are not format tokens", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"letters that are no token", "{{date:xyz}}", "xyz"},
			{"a token followed by a letter", "{{date:Do}}", "14o"},
			{"Q", "{{date:Q}}", "Q"},
			{"w", "{{date:w}}", "w"},
			{"punctuation", "{{date:YYYY/MM/DD - HH.mm}}", "2026/11/14 - 09.05"},
		})
	})
}

func TestKeepWhatIsNotAVariable(t *testing.T) {
	t.Run("keeps an unknown name as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"an unknown name", "{{author}}", "{{author}}"},
			{"a block helper", "{{#each x}}", "{{#each x}}"},
			{"an empty name", "{{}}", "{{}}"},
		})
	})
	t.Run("keeps a name in another case as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"capital", "{{Title}}", "{{Title}}"},
			{"all capitals", "{{DATE}}", "{{DATE}}"},
		})
	})
	t.Run("keeps a name with spaces as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"around the name", "{{ title }}", "{{ title }}"},
			{"before the name", "{{ date}}", "{{ date}}"},
		})
	})
	t.Run("keeps a title variable with a format as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"title", "{{title:x}}", "{{title:x}}"},
			{"cursor", "{{cursor:x}}", "{{cursor:x}}"},
		})
	})
}

func TestReadBracesInTheGrammar(t *testing.T) {
	t.Run("reads the variable inside three braces", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{{"three braces", "{{{title}}}", "{Plan}"}})
	})
	t.Run("reads the variable after a surplus closing brace", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{{"closing brace after the format", "{{date:YYYY}}}", "2026}"}})
	})
	t.Run("keeps a variable with one closing brace as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{{"one closing brace", "{{title}", "{{title}"}})
	})
	t.Run("keeps an unterminated variable as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"no closing braces", "{{title", "{{title"},
			{"no opening braces", "title}}", "title}}"},
			{"a lone opening pair", "{{", "{{"},
			{"a lone closing pair", "}}", "}}"},
			{"a pair after the text", "cost {{", "cost {{"},
		})
	})
	t.Run("keeps a variable that spans lines as typed", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{
			{"a break inside the name", "{{ti\ntle}}", "{{ti\ntle}}"},
			{"a break before the name", "{{\ntitle}}", "{{\ntitle}}"},
			{"a break inside the format", "{{date:YYYY\n}}", "{{date:YYYY\n}}"},
			{"an opening pair and a closing pair on later lines", "start {{ here\nmid\nend }} {{title}}", "start {{ here\nmid\nend }} Plan"},
		})
	})
	t.Run("keeps a single closing brace inside a format", func(t *testing.T) {
		runTextCases(t, getVariables(), []textCase{{"brace in brackets", "{{date:[}]YYYY}}", "}2026"}})
	})
}

func TestApplyInsideCodeAndFrontmatter(t *testing.T) {
	t.Run("replaces a known variable inside a fence and keeps an unknown one", func(t *testing.T) {
		in := "```\n{{title}} {{author}}\n```\n`{{date}}` and `{{name}}`"
		want := "```\nPlan {{author}}\n```\n`2026-11-14` and `{{name}}`"

		if got := substitute.Apply(in, getVariables()).Text; got != want {
			t.Errorf("text = %q, want %q", got, want)
		}
	})
	t.Run("replaces variables in the frontmatter as in any text", func(t *testing.T) {
		in := "---\ncreated: \"{{date}}\"\nupdated: {{date}}\n---\n# {{title}}\n"
		want := "---\ncreated: \"2026-11-14\"\nupdated: 2026-11-14\n---\n# Plan\n"

		if got := substitute.Apply(in, getVariables()).Text; got != want {
			t.Errorf("text = %q, want %q", got, want)
		}
	})
}

func TestReportTheCursor(t *testing.T) {
	t.Run("removes the cursor marker and reports its offset", func(t *testing.T) {
		got := substitute.Apply("ab{{cursor}}cd", getVariables())

		if got.Text != "abcd" || got.Cursor != 2 {
			t.Errorf("result = %+v", got)
		}
	})
	t.Run("reports the offset of the first marker and removes the others", func(t *testing.T) {
		got := substitute.Apply("x{{cursor}}y{{cursor}}z", getVariables())

		if got.Text != "xyz" || got.Cursor != 1 {
			t.Errorf("result = %+v", got)
		}
	})
	t.Run("reports -1 when there is no marker", func(t *testing.T) {
		for _, in := range []string{"", "plain text", "{{title}}", "{{cursor:x}}", "{{Cursor}}"} {
			if got := substitute.Apply(in, getVariables()); got.Cursor != -1 {
				t.Errorf("Apply(%q).Cursor = %d, want -1", in, got.Cursor)
			}
		}
	})
	t.Run("counts the offset in UTF-16 units after earlier replacements and astral characters", func(t *testing.T) {
		for _, c := range []struct {
			name string
			in   string
			text string
			want int
		}{
			{"after a longer replacement", "{{date:YYYY}} {{cursor}}", "2026 ", 5},
			{"after a replacement of the title", "# {{title}}\n{{cursor}}", "# Plan\n", 7},
			{"after an astral character", "😀{{title}}{{cursor}}", "😀Plan", 6},
			{"after two BMP characters", "日本{{cursor}}", "日本", 2},
			{"after an earlier marker", "{{cursor}}{{title}}{{cursor}}", "Plan", 0},
		} {
			t.Run(c.name, func(t *testing.T) {
				got := substitute.Apply(c.in, getVariables())

				if got.Text != c.text || got.Cursor != c.want {
					t.Errorf("result = %+v, want text %q and cursor %d", got, c.text, c.want)
				}
			})
		}
	})
}

func TestApplyIsPureAndTotal(t *testing.T) {
	t.Run("returns the text unchanged when it has no braces", func(t *testing.T) {
		for _, in := range []string{"", "plain text", "# Heading\n\n- item\n", "a } b { c", "{ title }", "日本語 😀"} {
			if got := substitute.Apply(in, getVariables()).Text; got != in {
				t.Errorf("Apply(%q) = %q", in, got)
			}
		}
	})
	t.Run("returns the same result for the same input", func(t *testing.T) {
		in := "{{title}} {{date:dddd}} {{cursor}} {{time}}"

		if first, second := substitute.Apply(in, getVariables()), substitute.Apply(in, getVariables()); first != second {
			t.Errorf("results differ: %+v and %+v", first, second)
		}
	})
}

func FuzzApplyNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"", "{{", "}}", "{{:", "{{date:[", "{{date:[}}", "{{cursor}}{{cursor}}", "{{{{{{", "}}}}}}",
		"{{title}}{{", "{{date:YYYYYYYYYY}}", "\xed\xa0\x80{{title}}", "\xed\xb0\x80{{cursor}}\xff", "😀{{cursor}}😀",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got := substitute.Apply(in, getVariables())

		if !strings.Contains(in, "{{cursor}}") && got.Cursor != -1 {
			t.Errorf("Apply(%q).Cursor = %d, want -1", in, got.Cursor)
		}
		if got.Cursor > len(utf16.Encode([]rune(got.Text))) {
			t.Errorf("Apply(%q).Cursor = %d is beyond the text", in, got.Cursor)
		}
	})
}
