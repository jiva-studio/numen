# Templates

A template is a note in the templates folder. The command "New note from template" makes a new note from one: it copies the template, fills in the variables and opens the result. The Templates plugin, `md.numen.tools.templates`, does this.

## Making a template

Make a folder named `Templates` at the root of the vault and put notes in it. Each note is a template, and each subfolder is a group that the picker shows its templates under. The folder is the setting `folder` of the plugin (see [Settings](settings.md)). When the folder holds no note, the command is not listed anywhere, so a vault that has no `Templates` folder shows no "New note from template".

## Making a note from a template

Right-click a folder in the file tree, open "New" and pick "New note from template", then pick a template. The note is made inside that folder. From the palette, the note is made in the folder of the note in front. A note that has no path makes it at the root of the vault.

The new note is named after the template, and a number is added where the name is taken: `Meeting.md`, `Meeting 2.md`, `Meeting 3.md`. The note opens in the current pane. Rename it as any note. The template is never changed.

The new note holds the frontmatter of the template with its keys in their order, and its own `id`. A template without frontmatter gives a note with the `id` alone.

## Variables

A variable is a name in double braces, replaced when the note is made. It is replaced in the frontmatter as well as in the text, and inside code too.

| Variable | Becomes |
| --- | --- |
| `{{title}}` | the name of the new note |
| `{{date}}` | today's date, `2026-11-14` |
| `{{time}}` | the time now, `09:05` |
| `{{date:format}}`, `{{time:format}}` | the date or the time written as the format says |
| `{{cursor}}` | nothing; it is removed |

Anything else between double braces stays as it is typed: `{{author}}`, `{{Title}}` and `{{ title }}` are not variables. `{{title:x}}` and `{{cursor:x}}` are not variables either. A variable that spans two lines is not a variable. There is no way to write a literal `{{title}}`.

The date and the time are those of the machine at the call, in its time zone, and the names of months and days are in English.

## Formats

A format is everything after the first colon, so `{{time:HH:mm:ss}}` has the format `HH:mm:ss`. An empty format is the default one, `YYYY-MM-DD` for a date and `HH:mm` for a time. The tokens are those of Moment.js, a subset of them:

| Token | Becomes | Token | Becomes |
| --- | --- | --- | --- |
| `YYYY` | 2026 | `HH`, `H` | 09, 9 (24-hour clock) |
| `YY` | 26 | `hh`, `h` | 09, 9 (12-hour clock, 12 at midnight and noon) |
| `MMMM`, `MMM` | November, Nov | `mm`, `m` | 05, 5 |
| `MM`, `M` | 11, 11 | `ss`, `s` | 00, 0 |
| `DD`, `D` | 14, 14 | `A`, `a` | AM or PM, am or pm |
| `dddd`, `ddd` | Saturday, Sat | `[text]` | `text`, with no token read inside |

Every other character is copied, and so is a letter that is not a token of this list: `{{date:Do}}` gives `14o`. A `[` that is never closed is copied.
