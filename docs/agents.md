# What the panel's agent can reach

A person types a question into the panel beside their notes, and the application starts an agent to answer it. It chooses the program, the model, the tools, the working directory and the environment. This page is what that agent is given and what is restricted.

## What the agent is given

One built-in tool: a web search. The set is named in full on the command line, so every other built-in is absent — no shell, no file reader, no file writer, no subagent, and no web fetch. A fetch goes to an address the text names, and a note may have been written by anybody, so the address would be the vault's to choose.

One tools server: this vault's, at the address the application is already serving on, presented with the bearer token. No other server's configuration is read.

This vault's tools are approved ahead of the run, as `mcp__numen__*`, and nothing is asked about them while the run is on. The panel prompts for nothing.

The working directory is the vault.

## What is restricted

Hooks, skills, standing instructions, plugins and custom agents are blocked by default: the sources of configuration the agent may read are named, and by default the list is empty.

`agent.claude.reads_hooks_and_skills` gives back the person's own, and only theirs. What a vault itself carries is blocked in either mode: a vault is a folder that arrives from elsewhere — synced, cloned, shared, restored — and a configuration file inside one names commands for this machine to run.

The sources are named one by one. This vault's own tools arrive on a command line and are read as a customisation like any other.

## The environment

The agent inherits this machine's environment, less nine named variables that describe a Claude Code session somebody else is running:

```
CLAUDECODE                    CLAUDE_CODE_MESSAGING_SOCKET
CLAUDE_CODE_SESSION_ID        CLAUDE_CODE_MESSAGING_TOKEN
CLAUDE_CODE_CHILD_SESSION     CLAUDE_PID
CLAUDE_CODE_ENTRYPOINT        CLAUDE_EFFORT
CLAUDE_CODE_EXECPATH
```

Everything else passes. It is a denylist: an inherited environment with nine holes plugged. What says how to reach a model passes, which is the point, and so does what says which server sees the notes, which configuration root is read, which proxy the traffic goes through, and which program the name `claude` resolves to.

## What the agent is told

Three things, appended to whatever the model already carries:

- It is answering inside the application the person keeps these notes in, beside the note they are looking at.
- A note is called by its title. A path, a file name and an extension are how the tools address a note, and are never shown.
- The answer is read in a narrow panel.

Where the person has a note in front of them, its path is named, and a task saying "this note" means that one.

## What an answer links to

A note an answer speaks about is named as `[[Harmonic oscillator]]` — the same brackets a note is written with, resolved the same way, and pressing one opens the note in a tab beside what the person is looking at. A name no note answers to is drawn as reaching nothing, and pressing it opens nothing. `[[note://<identifier>]]` names one note where a title is shared. See [Links](links.md).

A passage of a book is named as `[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)`: a document's path, percent-encoded, and the run of its text the search gave. Pressing one opens the document there, with the other passages of that answer lit beside it.

The window a person runs their cards in opens neither: an answer there says where each part came from in words, and a link is not a place a person can go.

## The tools

Three plugins register the tools an agent reaches, each over the subject it owns, and `agents-mcp-server` serves them. The server column says where each is offered: `full`, `reading`, or both.

`agents-tools-vault`, the notes and files of the vault:

| Tool                              | What it does                                                                                                  | Servers |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------- | ------- |
| `note_read`                       | the content, prose, title, frontmatter and fingerprint of one note, by its path                               | both    |
| `note_write`                      | write or create one note; with the fingerprint `note_read` gave, a write over somebody else's edit is refused | full    |
| `note_headings`                   | the headings of one note: level, text, line and byte offset                                                   | both    |
| `note_search`, `search_query`     | the notes whose title, path or text answer a query, at most 100, 20 where the call names no limit             | both    |
| `note_neighbourhood`, `link_list` | the links one note writes and the notes that link to it                                                       | both    |
| `vault_info`, `vault_get`         | the vault's name, root path and number of notes                                                               | both    |
| `file_read`                       | a byte range of any file in the vault, by its path from the vault root                                        | both    |
| `file_create`                     | a new file in the vault, holding the content given                                                            | full    |

`agents-tools-window`, the window the person is looking at:

| Tool               | What it does                                                                          | Servers |
| ------------------ | ------------------------------------------------------------------------------------- | ------- |
| `window_tabs_get`  | the tabs open in the window — title, path, kind, position — and which one is in front | both    |
| `window_focus_tab` | put a tab in front, named by its id, its path or its title                            | both    |
| `edit_preview`     | propose a change to a note as a diff in the editor, leaving the file as it is         | both    |

`views-flashcards-studio`, the decks and stencils of the vault:

| Tool                                                             | What it does                                                                              | Servers |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ------- |
| `card_current`                                                   | the card in front of the person in the window they run their cards in                     | both    |
| `card_stencil_list`                                              | the stencils of the vault, at most 50, and the fields each declares                       | both    |
| `card_stencil_read`                                              | the fields and faces of one stencil                                                       | both    |
| `card_read`                                                      | the cards of one deck in the order they stand, at most 50 a call, or one card by its mark | both    |
| `card_deck_create`                                               | a deck holding no cards                                                                   | full    |
| `card_stencil_create`                                            | a stencil: the fields a card is asked for, and its faces                                  | full    |
| `card_stencil_edit`                                              | change a stencil's fields and faces in place                                              | full    |
| `card_field_rename`                                              | give one of a stencil's fields another name, everywhere it is written                     | full    |
| `card_add`                                                       | write one card at the end of a deck or of one of its sections                             | full    |
| `card_edit`                                                      | write values into one card, leaving the fields the call does not name                     | full    |
| `card_value_remove`                                              | take one field off one card                                                               | full    |
| `card_remove`                                                    | take one card out of a deck                                                               | full    |
| `card_section_add`, `card_section_rename`, `card_section_remove` | write, rename and remove the sections a deck is divided into                              | full    |

A card is named by its mark and never by the place it stands in, so a deck reordered under an agent leaves what it holds addressable. Every write to a deck takes the fingerprint `card_read` gave, and lands only on the deck that fingerprint names.

`file_read` is how a file that is not a note is read — a transcript somebody typed, an export, whatever they put in the folder — and how a file too long to answer with is read a range at a time.

### How a tool is named

A tool is named for the subject it acts on, then what it does: `note_read`, `card_add`, `window_focus_tab`. Three tools carry a second name that answers the same call: `search_query` is `note_search`, `link_list` is `note_neighbourhood`, and `vault_get` is `vault_info`. `edit_preview` stands under no subject.

## The servers

A tool declares the servers it is offered on: the full server, which an agent works the whole vault through, and the reading server, whose every tool reads. A tool that writes is on the full server alone; a tool that reads is on both. `card_current` answers with the card in front of the person in the window they run their cards in, and with an empty deck between cards.

The tools are registered by the plugins that own their subjects — `agents-tools-vault`, `agents-tools-window`, `views-flashcards-studio` — and `agents-mcp-server` gathers the tools of every plugin and of the host and serves them to an agent.

## An agent a person runs themselves

The tools are served on a port, and the panel's agent is one caller of it. An agent somebody has configured in their own terminal is another, and `agent.serve_tools` is what serves them to it: on, the endpoint answers whether or not the panel has an agent, and where to reach it and what to present are written to `agents.json` beside this installation's own state, at mode `0600`. The file is removed when the window goes, so a live-looking token never outlives the port it was for.

It is off. An installation that names no agent and asks for no tools opens no port and writes no token file — a person who never asked for an agent is running a window and nothing besides.

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

`agent.use` names which program answers, and empty names none. Left out, the program the plugins mark as the default answers. `agent.serve_tools` puts the tools on a port for an agent a person runs themselves, and is off. Each program a plugin lists has a section under its name, `agent.claude.*` for Claude Code: `command` starts it, `model` is which of its models answers, `max_steps` is how many times it may go to the model before it is stopped, and `reads_hooks_and_skills` is what this machine holds for it. A section is kept whether it is the one in use or not. The programs and their models are listed by the plugins that run them, and the window offers what they list. See [Settings](settings.md).

Where `command` is empty, the path is asked first, then the folders the command line's installers write to.

## What is not covered

**The bearer token is on the command line.** It is written into the server configuration the agent is started with, so any local process on this machine that can read `/proc` can read it, and it grants what the tools grant. The token file is 0600 and this undoes that.

**An agent whose tools did not arrive answers anyway.** The stream says the server was unreachable, the run continues, and the reason is reported after the answer. The posture intended is fail-closed and what is implemented is answer-then-explain.

**A file the agent leaves in the vault is read by the person's own terminal.** The panel's agent reads the vault's instructions in neither mode. A note it writes stays in the folder, though, and the person's own agent, started in that folder, reads what is there. Where a note may be written is bounded to places a note may live and this application's own folders, so another tool's dot-folder is blocked; the vault's root is not, and a file named for what another tool reads can land in it.

**Two hand-maintained lists track somebody else's program.** The nine filtered environment variables and the named built-in tools are both lists, and the program they filter changes without asking. Each release is a thing to check, and nothing checks it automatically.
