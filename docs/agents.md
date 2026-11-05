# What the panel's agent can reach

A person types a question into the panel beside their notes, and the application starts an agent to answer it. It chooses the program, the model, the tools, the working directory and the environment. This page is what that agent is given and what is restricted.

## What the agent is given

One built-in tool: a web search. The set is named in full on the command line, so every other built-in is absent — no shell, no file reader, no file writer, no subagent, and no web fetch. A fetch goes to an address given in the text, and a note may have been written by anybody, so the vault's content would decide where the fetch goes.

One tools server: this vault's, at the address the application is already serving on, sent with the bearer token. No other server's configuration is read.

This vault's tools are approved ahead of the run, as `mcp__numen__*`, and no permission prompt is shown for them during the run. The panel shows no prompts.

The working directory is the vault.

## What is restricted

Hooks, skills, standing instructions, plugins and custom agents are blocked by default: the agent is given a list of the configuration sources it may read, and by default the list is empty.

The setting `readsHooksAndSkills` of the plugin `md.numen.agents.claude` enables the person's own hooks and skills, and only those. What a vault itself contains is blocked in either mode: a vault is a folder that comes from elsewhere — synced, cloned, shared, restored — and a configuration file inside one can list commands for this machine to run.

The sources are listed one by one. This vault's own tools are passed on a command line and are treated as a customisation like any other.

## The environment

The agent inherits this machine's environment, except nine named variables that describe a Claude Code session somebody else is running:

```
CLAUDECODE                    CLAUDE_CODE_MESSAGING_SOCKET
CLAUDE_CODE_SESSION_ID        CLAUDE_CODE_MESSAGING_TOKEN
CLAUDE_CODE_CHILD_SESSION     CLAUDE_PID
CLAUDE_CODE_ENTRYPOINT        CLAUDE_EFFORT
CLAUDE_CODE_EXECPATH
```

Everything else is passed on. It is a denylist: the inherited environment minus nine variables. The variables that set how to reach a model are passed on, as intended. So are the variables that set which server sees the notes, which configuration root is read, which proxy the traffic goes through, and which program the name `claude` resolves to.

## What the agent is told

Three things, appended to the model's default instructions:

- It is answering inside the application the person keeps these notes in, beside the note they are looking at.
- A note is called by its title. A path, a file name and an extension are how the tools address a note, and are never shown.
- The answer is read in a narrow panel.

Where the person has a note open, its path is named, and a task saying "this note" means that one.

The tools server also gives every agent that connects to it its own instructions: how to name a note and a passage as links, which tool searches and reads what (`note_search` the notes, `library_search` and `library_read` everything the vault holds), and that a passage is a window cut to a size and is read on before a source is said not to say something.

## What an answer links to

A note an answer mentions is named as `[[Harmonic oscillator]]` — the same brackets a note is written with, resolved the same way, and pressing one opens the note in a tab beside the tab the person is looking at. A name that matches no note is drawn as an unresolved link, and pressing it opens nothing. `[[note://<identifier>]]` identifies one note where a title is shared. See [Links](links.md).

A passage of a book is named as `[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)`: a document's path, percent-encoded, and the text range the search returned. Pressing one opens the document there, with the other passages of that answer highlighted.

The window a person runs their cards in opens neither: an answer there states in words where each part came from, because a link cannot be opened there.

## The tools

Three plugins register the tools an agent can call, each for its own subject, and `agents-mcp-server` serves them. The server column says where each is offered: `full`, `reading`, or both.

`agents-tools-vault`, the notes and files of the vault:

| Tool             | What it does                                                                                                                                                                                                                 | Servers |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `note_read`      | the content, prose, title, frontmatter and fingerprint of one note, by its path                                                                                                                                              | both    |
| `note_write`     | write or create one note; with the fingerprint `note_read` gave, a write over somebody else's edit is refused                                                                                                                | full    |
| `note_headings`  | the headings of one note: level, text, line and byte offset                                                                                                                                                                  | both    |
| `note_search`    | the notes whose title, path or text match a query, at most 100, 20 where the call names no limit                                                                                                                             | both    |
| `note_list`      | the paths and titles of the notes of the vault or of one folder, ordered by path; at most 500, 100 where the call names no limit, paged with `offset` and `next_offset`, with the total count                                | both    |
| `note_titles`    | the titles of up to 50 notes by their paths, and the paths that name no note                                                                                                                                                 | both    |
| `note_resolve`   | the paths of every note with one file name, as a wikilink names it; more than one path means the name is ambiguous                                                                                                           | both    |
| `library_search` | passages of everything the index holds (notes, books, PDFs, scans, recordings) by the words and by the meaning, with the file, the title, the place, the text and the address of each; narrowed by `kinds` and `path_prefix` | both    |
| `library_read`   | the text of one file from its text version, by page or by byte range, with the places, the size and where to read on; an error naming what makes a text version where there is none                                          | both    |
| `link_list`      | the links written in one note and the notes that link to it                                                                                                                                                                  | both    |
| `vault_get`      | the vault's name, root path and number of notes                                                                                                                                                                              | both    |
| `vault_list`     | every vault of the installation: identifier, name and folder, with the vault the call works on marked `showing`                                                                                                              | both    |
| `vault_problems` | the files of the vault that could not be parsed, with the path and a description of each problem; only the `parse` check is served                                                                                           | both    |
| `file_read`      | a byte range of a text file in the vault, by its path from the vault root; a binary is refused with an error that names `library_read`                                                                                       | both    |
| `file_create`    | a new file in the vault, holding the content given                                                                                                                                                                           | full    |

`agents-tools-window`, the window the person is looking at:

| Tool               | What it does                                                                        | Servers |
| ------------------ | ----------------------------------------------------------------------------------- | ------- |
| `window_tabs_get`  | the tabs open in the window — title, path, kind, position — and which one is active | both    |
| `window_focus_tab` | make a tab active, named by its id, its path or its title                           | both    |
| `edit_preview`     | propose a change to a note as a diff in the editor, leaving the file as it is       | both    |

`views-flashcards-studio`, the decks and card models of the vault:

| Tool                                                             | What it does                                                                    | Servers |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------- |
| `card_current`                                                   | the card shown to the person in the window they run their cards in              | both    |
| `card_model_list`                                                | the card models of the vault, at most 50, and the fields each declares          | both    |
| `card_model_read`                                                | the fields and templates of one card model                                      | both    |
| `card_read`                                                      | the cards of one deck in deck order, at most 50 a call, or one card by its id   | both    |
| `card_deck_create`                                               | a deck holding no cards                                                         | full    |
| `card_model_create`                                              | a card model: the fields a card requires, and its templates                     | full    |
| `card_model_edit`                                                | change a card model's fields and templates in place                             | full    |
| `card_field_rename`                                              | rename one of a card model's fields everywhere it is used                       | full    |
| `card_add`                                                       | write one card at the end of a deck or of one of its sections                   | full    |
| `card_edit`                                                      | write values into one card, leaving the fields the call does not name unchanged | full    |
| `card_value_remove`                                              | remove one field from one card                                                  | full    |
| `card_remove`                                                    | remove one card from a deck                                                     | full    |
| `card_section_add`, `card_section_rename`, `card_section_remove` | write, rename and remove the sections a deck is divided into                    | full    |

A card is named by its card id and never by the place it is in, so the cards stay addressable when a deck is reordered during a run. Every write to a deck takes the fingerprint `card_read` returned, and is applied only to the deck that fingerprint identifies.

`file_read` is how a text file that is not a note is read — a transcript somebody typed, an export, whatever text they put in the folder — and how a file too long to return whole is read one range at a time. A binary file, such as a PDF, is read with `library_read`.

### How a tool is named

A tool is named for the subject it acts on, then what it does: `note_read`, `card_add`, `window_focus_tab`. `edit_preview` has no subject prefix.

## The servers

A tool declares the servers it is offered on: the full server, which gives an agent access to the whole vault, and the reading server, whose tools all read. A tool that writes is on the full server alone; a tool that reads is on both. `card_current` returns the card shown to the person in the window they run their cards in, and an empty deck between cards.

The tools are registered by the plugins that own their subjects — `agents-tools-vault`, `agents-tools-window`, `views-flashcards-studio` — and `agents-mcp-server` gathers the tools of every plugin and of the host and serves them to an agent.

## An agent a person runs themselves

The tools are served to an agent somebody has configured in their own terminal by the plugin `md.numen.agents.mcp.server`, which listens on `127.0.0.1` at the plugin's setting `port` (3333 by default). The application itself opens no port and writes no token file.

## Host agent auto-discovery

At startup and on demand, the application inspects the host environment for installed CLI agents:

- Scans system `PATH` and standard installation directories (`~/.local/bin`, npm global prefix, Homebrew, Nix store).
- Probes binary availability, version, and execution readiness.
- Discovered agents populate available options across the interface with their runtime status.

## Multi-agent selector island

The chat panel provides a floating selector island above the message composer:

- Allows quick switching between discovered host agents and configured models per session.
- Isolates MCP server connections and environment configuration per selected agent.
- Manages sub-process lifetimes, terminating background child processes concurrently when sessions close.

## Settings

`app.agent.use` sets which program answers, and empty sets none. When unset, the program the plugins mark as the default answers. Each program a plugin lists has a section under its name, `app.agent.claude.*` for Claude Code: `command` starts it, and `model` is which of its models answers. What a plugin reads for itself, such as `readsHooksAndSkills` for Claude Code, is the plugin's own setting. A section is kept whether it is the one in use or not. The programs and their models are listed by the plugins that run them, and the window shows what they list. See [Settings](settings.md).

Where `command` is empty, the path is searched first, then the folders the command line's installers write to.

## What is not covered

**The bearer token is on the command line.** It is written into the server configuration the agent is started with, so any local process on this machine that can read `/proc` can read it, and it grants what the tools grant. The token file is 0600 and this removes that protection.

**An agent whose tools did not load answers anyway.** The stream reports that the server was unreachable, the run continues, and the reason is reported after the answer. The posture intended is fail-closed and what is implemented is answer-then-explain.

**A file the agent writes in the vault is read by the person's own agent.** The panel's agent reads the vault's instructions in neither mode. A note it writes stays in the folder, though, and the person's own agent, started in that folder, reads what is there. Where a note may be written is limited to places a note may live and this application's own folders, so another tool's dot-folder is blocked; the vault's root is not, and a file named for what another tool reads can be written there.

**Two hand-maintained lists track somebody else's program.** The nine filtered environment variables and the named built-in tools are both lists, and the program they filter can change in any release without notice. Each release must be checked, and nothing checks it automatically.
