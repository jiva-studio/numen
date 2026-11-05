# Settings

Settings are stored in one JSON file named `numen.json` in the folder where this desktop keeps a person's configuration — `~/.config/numen/` on Linux, `~/Library/Application Support/numen/` on macOS, `%AppData%\numen\` on Windows. A run that finds no file writes one containing the settings that run is using, so the settings a person edits are the ones in effect.

Every field left out keeps its default. A file that sets one setting is valid.

A setting the window shows a control for is changed by that control, which patches the file as an object and leaves every other byte unchanged.

This document covers every section of the file, and states for each section which keys are worth changing and why. The complete list is the manual's [settings reference](../apps/docs/src/content/docs/reference.md), which is generated from the code and checked against it on every build. A plugin's settings belong to the plugin: its manifest declares them, the settings page shows them on the plugin's page, and the file stores them under `plugins`, keyed by the plugin's id. Where what a key does is specified elsewhere, the page that specifies it is linked from the section.

## Appearance

How the window is rendered: its size, which variant of a colour pair, and which palette.

```json
{
  "appearance": {
    "interface_scale": 1,
    "text_scale": 1,
    "mode": "system",
    "theme": "preset:numen"
  }
}
```

|                   |                                                                                                                                                                                                                                            |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `interface_scale` | how large the interface is — its chrome, its controls, the spacing between them and the type in them. 1 is the designed size, and the range is 0.8 to 2.                                                                                   |
| `text_scale`      | how large the text a person reads is: a note, a book, an answer, the editor. 1 is the designed size, and the range is 0.8 to 1.75.                                                                                                         |
| `mode`            | which variant of a colour pair is used: `system`, `light` or `dark`. Any other value is treated as `system`.                                                                                                                               |
| `theme`           | the stylesheet the window applies, named by its origin and its identifier: `preset:numen` ships inside the application, `mine:dracula` is the person's file, and `plugin:md.numen.themes.presets/dracula` is a theme of an enabled plugin. |

Each size is a multiplier. `interface_scale` is the root's font size, and every length in the window is a multiple of it: the height of a field and a button, the padding inside a node, the radii, the spacing, and the type throughout. `text_scale` is a second multiplier applied on top of it to the text a person reads — the editor and rendered markup, including headings, code, lists and quotations. A hairline, a border, a focus ring and the stroke of a handle have the same physical width under either.

A number outside the valid range is rejected. What the window uses instead is described in [Starting](starting.md). The number in the file is left as it was written.

A file with no size at all uses the desktop value: a session that set `GDK_DPI_SCALE` renders its interface at that scale, and a session without it uses 1. A scale outside the `interface_scale` range is not used as the initial value.

`-interface-scale` and `-text-scale` set a size for a single launch, overriding whatever is here. Each stays in effect until that size is chosen in the window, and choosing it writes it to the file.

`zoom` is the reader's term for how large a page of a document is shown. It is changed in the open document and it is not a setting.

The tokens a theme sets, where the person's themes are stored, and how a file declares itself light-and-dark or one variant only are in [Themes](themes.md).

A theme that sets `color-scheme` provides one variant, and `mode` has no effect while it is applied. A name matching nothing falls back to `preset:numen`, and the name that was not found is reported; this file is left as it is.

Choosing a theme, light or dark, or either size writes the matching field to this file — each is a separate command in the command palette. The file is read as an object, the named fields are set, and it is written back, so a key this build does not know is preserved. A file that does not parse is not written.

Whether a node in the plex nests the headings of its note, and how many, are settings of the Plex Graph plugin: `plugins.md.numen.views.plex.graph.nest_headings` and `heading_limit`.

## A day of review

The hour a day of review starts is the setting `day_start_time` of the Flashcards plugin, `md.numen.views.flashcards.studio`, stored under `plugins.md.numen.views.flashcards.studio`. How a deck is scheduled is not set here: it is set in the vault, in the preset the deck refers to — [Cards](cards.md).

```json
{
  "plugins": {
    "md.numen.views.flashcards.studio": {
      "day_start_time": "04:00"
    }
  }
}
```

|                  |                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------- |
| `day_start_time` | the hour a day of review begins, in local time (`04:00`, between `00:00` and `12:00`). |

An answer given before that hour is recorded in the previous day: a user reviewing cards at one in the morning is finishing the previous evening's session, and a boundary at midnight would split one session in two. The hour uses local time — [Flashcards](flashcards.md).

Anything that is not an hour of the day is reported, and `04:00` is used. The file is left as the person wrote it.

## What a note is called

`naming` controls whether a note's title and the name of its file are kept in sync.

```json
{
  "naming": {
    "sync_title_and_filename": true
  }
}
```

|                           |                                                           |
| ------------------------- | --------------------------------------------------------- |
| `sync_title_and_filename` | whether renaming either of the two updates the other. On. |

A note is shown by its `title`, else by its filename — [Note format](note-format.md). On, giving a note a different name renames its file after that name, and renaming its file writes the new name into the key where the note has one. Off, the two are independent in both directions: a new name is written into the note and the file is not moved, and a renamed file leaves the note as it was written.

A note with no `title` is named by its file, and nothing else in it can hold a name. Renaming such a note renames its file whichever way this is set, and no key is written into it unless the filename cannot hold the whole title.

Renaming a file into another folder is not renaming a note, and neither is renaming a folder. Neither changes what a note is called.

Typing a new `# Heading` into a note does not rename its file. A save writes the text a person typed and adds nothing to it.

`Choose title and file name sync` is a separate command in the palette. Choosing `On` or `Off` writes this field, and the next rename uses the value written.

## Performance profiles

`performance` sets how much of the machine the core's CPU-bound work and the Go runtime use. Settings of a plugin's own work, such as OCR or embedding, are that plugin's settings.

```json
{
  "performance": {
    "profile": "balanced"
  }
}
```

|           |                                                                                                           |
| --------- | --------------------------------------------------------------------------------------------------------- |
| `profile` | the workload profile: `eco`, `balanced` (default) or `maximum`. Any other value is treated as `balanced`. |

### Hardware adaptation

numen reads the machine and resolves every pool from it. The machine is described by its logical CPUs (hardware threads the process may run on, limited by the process affinity and, on Linux, the cgroup v2 CPU quota), its physical cores, its performance cores (the cores of the highest performance class; equal to the physical cores when none differ), and its installed and free memory. A figure that cannot be read is unknown and takes the fallback below.

The **core budget** is the number of hardware threads CPU-bound work may occupy at once: a share of the hardware threads of the performance cores. A performance core is counted with as many threads as the machine has logical CPUs per physical core, rounded up. When the core counts are unknown, every logical CPU counts as a core.

|                                  | Eco                     | Balanced _(default)_                       | Maximum                         |
| -------------------------------- | ----------------------- | ------------------------------------------ | ------------------------------- |
| Core budget                      | 1                       | 50% of the performance threads, at least 1 | 100% of the performance threads |
| Workers (general CPU-bound pool) | 1                       | half the budget, at least 1                | the budget                      |
| Memory limit (soft)              | 25% of installed memory | 50% of installed memory                    | 75% of installed memory         |

The core uses the workers value as the number of sources a search extracts at once; a search built without a plan extracts 4.

The memory limit is a soft limit on the memory the Go runtime manages, set once at startup from the installed memory: near it the collector runs more often, and no allocation fails. It is not set when the installed memory is unknown, when `GOMEMLIMIT` is set, or when a limit is already in force.

- **Eco**: Keeps background work unobtrusive and battery-friendly: one thread for each pool.
- **Balanced**: Optimized for everyday responsive use; half the performance threads, so the interface keeps the rest.
- **Maximum**: Full use of the performance cores for fast initial indexing and bulk processing.

`Set performance profile` is a separate command in the palette. Choosing a profile writes the configuration and applies it to the plugins that are running at once. The core sizes its own pools and the memory limit once, at start-up, so they take the profile after the next start of the application.

### Plugin background work

The core passes the profile to every running plugin, in each Configure call, as the value `host.performanceProfile` (`performance.Read` of `plugin-sdk-go` reads it, and `performance.Resolve` gives the threads and the niceness the table below lists). A change of the profile in the palette is pushed to the running plugins at once, so a plugin needs no restart; the core itself takes it at the next start. A profile that is not one of the three counts as `balanced`.

The Vector Embeddings plugin sizes the local model by it while it indexes the vault. The threads are the intra-op threads of the model and the workers that tokenize; the texts are the most texts in one pass. An `NUMEN_ONNX_THREADS` in the environment of the plugin wins over the threads of the profile.

|                     | Eco | Balanced _(default)_           | Maximum                |
| ------------------- | --- | ------------------------------ | ---------------------- |
| Threads             | 1   | a quarter of the cores, 1 to 4 | half the cores, 1 to 8 |
| Texts in one pass   | 2   | 4                              | 8                      |
| Niceness (priority) | 19  | 10                             | 0                      |

A model open when the profile changes closes when its running batches end, and the next batch opens it with the new threads. The niceness is set on every thread of the plugin process, and the system lets a process only lower its priority: a change to a profile that asks for a higher priority than the plugin has keeps the priority in force until the plugin restarts. The mock model of the tests takes the niceness but is not sized by the profile.

## How the index keeps vectors

The Vector Embeddings plugin creates the vectors: which model, where it runs, the key of a service and the longest chunk of a note are its own settings. The index stores the vectors the plugin creates and searches them, and this section is about the index alone.

The core settings file holds no setting of the chunk size. The plugin setting `chunkTokens` (default 256, from 16 to 8192, held to the input limit of the model) is reported to the core with the model, and the core cuts notes at it. With no embedding plugin, or one that names no size, notes are cut at the chunking default of 1000 characters. The index records the cut of each note, and a note recorded with another cut is read and cut again on the next scan; a change of the setting in an open window builds the session of the vault again. A scan that does not know the size cuts new and changed notes at the default and does not cut the others again.

The floor, how similar a passage has to be to a question to count as an answer, is not a setting in the file. The index stores one floor for each vault under each model and applies it to every search by meaning. A floor given in a query takes precedence over the one the index stores, which takes precedence over the initial floor the model's plugin reports, which takes precedence over 0.30. The Search Relevance plugin, `md.numen.tools.search.relevance`, measures the floor on the vault, lets a person set one, and stores the result in the index; with the plugin disabled the index keeps applying the last floor it stored. How similar a model scores two texts about different things is a property of that model, so a changed model has a floor of its own.

A vector is stored under the model the plugin reports: its name, width, input limit and pooling, a short hash of how the plugin handles the text (the prefixes it puts in front of a document and of a query, the model file, the token limit and the normalisation), and where it was made. Changing the model in the plugin, or its profile, recreates every stored vector, and changing back finds the old vectors still stored. A model that tells a question from the passage that answers it is asked for a document vector when the index is filled and for a query vector when a question is typed.

A local model the Vector Embeddings plugin does not list is added with a profile file: a JSON file in the `profiles` folder of the plugin data directory that gives the model's id, width, pooling, prefixes and files, merged over the profiles that ship with the plugin by `id`. The shape is described in the README of the plugin, and a model with no profile is refused with an error that names the folder. A vault with the plugin disabled is searched by its words alone.

## Correcting a text

The proofreading plugin, `md.numen.tools.proofreading`, corrects a transcript or a recognition when a person requests it from the file's menu, and its settings belong to the plugin. What a correction may change, and what rejects one, is in [Proofreading](proofreading.md).

The plugin has two ways to reach a model, chosen by its setting `provider`. With `agent`, the default, it requests a completion from the host through the host service `ai:complete`, and the host uses [`ai.default`](#where-a-plugins-questions-to-a-model-are-answered), which sets the agent program, the model and the number of questions answered at once. With `openrouter`, `openai` or `custom`, it calls an OpenAI-compatible chat service directly, with the `model` and, for `custom`, the `baseUrl` of its own settings and the `apiKey` secret kept in the system keychain. With no agent available, a proofreading through the agent fails with "unavailable", and no version is made.

```json
{
  "ai": {
    "default": { "agent": "claude", "model": "haiku", "concurrency": 2 }
  },
  "plugins": {
    "md.numen.tools.proofreading": {
      "provider": "agent",
      "batch_size": 40,
      "overlap": 4,
      "in_flight": 4,
      "max_edit_distance": 0.3
    }
  }
}
```

|                     |                                                                                                                                                                                                                                                                                                                                                           |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `batch_size`        | how many lines of a transcript one request contains. 40. A recognition is sent one page per request.                                                                                                                                                                                                                                                      |
| `overlap`           | how many lines neighbouring batches of a transcript share, so a phrase split at a batch boundary still appears whole in one of them. 4.                                                                                                                                                                                                                   |
| `in_flight`         | how many batches of a transcript one window contains; the task shows the lines done after each window. 4.                                                                                                                                                                                                                                                 |
| `max_edit_distance` | how far a correction of a recognised line may change the line's letters and still be accepted as a correction: the Levenshtein distance between what is left after spaces, punctuation, symbols, diacritics and case come off, as a share of the longer of the two. 0.30. A correction with a larger distance is dropped and that line is left as it was. |

`max_edit_distance` is 0.30 because the measured distribution has a gap there. Over 931 corrections of one book, every correction with a distance above 0.30 was damage — text taken from the next line, or one corrected word in place of a whole line — and every one below it was a correct correction. It is a setting because other books may differ; the figures are in [Performance](performance.md).

## Importing from an address

How a link's address is fetched is set by the Media Importer plugin's settings, under `plugins.md.numen.tools.media.importer`: `yt_dlp` and `ffmpeg`, the paths of the programs (empty is the one on `PATH`); `captions`, the languages captions are taken in, best first, separated by commas; `automatic_captions`, whether captions a machine wrote are used where a site publishes none (on); and `copy_max_size_mb`, the maximum size in megabytes of a copy of a video (500, and 0 is no limit). What is stored, and where, is [Importing an address](importing.md).

## Which agent answers

`agent` sets which agent answers in the panel and how each program is run. What an agent may request from a vault is in [Agents](agents.md).

```json
{
  "agent": {
    "use": "claude",
    "claude": {
      "command": [],
      "model": ""
    }
  },
  "plugins": {
    "md.numen.agents.claude": {
      "reads_hooks_and_skills": false
    }
  }
}
```

|                                                         |                                                                                                                                                                                                                                                                                                                                                                            |
| ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `use`                                                   | which agent answers. `claude` is Claude Code, used by starting it and reading its output. `numen` is Numen Agent, numen's own: the `md.numen.agents.numen` plugin sends requests to an OpenAI-compatible chat API and calls this vault's tools on behalf of the model. Empty means no agent, and the panel says so.                                                        |
| `claude.command`                                        | the command that starts it: the path of the command line program, and any wrapper it is started through. Empty searches the path, then the folders its installers write to. Worth setting for an installation those folders do not cover, and for a machine with several.                                                                                                  |
| `claude.model`                                          | which of its models answers — `opus`, `sonnet`, `haiku`, or a full name. Empty uses that installation's default model. Worth setting because the panel is read while somebody waits.                                                                                                                                                                                       |
| `plugins.md.numen.agents.claude.reads_hooks_and_skills` | whether it reads the configuration of this machine: hooks, skills, standing instructions in `CLAUDE.md`, plugins. Off. A hook is a shell command Claude Code runs itself, and a question typed into a panel is not a request to run one. On, what is configured for this person is read; what a vault contains is rejected either way, since a vault comes from elsewhere. |
| `numen.model`                                           | which model of the chat API answers. Empty takes the plugin's own `model` setting.                                                                                                                                                                                                                                                                                         |

How many requests Numen Agent may make to the model for one task before it is stopped is the plugin's own setting, `plugins.md.numen.agents.numen.maxSteps`, 25 unless it names another. Which chat API it uses is the plugin's own setting, `plugins.md.numen.agents.numen.baseUrl`: OpenRouter, `https://openrouter.ai/api/v1`, unless it names another, such as `http://localhost:11434/v1` for Ollama. Its key is the plugin's `apiKey` secret, kept in the system keychain; an endpoint on this machine may take none. A key is sent only over https, or over http to this machine. The agent counts as installed while the chat API lists its models, and the models it lists are the ones offered.

A section is kept whether or not it is in use, so trying another agent temporarily loses no settings.

## Where a plugin's questions to a model are answered

A plugin that holds the `ai:complete` permission requests a completion from the host and specifies no model, agent or key. The host identifies the plugin by the token it gave that plugin at launch, and uses `ai.default`. Nothing in a question selects where it is sent. The question is sent to the agent as a task that names no tool and no conversation, so the agent has none of the vault's tools, and the answer is the agent's output.

```json
{
  "ai": {
    "default": { "agent": "claude", "model": "haiku", "concurrency": 2 }
  },
  "plugins": {
    "md.numen.tools.proofreading": {
      "provider": "agent"
    }
  }
}
```

|                          |                                                                                                                                                                                                                                               |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ai.default.agent`       | the name of the agent program that answers the questions plugins ask (`claude`, `codex`, `antigravity`, `acp:<name>`, the names of the programs the plugins list). Empty uses the program `agent.use` names, else the default of the plugins. |
| `ai.default.model`       | the model that program answers with. Empty uses the model `agent.<program>.model` names, else the program's own. It applies to the default agent alone.                                                                                       |
| `ai.default.concurrency` | how many questions one agent answers at once; the others wait in a queue, and a question whose plugin gave up is removed from the queue. Below one is one.                                                                                    |

A program that is not installed or that no plugin lists returns `NO_ROUTE`, and so does a question when `agent.use` is empty or no program is installed; no other program is used instead. A program whose plugin cannot be reached, that stops without an answer or that produces no output returns `UNAVAILABLE`, with the reason it stopped in the log. The command line and the model of the program's section under `agent` apply to the task as they do to the panel's. A model the program does not offer is reported as the program's own error.

Every agent that runs a task answers a question, and a task that names no tool is given none. `md.numen.agents.claude` starts `claude` with no tool, no MCP server and one turn. `md.numen.agents.codex` starts `codex exec` read-only and ephemeral, without the user's configuration, the MCP servers in it and the tools of its own that can be turned off, in an empty folder. `md.numen.agents.antigravity` starts `agy` without the permission skip, with a home that holds no MCP server, in an empty folder. `md.numen.agents.numen` asks the chat API with no tool and connects to no tool server. `md.numen.agents.acp` is the generic ACP client: it lists a program `acp:<name>` for each section of that name under `agent` with a `command`, with the models the agent reports for a session, and answers a task that names no tool in one session of its own in an empty folder, with no MCP servers, no file system or terminal and every permission denied. It rejects a task that names a tool.

The file is read at every question, so a choice written while the application runs applies to the next one.

The settings page writes `agent.use` through the agent chat plugin, which chooses the agent of a conversation in a dropdown with its model and reasoning. The reasoning a new question starts with is the setting `reasoning` of the agent chat plugin, `plugins.md.numen.views.agent.chat.reasoning`: the name of one of the levels a model lists, empty for the default of the model. The agents in its lists are the programs the plugins list, and the models are the ones each program lists for `GetAgentPrograms`.

No key is written in this file. An agent plugin that needs one declares it under `secrets` in its manifest; the person enters, replaces and deletes it on the plugin's page in the settings, and the host stores it in the system keychain and passes it to the plugin in `Configure`, so it is in neither the plugin's environment nor its command line. The page shows whether a key is set and never shows the key. Under `plugins.<id>.storedSecrets` the host records only the names of the secrets that have a key, and reads the keychain for those alone. Storing a key on a machine without a keychain fails with an error saying so and keeps the key nowhere else; the plugin is then configured without it.

A setting of type `string` that declares `options` in its manifest is chosen from them in a list that can be searched, and its value can also be typed.
