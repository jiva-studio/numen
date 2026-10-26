# Settings

One file, JSON, named `numen.json` in the folder this desktop keeps a person's configuration in — `~/.config/numen/` on Linux, `~/Library/Application Support/numen/` on a mac, `%AppData%\numen\` on Windows. A run that finds none writes it, holding exactly what that run is doing, so the settings a person changes are the ones in front of them.

Every field left out keeps its default. A file naming one setting is a valid file.

A setting the window draws a control for is turned by that control, which patches the file as an object and leaves every other byte of it where it was. The window also opens the file whole, in a tab of its own, from the settings page: what is typed there is written as it stands, and a malformed file is rejected with the syntax error location.

This document is every section of the file, and says of each section which keys are worth turning and why. The complete list is the manual's [settings reference](../apps/docs/src/content/docs/reference.md), which is written out of the code itself and checked against it on every build — `indexing.recognition` alone holds twenty-six keys, and three of them are below. Where what a key does is specified elsewhere, the page that specifies it is linked from the section.

## Appearance

How the window is drawn: how large, which half of a colour pair, and which palette.

```json
{
  "appearance": {
    "interface_scale": 1,
    "text_scale": 1,
    "mode": "system",
    "theme": "preset:numen",
    "nest_headings_under_notes": true,
    "nested_heading_limit": 6
  }
}
```

| | |
| --- | --- |
| `interface_scale` | how large the interface is drawn — its chrome, its controls, the spacing between them and the type in them. 1 is as designed, and it goes from 0.8 to 2. |
| `text_scale` | how large the text a person reads is set: a note, a book, an answer, the editor. 1 is as designed, and it goes from 0.8 to 1.75. |
| `mode` | which half of a colour pair is read: `system`, `light` or `dark`. Any other word is read as `system`. |
| `theme` | the stylesheet the window wears, named by its origin and its filename: `preset:dracula` ships inside the application, `mine:dracula` is the person's file. |
| `nest_headings_under_notes` | whether a node nests the headings of the note it stands for under its box. On. |
| `nested_heading_limit` | how many of those headings stand under a node at once, the rest being reached by scrolling. 6, and it goes from 1 to 12. |

Each size is a multiplier. `interface_scale` is the root's font size, and every length in the window is a multiple of it: the height of a field and a button, the clearance inside a node, the radii, the spacing, and the type throughout. `text_scale` is a second multiplier over the text a person reads — the editor, and marked-up text with the scale above it, its headings, code, lists and quotations. A hairline, a border, a focus ring and the stroke of a handle are one physical line under either.

A number outside the valid range is rejected. What stands in the window's place is in [Starting](starting.md). The number is left as it was written.

A file naming no size at all asks the desktop: a session that set `GDK_DPI_SCALE` draws its interface by that, and one that said nothing is drawn at 1. A scale outside what `interface_scale` goes to is not one it is seeded with.

`-interface-scale` and `-text-scale` say a size for a single launch, over whatever is here. Each stands until that size is chosen in the window, and choosing it is what writes it down.

A file naming `appearance.zoom` is drawn at that number, and the field is given the name `appearance.interface_scale` where it is written. Its value, its place among the fields around it and every other byte of this file stay as they were.

A `zoom` outside what `interface_scale` goes to keeps its name and is said, and the window is drawn as designed until a number in range is written. `zoom: 0` names no size, so the desktop is asked. A file naming a size under both names is drawn at `interface_scale` and keeps both names. A file nothing may be written into is read the same way at every launch, and drawn at the same size.

`zoom` is the reader's word: how large a page of a document is drawn. It is turned in the document in front of a person and it is not a setting.

The tokens a theme sets, where the person's themes live, and how a file names itself light-and-dark or one half only are in [Themes](themes.md).

A theme pinning `color-scheme` is published in one half, and `mode` has nothing left to choose while it is worn. A name matching nothing wears `preset:numen`, and the name that was not found is said; this file is left as it is.

Choosing a theme, light or dark, or either size writes the field it names back here — each is a command of its own in the palette, over the window. The file is read as an object, the named fields are set, and it is written back, so a key this build knows nothing about comes through the write unchanged. A file that does not parse is not written.

On, a node in the plex nests the headings of the note it stands for under its box, each at the line it stands on, and choosing one puts that note in front of the person with the keyboard on that line. Off, a node is its box alone, and nothing is asked of the vault about what a note is divided into.

`Nest headings under notes` is a command of its own in the palette, over the window. Choosing `On` or `Off` writes this field back here, and every plex the window holds is drawn again from what was written.

## A day of review

`review` is what a day of review is, on this person's clock. How a deck is scheduled is not here: it is in the vault, in the preset the deck points at — [Cards](cards.md).

```json
{
  "review": {
    "day_start_time": "04:00"
  }
}
```

| | |
| --- | --- |
| `day_start_time` | the hour a day of review begins at, in local time (`04:00`, between `00:00` and `12:00`). |

An answer given before that hour is recorded in the previous day: a user reviewing cards at one in the morning is finishing the previous evening's session, and a boundary at midnight would split one session in two. The hour uses local time — [Flashcards](flashcards.md).

Anything that is not an hour of the day is said, and `04:00` stands. The file is left as the person wrote it.

## What a note is called

`naming` is how a note's title and the name of its file are held together.

```json
{
  "naming": {
    "sync_title_and_filename": true
  }
}
```

| | |
| --- | --- |
| `sync_title_and_filename` | whether renaming either of the two brings the other into line. On. |

A note is shown by its `title`, else by its filename — [Note format](note-format.md). On, giving a note a different name renames its file after that name, and renaming its file writes the new name into the key where the note carries one. Off, the two are told apart in both directions: a new name is written into the note and the file stays where it is, and a renamed file leaves the note as it was written.

A note carrying no `title` is named by its file, and nothing else in it can carry a name. Renaming such a note renames its file whichever way this is set, and no key is written into it unless the filename cannot carry the whole title.

Renaming a file into another folder is not renaming a note, and neither is renaming a folder. Neither changes what a note is called.

Typing a new `# Heading` into a note does not rename its file. A save puts down the text a person typed and adds nothing to it.

`Sync title and filename` is a command of its own in the palette, over the window. Choosing `On` or `Off` writes this field back here, and the next rename reads what was written.

## Performance profiles

`performance` governs how intensively background AI pipelines (vector embedding, OCR, transcription, and agent concurrency) utilize system hardware.

```json
{
  "performance": {
    "profile": "balanced",
    "custom": {
      "embedding_batch_size": 32,
      "ocr_threads": 4,
      "ocr_sessions": 1,
      "llm_concurrency": 2
    }
  }
}
```

| | |
| --- | --- |
| `profile` | the workload profile: `eco`, `balanced` (default), `maximum`, or `custom`. |
| `custom.embedding_batch_size` | how many passages are embedded in a single forward pass when `profile` is `custom`. |
| `custom.ocr_threads` | intra-op thread count per OCR model instance. |
| `custom.ocr_sessions` | concurrent OCR inference workers. |
| `custom.llm_concurrency` | maximum simultaneous LLM/agent requests in flight. |

### Hardware adaptation

When `profile` is `eco`, `balanced`, or `maximum`, numen reads the machine and resolves every pool from it. The machine is described by its logical CPUs (hardware threads the process may run on, limited by the process affinity and, on Linux, the cgroup v2 CPU quota), its physical cores, its performance cores (the cores of the highest performance class; equal to the physical cores when none differ), and its installed and free memory. A figure that cannot be read is unknown and takes the fallback below.

The **core budget** is the number of hardware threads CPU-bound work may occupy at once: a share of the hardware threads of the performance cores. A performance core is counted with as many threads as the machine has logical CPUs per physical core, rounded up. When the core counts are unknown, every logical CPU counts as a core.

| | Eco | Balanced *(default)* | Maximum |
| --- | --- | --- | --- |
| Core budget | 1 | 50% of the performance threads, at least 1 | 100% of the performance threads |
| OCR sessions × threads | 1 × 1 | the budget, split as below | the budget, split as below |
| Workers (general CPU-bound pool) | 1 | half the budget, at least 1 | the budget |
| Embedding batch | 16 | 1 per 256 MB free, from 16 to 64 | 1 per 128 MB free, from 32 to 256 |
| LLM concurrency | 1 | 2 | 4 |
| Memory limit (soft) | 25% of installed memory | 50% of installed memory | 75% of installed memory |

**Where each figure is read.** The resolved plan is the source of every limit below; a limit the user sets in a section of its own wins over it, and a component built without a plan keeps the fallback in the last column.

| Consumer | Limit | Fallback without a plan |
| --- | --- | --- |
| Local embedding batch | embedding batch, unless the model names `batch_texts` | the model's own |
| Local embedding, threads of the loaded model | core budget, unless the model names `threads` | the default of the runtime |
| Local embedding, forward passes at once | workers, lowered to what a quarter of the available memory holds, unless the model names `sessions`; one on an accelerator | one up to 2 CPUs, two up to 4, three beyond |
| Search question and indexing sharing one local model | indexing holds one pass fewer than the model runs at once, at least one; a question is never held | not applicable |
| OCR | OCR threads and sessions | unresolved counts, then named defaults |
| Proofreading, batches of a transcript one window carries | the plugin's `in_flight` | 4 |
| Embedding service, requests in flight | LLM concurrency | 4 |
| Search, sources extracted at once | workers | 4 |
| Cards, decks parsed and curve places worked out at once | workers | one per CPU |
| PDF worker pool | workers, at least 2; the workers kept alive between documents are the planned workers, lowered to what 5% of the available memory holds at 32 MB each, none when it is unknown, and all are released after 5 minutes without an open document | one per CPU, at least 2; none kept alive |
| Transcription, resampling and feature extraction | workers | one per CPU |

OCR threads × sessions never exceeds the core budget. The budget is split into 4 sessions, or more when one session would need over 8 threads; sessions are capped at 16 and at the budget, and each session gets the budget divided by the sessions, at most 8 threads.

The memory limit is a soft limit on the memory the Go runtime manages, set once at startup from the installed memory: near it the collector runs more often, and no allocation fails. It is not set when the installed memory is unknown, when `GOMEMLIMIT` is set, or when a limit is already in force. Custom follows the Balanced share.

Free memory is what the operating system reports as available. When only installed memory is known, half of it is taken as free; when neither is known, 4096 MB is taken as free. macOS does not report free memory, so it takes the second rule.

- **Eco**: Keeps background work unobtrusive and battery-friendly: one thread for each pool.
- **Balanced**: Optimized for everyday responsive use; half the performance threads, so the interface keeps the rest.
- **Maximum**: Full use of the performance cores for fast initial indexing and bulk processing.
- **Custom**: Uses the explicit parameters written in `performance.custom`, clamped to their bounds: batch size up to 512 (32 when unset), OCR threads up to 64, OCR sessions up to 16, LLM concurrency from 1 to 16. When both `ocr_threads` and `ocr_sessions` are set they are used as written, even if their product exceeds the core budget. When one is unset, it is derived so that the product fills the Maximum budget; when both are unset, the Maximum split applies. Workers follow the Balanced rule.

`Set performance profile` is a command of its own in the palette, over the window. Choosing a profile updates the configuration immediately.

## How the index keeps vectors

The Vector Embeddings plugin makes the vectors: which model, where it runs, and the key of a service are its own settings. The index keeps the vectors the plugin makes and searches them, and this section is about the index alone.

```json
{
  "indexing": {
    "embedding": {
      "max_tokens": 256,
      "floor": 0.0
    }
  }
}
```

| | |
| --- | --- |
| `max_tokens` | the input bound, in tokens, a chunk that carries a vector is cut below. A chunk cut longer than the model reads is a chunk whose vector describes text it does not hold. Zero cuts at the chunking defaults. |
| `floor` | how near a question a passage stands to be an answer, in cosine similarity. Zero takes what the search was built against. Where a model puts two pieces of text about different things is a fact about that model, so a model changed is a floor measured again. |

A vector is kept under the model the plugin answers with: its name, width, input bound and pooling, and where it was made. A model changed in the plugin makes every stored vector again, and going back finds the old vectors where they were. A vault with the plugin disabled is searched by its words alone.

## Putting a text right

The proofreading plugin, `md.numen.tools.proofreading`, puts a transcript or a recognition right when a person asks for it from the file's menu, and its settings are the plugin's own. What a correction may change, and what refuses one, is in [Proofreading](proofreading.md).

Which model answers is not the plugin's to say: it asks the host, and the host answers from the plugin's own `agent` setting, else from [`ai.default`](#where-a-plugins-questions-to-a-model-are-answered), which names the agent program, the model and the number of questions answered at once. With no agent to answer, proofreading fails with "unavailable"; before the person lets the plugin ask, it fails with "AI access not allowed — allow it in Settings → AI". Either way no version is made.

```json
{
  "ai": {
    "default": { "agent": "claude", "model": "haiku", "concurrency": 2 }
  },
  "plugins": {
    "md.numen.tools.proofreading": {
      "aiConsent": true,
      "batch_size": 40,
      "overlap": 4,
      "in_flight": 4,
      "max_edit_distance": 0.30
    }
  }
}
```

| | |
| --- | --- |
| `batch_size` | how many lines of a transcript one request carries. 40. A recognition is asked about a page a request. |
| `overlap` | how many lines neighbouring batches of a transcript share, so a phrase torn at a batch boundary is still seen whole by one of them. 4. |
| `in_flight` | how many batches of a transcript one window carries; the task shows the lines done after each window. 4. |
| `max_edit_distance` | how far a correction of a recognised line may move the line's letters and still be a correction: the Levenshtein distance between what is left after spaces, punctuation, symbols, diacritics and case come off, as a share of the longer of the two. 0.30. A correction standing further apart is dropped and that line is left as it was. |

`max_edit_distance` is 0.30 because the measured distribution has a hole there. Over 931 corrections of one book, every correction standing further apart than 0.30 was damage — text dragged in from the next line, or one corrected word in place of a whole line — and every one below it was a correction. It is a setting because the next book is not that book; the figures are in [Performance](performance.md).

## Reaching an address

How a link's address is reached is the Media Importer plugin's settings, under `plugins.md.numen.tools.media.importer`: `yt_dlp` and `ffmpeg`, the paths of the programs (empty is the one on `PATH`); `captions`, the languages captions are taken in, best first, separated by commas; `automatic_captions`, whether captions a machine wrote count where a site publishes none (on); and `copy_max_size_mb`, how many megabytes a copy of a video may take (500, and 0 is no limit). What is kept, and where, is [Importing an address](importing.md).

## Which agent answers

`agent` is which agent answers in the panel, what it may reach, and whether the tools go on a port. What an agent may ask of a vault is in [Agents](agents.md).

```json
{
  "agent": {
    "use": "claude",
    "serve_tools": false,
    "claude": {
      "command": [],
      "model": "",
      "max_steps": 30,
      "reads_hooks_and_skills": false
    }
  }
}
```

| | |
| --- | --- |
| `use` | which agent answers. `claude` is Claude Code, reached by starting it and reading what it prints. `numen` is Numen Agent, numen's own: the `md.numen.agents.numen` plugin asks an OpenAI-compatible chat API and calls this vault's tools for the model. Empty answers with none, and the panel says so. |
| `serve_tools` | whether the tools go on a port, which is how an agent a person runs themselves reaches this vault. Off. The agent `use` names is served either way, so an installation naming one has the port open for it. |
| `claude.command` | what starts it: the command line's path, and anything it is started through. Empty asks the path, then the folders its installers write to. Worth naming for an installation those folders do not cover, and for one machine carrying several. |
| `claude.model` | which of its models answers — `opus`, `sonnet`, `haiku`, or a full name. Empty takes whatever that installation answers with. Worth naming because a panel is read while somebody waits. |
| `claude.max_steps` | how many times it may go to the model before it is stopped. 30. |
| `claude.reads_hooks_and_skills` | whether it reads what this machine holds configured for it: hooks, skills, standing instructions in `CLAUDE.md`, plugins. Off. A hook is a shell command Claude Code runs itself, and a question typed into a panel is not asking for one. On, what is configured for this person is read; what a vault carries is rejected either way, since a vault arrives from elsewhere. |
| `numen.model` | which model of the chat API answers. Empty takes the plugin's own `model` setting. |
| `numen.max_steps` | how many requests it may make to the model before it is stopped. 25. |

Which chat API Numen Agent asks is the plugin's own setting, `plugins.md.numen.agents.numen.baseUrl`: OpenRouter, `https://openrouter.ai/api/v1`, unless it names another, such as `http://localhost:11434/v1` for Ollama. Its key is the plugin's `apiKey` secret, kept in the system keychain; an endpoint on this machine may take none. A key is sent only over https, or over http to this machine. The agent counts as installed while the chat API lists its models, and the models it lists are the ones offered.

A section is kept whether it is the one in use or not, so trying another agent for an afternoon costs nothing.

An installation naming no agent and asking for no tools opens no port and writes no token file: a person who never asked for an agent is running a window and nothing else. Where the port is open, `-mcp-addr` says which address it answers on for a single launch and `-no-mcp` shuts it for one, whatever this file says.


## Where a plugin's questions to a model are answered

A plugin that holds the `ai:complete` permission asks the host for a completion and names neither a model, an agent nor a key. The host knows which plugin asks by the token it gave that plugin at launch, and answers from that plugin's own agent setting, else from `ai.default`. Nothing in a question chooses where it goes. The question reaches the agent as a task that names no tool and no conversation, so the agent has none of the vault's tools, and the answer is what the agent said.

```json
{
  "ai": {
    "default": { "agent": "claude", "model": "haiku", "concurrency": 2 }
  },
  "plugins": {
    "md.numen.tools.proofreading": {
      "agent": { "agent": "claude", "model": "sonnet" },
      "aiConsent": true
    }
  }
}
```

| | |
| --- | --- |
| `ai.default.agent` | the name of the agent program that answers a plugin whose own setting names none (`claude`, `codex`, `antigravity`, `acp:<name>`, the names the AI section lists). Empty is the program `agent.use` names, else the default of the plugins. |
| `ai.default.model` | the model that program answers with. Empty is the model `agent.<program>.model` names, else the program's own. It goes with the default agent alone. |
| `ai.default.concurrency` | how many questions one agent answers at once; the others wait for a place, and a question whose plugin gave up leaves the wait. Below one is one. |
| `plugins.<id>.<key>` | a plugin's own choice, under the key of the setting of type `agent` its manifest declares: `agent` and `model`. An empty `agent` is `ai.default`, agent and model both; an empty `model` is the chosen agent's own. |
| `plugins.<id>.aiConsent` | whether the person lets the plugin ask a model: `true` or `false`. Left out, the person has not answered. |

A program that is not installed or that no plugin lists is answered `NO_ROUTE`, and so is a question when `agent.use` is empty or no program is installed; no other program answers in its place. A program whose plugin cannot be reached, that stops without an answer or that says nothing is answered `UNAVAILABLE`, with the reason it stopped for in the log. The command line, the options and the model of the program's section under `agent` apply to the task as they do to the panel's. A model the program does not offer is the program's own error.

Every agent that runs a task answers a question, and a task that names no tool is given none. `md.numen.agents.claude` starts `claude` with no tool, no MCP server and one turn. `md.numen.agents.codex` starts `codex exec` read-only and ephemeral, without the user's configuration, the MCP servers in it and the tools of its own that can be turned off, in an empty folder. `md.numen.agents.antigravity` starts `agy` without the permission skip, with a home that holds no MCP server, in an empty folder. `md.numen.agents.numen` asks the chat API with no tool and connects to no tool server. `md.numen.agents.acp` is the generic ACP client: it lists a program `acp:<name>` for each section of that name under `agent` with a `command`, with the models the agent reports for a session, and answers a task that names no tool in one session of its own in an empty folder, with no MCP servers, no file system or terminal and every permission denied. It stops a task that names a tool.

The file is read at every question, so a choice written while the application runs applies to the next one.

The first question of a plugin the person has not answered for is rejected with `NO_CONSENT`, and the settings page puts it to the person: the AI section names the plugin that asked, with Allow and Deny. If denied, every question of that plugin is answered `NO_CONSENT`, which the plugin shows as the reason nothing was answered; the proofreading plugin shows it as the error of its task in the list of what is being done.

The settings page writes all of these. Its AI section holds the default agent and its model, a switch for each plugin that may ask, and the keys of the secrets plugins declare; a plugin's own section draws its setting of type `agent` as two lists, the agent first, "Default (<the agent answering it now>)" at its head, then the model, "Agent default" at its head. The agents a list offers are the programs the plugins list, and the models are the ones each program lists for `GetAgentPrograms`.

A file of an older shape, holding `ai.routes`, still loads: its `routes.default` is read as `ai.default` where the file names no `ai.default`, and the routes of single purposes are not read.

No key is written in this file. An agent plugin that needs one declares it under `secrets` in its manifest; the person enters, replaces and deletes it in the AI section, and the host keeps it in the system keychain and hands it to the plugin in `Configure`, so it is in neither the plugin's environment nor its command line. The page shows whether a key is set and never shows the key. Under `plugins.<id>.storedSecrets` the host records only the names of the secrets that have a key, and reads the keychain for those alone. Storing a key on a machine without a keychain fails with an error saying so and keeps the key nowhere else; the plugin is then configured without it.

A setting of type `string` that names a program in `modelsOf` in its manifest is chosen from the models that program lists, in a list that can be searched; the plugin's own `model` setting names the program it runs. It is typed by hand while the program lists no model, such as when the chat API cannot be reached.
