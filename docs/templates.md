# Templates

A template is a note whose path matches a mask of the templates setting. The command "New note from template" makes a new note from one: it runs the template, asks what the template asks, and opens the result. The Templates plugin, `md.numen.tools.templates`, does this.

## Making a template

Put a note in a folder named `Templates` at the root of the vault, or give it a name that ends in `.template.md` anywhere in the vault: `Meeting.template.md`. Either is a template. Which notes are templates is the setting `templates` of the plugin (see [Settings](settings.md)): masks separated by commas, `Templates/**, **/*.template.md` by default. A note is a template when its path matches any of them.

A mask is a path with wildcards. `*` stands for part of a name, `?` for one character, `[ab]` for one of the characters listed, and `**` for any number of folders, so `Templates/**` is every note under `Templates` at any depth and `Models/*` is the notes directly in `Models`. A mask is read from the root of the vault and tells capitals from small letters. A mask with a `..` segment is ignored, and a setting with no mask left is read as the default.

The picker shows each template under a group. Where a mask starts with a folder, as `Templates/**` does, the group is the folder of the note below that folder: `Templates/Work/Meeting.md` is in the group `Work`, and `Templates/Meeting.md` in none. Otherwise the group is the folder of the note, so `Notes/Weekly.template.md` is in the group `Notes`. When no note is a template, the command is not listed anywhere.

## Making a note from a template

Right-click a folder in the file tree, open "New" and pick "New note from template", then pick a template. The note is made inside that folder. From the palette, the note is made in the folder of the note in front. A note that has no path makes it at the root of the vault.

The new note is named after the template with `.template` left out, so `Meeting.template.md` makes `Meeting.md`, and a number is added where the name is taken: `Meeting.md`, `Meeting 2.md`, `Meeting 3.md`. The note opens in the current pane. Rename it as any note. The template is never changed.

The new note holds the frontmatter of the template with its keys in their order, and its own `id`: an `id` the template has is replaced. A template without frontmatter gives a note with the `id` alone. When the template asks questions (see [Asking](#asking)), a small form shows them first, and Cancel makes no note.

## The template language

The text of a template is run by the Go template language, `text/template`, with actions between double braces: `{{ .title }}`. It runs in the frontmatter as well as in the text, and inside code too, so a literal `{{` is written `{{ "{{" }}`. A template that does not parse or does not run makes no note and says why, with the line: `template: Meeting.md:3: function "nope" not defined`. A name that does not exist, as `{{ .tittle }}`, is an error as well, and so is a frontmatter that is not valid after the run. The template reaches nothing but what is listed here.

### What a template knows

| Name | Is |
| --- | --- |
| `.title` | the name of the new note |
| `.folder` | the folder the note is made in, empty for the root of the vault |
| `.id` | the `id` of the new note |
| `.now` | the date and time of the call |
| `.vars` | the answers to the questions, by label: `.vars.Client` |

The date and the time are those of the machine at the call, in its time zone, and the names of months and days are in English.

### Dates

| Function | Gives |
| --- | --- |
| `date "YYYY-MM-DD"` | the date written as the format says |
| `time "HH:mm"` | the time written as the format says |
| `addDays n` | the instant `n` days on, or back when `n` is negative |
| `addMonths n` | the instant `n` months on |
| `weekday` | the name of the day of the week, `Saturday` |

Each takes an instant as its last argument and takes `.now` where none is given, so they pipe: `{{ addDays 1 | date "YYYY-MM-DD" }}` is tomorrow, and `{{ addDays 2 | weekday }}` is the day after tomorrow's name. An empty format is the default one, `YYYY-MM-DD` for `date` and `HH:mm` for `time`. Moving by months keeps the day of the month, so a month on from the 31st of January is the 3rd of March.

### Formats

The tokens are those of Moment.js, a subset of them:

| Token | Becomes | Token | Becomes |
| --- | --- | --- | --- |
| `YYYY` | 2026 | `HH`, `H` | 09, 9 (24-hour clock) |
| `YY` | 26 | `hh`, `h` | 09, 9 (12-hour clock, 12 at midnight and noon) |
| `MMMM`, `MMM` | November, Nov | `mm`, `m` | 05, 5 |
| `MM`, `M` | 11, 11 | `ss`, `s` | 00, 0 |
| `DD`, `D` | 14, 14 | `A`, `a` | AM or PM, am or pm |
| `dddd`, `ddd` | Saturday, Sat | `[text]` | `text`, with no token read inside |

Every other character is copied, and so is a letter that is not a token of this list: `{{ date "Do" }}` gives `14o`. A `[` that is never closed is copied.

### Text

| Function | Gives |
| --- | --- |
| `upper s`, `lower s` | the text in capitals, or in small letters |
| `title s` | the text with each word starting with a capital |
| `slug s` | small letters and digits joined by `-`: `Book review!` is `book-review` |
| `default d v` | `v`, or `d` where `v` is empty |
| `trunc n s` | the first `n` characters (the last ones where `n` is negative) |
| `join sep items`, `split sep s` | a list written as text, and text cut into a list |
| `replace old new s` | the text with every `old` replaced |
| `trim s` | the text without the spaces at its ends |
| `uuid` | a new random identifier |
| `cursor` | nothing; marks the place for the cursor and is removed |

These pipe as well: `{{ .title | slug }}`, `{{ .vars.Client | default "none" | upper }}`. Control comes with the language: `if`, `else`, `range`, `with`, variables (`{{ $d := addDays 1 }}`), `eq`, `ne`, `lt`, `and`, `or`, `not`, `list` and `dict`.

### Asking

A template can ask before it is made into a note:

- `{{ prompt "Client" }}` asks for a line of text called Client, and `{{ prompt "Client" "Acme" }}` starts it with Acme.
- `{{ select "Kind" "call" "visit" }}` asks to pick one of the options, the first at the start.

Each becomes the answer, and the answer is kept in `.vars` under the label, so `{{ .vars.Kind }}` repeats it where the label has no spaces. The questions are collected before the note is made, once for each label, in the order the template reaches them; to find them the template is run once with the defaults, so a question under an `if` is asked when the defaults reach it. A field left as it is takes its default.

A template with no question makes its note at once.

### An example

```markdown
---
type: meeting
created: {{ date "YYYY-MM-DD" }}
---
# {{ .title }} with {{ prompt "Client" "Acme" }}

{{ if eq (select "Kind" "call" "visit") "visit" }}Address:{{ end }}
Next week: {{ addDays 7 | date "dddd, D MMMM" }}
{{ cursor }}
```
