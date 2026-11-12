# Settings

Settings are stored in one JSON file named `numen.json` in the folder where this desktop keeps a person's configuration — `~/.config/numen/` on Linux, `~/Library/Application Support/numen/` on macOS, `%AppData%\numen\` on Windows. A run that finds no file writes one containing the settings that run is using, so the settings a person edits are the ones in effect.

The first level of the file is the version `v` and the owners of settings. The owner `app` holds the settings of the application, and every other key is the id of a plugin with the settings of that plugin below it, next to the field `enabled` that says whether the plugin runs. Every field left out keeps its default. A file that sets one setting is valid.

A value that the declaration of its setting does not take is read and ignored: a key that is not declared, a value of the wrong type, a number outside its range, a word that is none of the options, a theme that is not installed, or an object of a plugin that is not installed. The application starts with the default in force, shows a notice, and does not rewrite the file because of it. A write that would store such a value is refused.

A file in the layout of an earlier build, with the sections `appearance`, `naming` or `plugins` at the first level, is not read: delete it and the application writes a new one with the defaults.

A setting of a plugin is written in camelCase (`executablePath`, `softWrap`). A key of an earlier layout, such as a value saved under a snake_case name or under `importing` or `review`, is ignored with a notice and the default applies; nothing moves it into the plugin, because no release has been published. A file that is not JSON stops the start, and the message names the file and the byte where it stops reading as JSON.

A setting the window shows a control for is changed by that control, which patches the file as an object and leaves every other byte unchanged.

This document covers every section of the file, and states for each section which keys are worth changing and why. The complete list is the manual's [settings reference](../apps/docs/src/content/docs/reference.md), which is generated from the code and checked against it on every build. A plugin's settings belong to the plugin: its manifest declares them, the settings page shows them on the plugin's page, and the file stores them in the object named by the plugin's id. Where what a key does is specified elsewhere, the page that specifies it is linked from the section.

## Appearance

How the window is rendered: its size, which variant of a colour pair, and which palette.

```json
{
  "app": {
    "appearance": {
      "interfaceScale": 1,
      "textScale": 1,
      "mode": "system",
      "theme": "preset:numen"
    }
  }
}
```

|                  |                                                                                                                                                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `interfaceScale` | how large the interface is — its chrome, its controls, the spacing between them and the type in them. 1 is the designed size, and the range is 0.8 to 2.                                                                                   |
| `textScale`      | how large the text a person reads is: a note, a book, an answer, the editor. 1 is the designed size, and the range is 0.8 to 1.75.                                                                                                         |
| `mode`           | which variant of a colour pair is used: `system`, `light` or `dark`. Any other value is ignored, with a notice, and `system` stands.                                                                                                       |
| `theme`          | the stylesheet the window applies, named by its origin and its identifier: `preset:numen` ships inside the application, `mine:dracula` is the person's file, and `plugin:md.numen.themes.presets/dracula` is a theme of an enabled plugin. |

Each size is a multiplier. `interfaceScale` is the root's font size, and every length in the window is a multiple of it: the height of a field and a button, the padding inside a node, the radii, the spacing, and the type throughout. `textScale` is a second multiplier applied on top of it to the text a person reads — the editor and rendered markup, including headings, code, lists and quotations. A hairline, a border, a focus ring and the stroke of a handle have the same physical width under either.

A number outside the valid range is ignored, with a notice, and the default stands. The number in the file is left as it was written. A write of such a number is refused.

A file with no size at all uses the desktop value: a session that set `GDK_DPI_SCALE` renders its interface at that scale, and a session without it uses 1. A scale outside the `interfaceScale` range is not used as the initial value.

`-interface-scale` and `-text-scale` set a size for a single launch, overriding whatever is here. Each stays in effect until that size is chosen in the window, and choosing it writes it to the file.

`zoom` is the reader's term for how large a page of a document is shown. It is changed in the open document and it is not a setting.

The tokens a theme sets, where the person's themes are stored, and how a file declares itself light-and-dark or one variant only are in [Themes](themes.md).

A theme that sets `color-scheme` provides one variant, and `mode` has no effect while it is applied. A name matching nothing falls back to `preset:numen`, and the name that was not found is reported; this file is left as it is. A write of a theme that is not installed is refused.

Choosing a theme, light or dark, or either size writes the matching field to this file — each is a separate command in the command palette. The file is read as an object, the named fields are set, and it is written back, so a key this build does not know is preserved. A file that does not parse is not written.

Whether a node in the plex nests the headings of its note, and how many, are settings of the Plex Graph plugin: `md.numen.views.plex.graph.nestHeadings` and `headingLimit`, stored in the object of that plugin.

## A day of review

The hour a day of review starts is the setting `dayStartTime` of the Flashcards plugin, `md.numen.views.flashcards.studio`, stored in the object `md.numen.views.flashcards.studio`. How a deck is scheduled is not set here: it is set in the vault, in the preset the deck refers to — [Cards](cards.md).

```json
{
  "md.numen.views.flashcards.studio": {
    "dayStartTime": "04:00"
  }
}
```

|                |                                                                                        |
| -------------- | -------------------------------------------------------------------------------------- |
| `dayStartTime` | the hour a day of review begins, in local time (`04:00`, between `00:00` and `12:00`). |

An answer given before that hour is recorded in the previous day: a user reviewing cards at one in the morning is finishing the previous evening's session, and a boundary at midnight would split one session in two. The hour uses local time — [Flashcards](flashcards.md).

Anything that is not an hour of the day is reported, and `04:00` is used. The file is left as the person wrote it.

## What a note is called

`app.naming` controls whether a note's title and the name of its file are kept in sync.

```json
{
  "app": {
    "naming": {
      "syncTitleAndFilename": true
    }
  }
}
```

|                        |                                                           |
| ---------------------- | --------------------------------------------------------- |
| `syncTitleAndFilename` | whether renaming either of the two updates the other. On. |

A note is shown by its `title`, else by its filename — [Note format](note-format.md). On, giving a note a different name renames its file after that name, and renaming its file writes the new name into the key where the note has one. Off, the two are independent in both directions: a new name is written into the note and the file is not moved, and a renamed file leaves the note as it was written.

A note with no `title` is named by its file, and nothing else in it can hold a name. Renaming such a note renames its file whichever way this is set, and no key is written into it unless the filename cannot hold the whole title.

Renaming a file into another folder is not renaming a note, and neither is renaming a folder. Neither changes what a note is called.

Typing a new `# Heading` into a note does not rename its file. A save writes the text a person typed and adds nothing to it.

`Sync title and filename` is a command in the palette, made from this declaration. Choosing `On` or `Off` writes this field, and the next rename uses the value written.

## Performance profiles

`app.performance` sets how much of the machine the core's CPU-bound work and the Go runtime use. Settings of a plugin's own work, such as OCR or embedding, are that plugin's settings.

```json
{
  "app": {
    "performance": {
      "profile": "balanced"
    }
  }
}
```

|           |                                                                                                                                   |
| --------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `profile` | the workload profile: `eco`, `balanced` (default) or `maximum`. Any other value is ignored, with a notice, and `balanced` stands. |

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

`Performance profile` is a command in the palette, made from this declaration. Choosing a profile writes the configuration and applies it to the plugins that are running at once. The core sizes its own pools and the memory limit once, at start-up, so they take the profile after the next start of the application.

### Plugin background work

The core passes the profile to every running plugin, in each Configure call, as the value `host.performanceProfile` (`performance.Read` of `plugin/sdk-go` reads it, and `performance.Resolve` gives the threads and the niceness the table below lists). A change of the profile in the palette is pushed to the running plugins at once, so a plugin needs no restart; the core itself takes it at the next start. A profile that is not one of the three is refused when it is written, and ignored with a notice when the file holds it.

The Vector Embeddings plugin sizes the local model by it while it indexes the vault. The threads are the intra-op threads of the model and the workers that tokenize; the texts are the most texts in one pass. An `NUMEN_ONNX_THREADS` in the environment of the plugin wins over the threads of the profile.

|                     | Eco | Balanced _(default)_           | Maximum                |
| ------------------- | --- | ------------------------------ | ---------------------- |
| Threads             | 1   | a quarter of the cores, 1 to 4 | half the cores, 1 to 8 |
| Texts in one pass   | 2   | 4                              | 8                      |
| Niceness (priority) | 19  | 10                             | 0                      |

A model open when the profile changes closes when its running batches end, and the next batch opens it with the new threads. The niceness is set on every thread of the plugin process, and the system lets a process only lower its priority: a change to a profile that asks for a higher priority than the plugin has keeps the priority in force until the plugin restarts. The mock model of the tests takes the niceness but is not sized by the profile.

## How the index keeps vectors

The Vector Embeddings plugin creates the vectors: its setting Source chooses the local model or a connection, and the longest chunk of a note is its own setting. Source "Local model" keeps the settings of the local model, including the model and the performance profile that sizes its threads. Source "Connection" shows one row, a connection and an embedding model, and the plugin holds no key and no address. The index stores the vectors the plugin creates and searches them, and this section is about the index alone.

The core settings file holds no setting of the chunk size. The plugin setting `chunkTokens` (default 256, from 16 to 8192, held to the input limit of the model) is reported to the core with the model, and the core cuts notes at it. With no embedding plugin, or one that names no size, notes are cut at the chunking default of 1000 characters. The index records the cut of each note, and a note recorded with another cut is read and cut again on the next scan; a change of the setting in an open window builds the session of the vault again. A scan that does not know the size cuts new and changed notes at the default and does not cut the others again.

The floor, how similar a passage has to be to a question to count as an answer, is not a setting in the file. The index stores one floor for each vault under each model and applies it to every search by meaning. A floor given in a query takes precedence over the one the index stores, which takes precedence over the initial floor the model's plugin reports, which takes precedence over 0.30. The Search Relevance plugin, `md.numen.tools.search.relevance`, measures the floor on the vault, lets a person set one, and stores the result in the index; with the plugin disabled the index keeps applying the last floor it stored. How similar a model scores two texts about different things is a property of that model, so a changed model has a floor of its own.

A vector is stored under the model the plugin reports: its name, width, input limit and pooling, a short hash of how the plugin handles the text (the prefixes it puts in front of a document and of a query, the model file, the token limit and the normalization), and where it was made. For a connection, where it was made is the type of the connection with its address, and the name, width, input limit, pooling and prefixes are the profile that the provider of the connection gives before any text is embedded, so the index is fitted before the first batch. Changing the connection, its address or the model, or changing the model or its profile in the plugin, recreates every stored vector, and the editor asks first when a person chooses another connection or model because a service may charge for it. Changing back finds the old vectors still stored. A model that tells a question from the passage that answers it is asked for a document vector when the index is filled and for a query vector when a question is typed.

A local model the Vector Embeddings plugin does not list is added with a profile file: a JSON file in the `profiles` folder of the plugin data directory that gives the model's id, width, pooling, prefixes and files, merged over the profiles that ship with the plugin by `id`. The shape is described in the README of the plugin, and a model with no profile is refused with an error that names the folder. A vault with the plugin disabled is searched by its words alone.

## Correcting a text

The proofreading plugin, `md.numen.tools.proofreading`, corrects a transcript or a recognition when a person requests it from the file's menu, and its settings belong to the plugin. What a correction may change, and what rejects one, is in [Proofreading](proofreading.md).

The plugin reaches a model through the rows of its setting `connections`: up to 5 connections from the Connections page, each with a model and a reasoning level, tried in order (see [Rows of connections](plugins/connection-rows.md)). With no row it is off, and its page links to Connections. The plugin keeps no address and no key.

```json
{
  "md.numen.tools.proofreading": {
    "connections": [
      { "connection": "c1f0a9e20", "model": "google/gemini-2.5-flash", "reasoning": "low" },
      { "connection": "c77b01d40", "model": "qwen2.5:14b" }
    ],
    "batchSize": 40,
    "overlap": 4,
    "inFlight": 4,
    "maxEditDistance": 0.3
  }
}
```

|                   |                                                                                                                                                                                                                                                                                                                                                           |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `connections`     | the connections that proofread, in the order they are tried: at most 5 rows of a connection, a model and optionally a reasoning level. Empty turns the plugin off.                                                                                                                                                                                        |
| `batchSize`       | how many lines of a transcript one request contains. 40. A recognition is sent one page per request.                                                                                                                                                                                                                                                      |
| `overlap`         | how many lines neighbouring batches of a transcript share, so a phrase split at a batch boundary still appears whole in one of them. 4.                                                                                                                                                                                                                   |
| `inFlight`        | how many batches of a transcript one window contains; the task shows the lines done after each window. 4.                                                                                                                                                                                                                                                 |
| `maxEditDistance` | how far a correction of a recognised line may change the line's letters and still be accepted as a correction: the Levenshtein distance between what is left after spaces, punctuation, symbols, diacritics and case come off, as a share of the longer of the two. 0.30. A correction with a larger distance is dropped and that line is left as it was. |

`maxEditDistance` is 0.30 because the measured distribution has a gap there. Over 931 corrections of one book, every correction with a distance above 0.30 was damage — text taken from the next line, or one corrected word in place of a whole line — and every one below it was a correct correction. It is a setting because other books may differ; the figures are in [Performance](performance.md).

## Importing from an address

How a link's address is fetched is set by the Media Importer plugin's settings, in the object `md.numen.tools.media.importer`: `ytDlp` and `ffmpeg`, the paths of the programs (empty is the one on `PATH`); `captions`, the languages captions are taken in, best first, separated by commas; `automaticCaptions`, whether captions a machine wrote are used where a site publishes none (on); and `copyMaxSizeMb`, the maximum size in megabytes of a copy of a video (500, and 0 is no limit). What is stored, and where, is [Importing an address](importing.md).

## Which agent answers

The agent chat answers through connections, as every other plugin that needs a service. The page of the plugin `md.numen.views.agent.chat` has the setting `connections` with the operation `acp.task`: one row for each agent to try, in order. A row names a connection of an agent type: `md.numen.agents.acp/claude-code`, `md.numen.agents.acp/codex`, `md.numen.agents.acp/antigravity`, `md.numen.agents.acp/acp` (any program that speaks the Agent Client Protocol, by its `program` field) or `md.numen.agents.numen/numen`, and the model and the reasoning level the program offers. With no row the chat says that no agent is set up and offers the page. The flashcard review has a setting of its own, `agentConnections`, for the agent it asks about a card. What an agent may request from a vault is in [Agents](agents.md).

```json
{
  "md.numen.views.agent.chat": {
    "connections": [{ "connection": "c1f0a9e20", "model": "sonnet", "reasoning": "high" }]
  }
}
```

The Numen Agent chats through one connection of its own: its page has a single row that names a connection that supports chat, the model and the reasoning level. How many requests it may make to the model for one task before it is stopped is the plugin's own setting, `maxSteps` in the object `md.numen.agents.numen`, 25 unless it names another. The temperature and the maximum of tokens are its settings too. If the stream breaks in the middle of an answer, the text that arrived stays and the turn ends with an error; it does not repeat a tool call by itself.

## Connections

A connection is a place the core sends requests to for plugins, and the way in: a service you pay for, a server you run, or an agent program on this computer. You enter it once, on the Connections page, the row Connections beside System in the list of settings. A connection has an id, a type, a name, and the fields its type declares, such as an address and a key. The model and the reasoning level are not part of it. The Secrets page stays beside it and is unchanged.

The types come from the plugins. A plugin lists them under `contributes.connectionTypes` in its manifest (see [Plugins](plugins/README.md#manifest-fields)). A type is named `<plugin id>/<type id>`, for example `md.numen.connections.llm/openrouter`. The Add dialog lists the types of the enabled plugins and builds its form from the fields of the chosen type. A type that fixes its address shows none, and a type of an agent program shows the program to start, which may stay empty where the type finds its program on the path. A connection whose type no enabled plugin provides stays in the list as "Type unavailable"; it is not deleted and the plugins skip it.

The name is the text you type, trimmed, at most 64 characters, and unique ignoring case. An empty name is replaced by the title of the type for the first connection of it, and by the title and the lowest free number from 2 for the next ones, such as "OpenRouter 2".

The connections are the section `connections` of `numen.json`, a list in the order they were made, whose entries are `{ "id", "type", "name", "revision", "fields" }`. The id is `c` and 8 lowercase hexadecimal digits. `fields` holds the fields you set that are neither secret nor fixed by the type. A field the type no longer declares is kept in the file and ignored; a required field that has no value marks the connection as incomplete. Neither makes the connection invalid. A connection with a bad or repeated id, a bad or repeated name, a type without a plugin id, or a revision below 1 is ignored with a notice and the file is not changed. Only Numen writes this section: a write of the settings from a window or a plugin under `connections` is refused, and an entry edited by hand takes effect.

A secret field is kept in the key store under the entry `connection/<connection id>/<field key>`, never in `numen.json`. The entry holds a fingerprint of the type and the address together with the key, and a key whose fingerprint does not match the connection as it is now is not used and not counted as set. An edit of the address or the type in the file by hand, or a restored file, therefore cannot send a key elsewhere. Changing the address in the dialog removes the stored keys of the connection before it stores the ones you typed. An empty key has no entry. On a machine that has no keychain, such as a Linux session without a secret service, the entries are kept in `credentials/credentials.json`, in a folder next to `numen.json`; the file is readable by its owner alone (mode 0600, in a folder of mode 0700). A keychain that is there but refuses, because it is locked or the access was denied, is an error and nothing is written to the file. While the keychain is reachable a value is read from it alone, and a value stored in the keychain is removed from the file. A value is at most 2560 bytes in either place, and a longer one is refused with an error that names the limit.

Every change of a connection, including a change of a key, adds 1 to its `revision`. An update carries the revision it read and is refused with "Another window changed this connection" if the connection has another revision, so two windows cannot overwrite each other.

An address must be an http or https address with a host and no user name or password. A key is sent over http only to this computer (`localhost` or a loopback address). Link-local addresses, which include the metadata address of the clouds, the unspecified address and the metadata host names are refused. An address on a private network is accepted. A host name other than `localhost` is not resolved by the core, so a provider has to check the address it resolves to.

A plugin that needs a model asks the core for an operation, and the core takes the connections from the rows in the settings of that plugin, in the order you set them. The first row is tried first. When a connection cannot serve, because its key is rejected, a payment is required, the rate limit is reached, or the service is unreachable or too slow, the core goes to the next row. A timeout or an unreachable service is tried once more before the connection is set aside. A connection that was set aside is skipped for 60 seconds, or for the wait the service asked for, at most 10 minutes. After a rejected key or a payment required it is skipped until you edit the connection. The skip is kept in memory and is forgotten when Numen restarts. When a connection was set aside and a row after it answered, the list of tasks shows one notice for each connection and reason. When every row fails, the plugin gets an error that names each connection and its reason, and a request the service finds wrong ends the call at once without trying another row. A plugin never receives a key and cannot use a connection that is not in its own rows.

The page of a plugin that needs a model shows its rows, and the Connections page names the plugins whose rows use each connection. Deleting a connection asks first and says "Used by <plugins>. Their rows for it are removed." (see [Rows of connections](plugins/connection-rows.md)).
