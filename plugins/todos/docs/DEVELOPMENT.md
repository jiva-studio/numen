# Todos: Development Notes

For contributors. The user manual is the [README](../README.md).

## Layout

- `backend/todo`: parser and line edits (`ParseNote`, tick and untick). Fields are read from tokens of the task text; `rec:` is parsed but not part of the release.
- `backend/usecase/index`: the in-memory index of tasks and tags, persisted as a snapshot in the plugin's vault store folder, kept current from the file change feed and the `tags` frontmatter key.
- `backend/api`, `backend/wire`, `backend/service`: Connect-RPC services, the search functions and the tools.
- `frontend/editor`: the CodeMirror extension (checkbox, field colours, completion). `frontend/status`: the widget and its popover.

## Task lines

A task is a list line (`-`, `*`, `+` or a number with `.` or `)`) followed by `[c]` and text. `c` is a space (open), `x` or `X` (done), anything else is another status. Lines in code fences, HTML comments, frontmatter and indented code (4 or more columns outside a list) are skipped; quote markers are stripped, so a task in a quote counts. Fields are single tokens: leading `(A)` or `pri:A`, `due:`, `done:`, `rec:`, `+project`, `@context`. Tokens inside code spans and links stay in the text.

## Lines

`Line` counts lines of the file, frontmatter included. The editor counts from the first line below the frontmatter, so a task also carries `ProseLine`; the tick names its line by `isProseLine` and the backend converts it with the frontmatter of the note it reads.

## Writes

Every write reads the note, changes one line and writes the note over the file stat it was read with; a note that changed meanwhile is retried once, then refused. A tick sets the box to `x` and adds `done:` with today's date, or sets it to a space and removes `done:`.

## Query functions

Day-based functions use the local date of the machine. Open-only functions: `hasOpenTask`, `hasTaskOverdue`, `hasTaskDueToday`, `hasTaskDueWithin`, `hasTaskWithoutDue`, `hasTaskProject`, `hasTaskContext`, `hasTaskPriority`, `hasTaskText`. `hasDoneTask` matches done tasks; `hasTaskDueBefore` matches open and done tasks with a due date; a task with another status matches none. A bad date, negative `days`, empty name or text, or a priority that is not a letter fails the query. While the plugin is still reading notes, the answer carries a warning that it may be incomplete.

## Tools

The plugin has the capability `tools`. Tools and functions serve only the vault the window shows; a call naming another vault, or made while tasks are still being read, is refused.

- `task_list`: tasks with `path`, `line`, `proseLine`, `text`, `status`, `due`, `priority`, `projects`, `contexts`. Filters: `status` (`open`, `done`, `all`), `due` (`any`, `none`, `overdue`, `today`, `thisWeek`, `range` with `from` and `to`), `project`, `context`, `priority`, `path` (a note, or a folder ending in `/`), `text`, `limit` (100, at most 500).
- `task_add`: appends `- [ ] text` to an existing note.
- `task_set_done`: ticks or unticks the task at a line, which must still have `expectedText`.
- `task_edit`: replaces the text of the task at a line, fields included, keeping its checkbox; requires `expectedText`.
- `task_remove`: deletes the line; requires `expectedText`.

A write tool refuses a line that is no longer a task, or whose text differs from `expectedText`.

## Settings

`ignoredFolders` (list), `hideFields`, `showOverdue`, `showToday`, `showNextWeek`, `showNoDate`; see `manifest.json`. Next 7 days holds the open tasks due in the six days after today; a task due today is only in Today.
