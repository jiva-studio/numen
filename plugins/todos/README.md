# Todos

Lists the tasks written in notes: a widget shows how many are due or overdue, and a popover lists them.

<!-- more -->

## What You Can Do

- **See what is due**: The widget, in the title bar by default, shows the count of overdue tasks and tasks due today. Click it for the lists Overdue, Today and No date.
- **Open the note of a task**: Click the text of a task to open its note at its line.
- **Find notes by task in a query**: A query of the search language takes the functions below. A note holds when it has at least one task that satisfies the condition, and they combine with `&&`, `||` and `!` like `kind`.
- **Ask an agent about your tasks**: An agent lists, adds, ticks, edits and removes tasks with the tools below.

## Task lines

A task is a `- [ ]` line of a note. Fields are written in the line in the style of todo.txt: `due:2026-11-20`, `done:2026-11-14`, `rec:1w`, `pri:A`, a leading `(A)`, `+project` and `@context`. Lines in code fences, HTML comments and frontmatter are not tasks.

## Query functions

Each function is a condition on a note. The day is the local date of the machine.

| Function                         | The note has                                                       |
| -------------------------------- | ------------------------------------------------------------------ |
| `hasOpenTask()`                  | an open task                                                       |
| `hasDoneTask()`                  | a done task                                                        |
| `hasTaskOverdue()`               | an open task due before today                                      |
| `hasTaskDueToday()`              | an open task due today                                             |
| `hasTaskDueWithin(days)`         | an open task due from today to `days` days ahead, `days` 0 or more |
| `hasTaskDueBefore("2026-12-31")` | an open or done task due on or before the date                     |
| `hasTaskWithoutDue()`            | an open task with no due date                                      |
| `hasTaskProject("name")`         | an open task of the project `+name`                                |
| `hasTaskContext("name")`         | an open task of the context `@name`                                |
| `hasTaskPriority("A")`           | an open task of the priority, one letter A to Z                    |
| `hasTaskText("text")`            | an open task whose text contains the text, without case            |

A date that is not `YYYY-MM-DD`, negative `days`, an empty name or text, and a priority that is not a letter fail the query with a message at the call. While the plugin is still reading the notes, an answer carries a warning that it may be incomplete.

## Tools

The plugin has the capability `tools`. Tools and query functions work the vault of the call; a call that names no vault is refused, and the plugin keeps one index of tasks for each vault a call names.

- `task_list`: Lists tasks with `path`, `line`, `text`, `status`, `due`, `priority`, `projects` and `contexts`. Filters: `status` (`open`, `done`, `all`), `due` (`any`, `none`, `overdue`, `today`, `thisWeek`, `range` with `from` and `to`), `project`, `context`, `priority`, `path` (a note, or a folder ending in `/`), `text` and `limit` (100, at most 500).
- `task_add`: Appends `- [ ] text` to an existing note.
- `task_set_done`: Ticks or unticks the task at a line. A tick adds `done:` with today's date.
- `task_edit`: Replaces the text of the task at a line, its fields included, and keeps its checkbox.
- `task_remove`: Deletes the line of a task.

A task is addressed by note path and line from `task_list`. Lines count the lines of the file, frontmatter included; the note editor counts from the first line below the frontmatter, so a task also carries a `proseLine` and the checkbox tick names its line that way (`isProseLine`), which the backend converts with the frontmatter of the note it reads. The write tools refuse a line that is no longer a task, and `task_edit` and `task_remove` require `expectedText`, the text of the task, to refuse a task whose text changed. Every write reads the note, changes the line and writes the note over the file stat it was read with, as the checkbox tick does; a note that changed meanwhile is retried once, then refused.

## Settings

- **Ignored folders**: Folders whose notes are not searched for tasks, comma separated.
- **Hide task markers until the line is edited**: A task keeps its markers (`due:`, `done:`, `pri:`, `+project`, `@context`, `#tag`), but they are drawn only while the cursor is on its line.
- **Overdue**, **Today**, **Next 7 days**, **No date**: Which groups the popup of the widget lists and the chip counts. All four are on by default. Next 7 days holds the open tasks due in the six days after today; a task due today is only in Today.
