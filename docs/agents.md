# What the panel's agent can reach

A person types a question into the panel beside their notes, and the application starts an agent to answer it. It chooses the program, the model, the tools, the working directory and the environment. This page is what that agent is given and what it is refused; the decisions behind it are [An agent reaches the vault through tools](adr/0021-an-agent-reaches-the-vault-through-tools.md).

## What the agent is given

Two built-in tools: a web search and a web fetch. The set is named in full on the command line, so every other built-in is absent — no shell, no file reader, no file writer, no subagent.

One tools server: this vault's, at the address the application is already serving on, presented with the bearer token. No other server's configuration is read.

This vault's tools are approved ahead of the run, as `mcp__numen__*`, and nothing is asked about them while the run is on. The panel prompts for nothing.

The working directory is the vault.

## What is refused

Hooks, skills, standing instructions, plugins and custom agents are refused by default: the sources of configuration the agent may read are named, and by default the list is empty.

`agent.claude.reads_hooks_and_skills` gives back the person's own, and only theirs. What a vault itself carries is refused in either mode: a vault is a folder that arrives from elsewhere — synced, cloned, shared, restored — and a configuration file inside one names commands for this machine to run.

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

Five families and a reader of files, served over the vault's own endpoint.

| Family | What it is for |
| --- | --- |
| `note_*` | search the vault, look notes up by path and by the name a link writes, read and rewrite their prose, edit a stretch, rename, move, remove, and put one in front of the person |
| `file_read` | read a run of any file the vault holds, by its path from the vault folder |
| `link_*` | add, change, remove and list the links a note carries |
| `card_*` | list the stencils a vault holds, read a deck and the cards in it, and make, change and remove one card at a time |
| `source_*` | list the documents a vault holds, read a run of one's text, ask for a scanned one to be read, and show the person a passage |
| `vault_*` | show the vault and what a scan could not act on, and list, add, rename, forget and open vaults |

A call that carries names takes as many as are wanted — at most fifty for a lookup, at most ten for reading prose. A call that carries the text of a document takes one: creating a note, writing one and editing one are each a call of their own, and each is filed as it is finished.

A deck is as long as somebody made it, so `card_read` answers with at most fifty cards at a time and is told where to start. An agent that knows which card it wants names its mark and is answered with that one, and one that wants a field names it and is answered that field alone. A card is named by a mark and never by the place it stands in, so a deck reordered under an agent leaves what it holds addressable.

A tool that writes returns only once the index is level again. An agent that creates a note and searches for it in the next breath finds it.

`file_read` is the path a person names when the file behind it is neither a note nor a document the vault has read — a transcript somebody typed, an export, whatever they put in the folder — and it is how a file too long to answer with is read a run at a time. It takes a path from the vault root and refuses every other, including one that reaches outside through a link. What the vault passes over it passes over too: the application's own folder, and every name the vault's ignore rules match. The tools write notes, so a file of another kind is read here and not written.

## The reviewer's surface

The window a person runs their cards in serves a surface of its own. It reads the whole vault — `note_search`, `note_titles`, `note_read`, `note_neighbourhood`, `link_list`, `source_list`, `source_read`, `card_stencil_list`, `card_read` and `vault_get` — and writes cards alone: `card_add`, `card_edit`, `card_value_remove`, `card_remove`, `card_section_add`, `card_section_rename` and `card_section_remove`. Nothing else is on it. A deck and a stencil are what a vault is arranged into, and nothing there makes one; no note, link or document is written there either. The decisions behind it are [Review is an application of its own](adr/0027-review-is-an-application-of-its-own.md).

The reading half of it is a surface in its own right, with no writer on it at all.

The tools themselves are the same tools: each family registers its reading half and its writing half separately, and the full surface is both halves.

That window opens the index for reading alone, and nothing embeds behind it, so a search there answers by the words in the vault and not by what they mean. It serves its port on a loopback address the machine picks, with a token that lives in memory, and writes no `agents.json`: the address file names one window's vault, and a second window rewriting it would point a person's own agent at whichever started last.

The agent is told which vault it works when it is started, so it is started when a sitting opens and stopped when a sitting opens on another vault or the window closes.

## An agent a person runs themselves

The tools are served on a port, and the panel's agent is one caller of it. An agent somebody has configured in their own terminal is another, and `agent.serve_tools` is what serves them to it: on, the endpoint answers whether or not the panel has an agent, and where to reach it and what to present are written to `agents.json` beside this installation's own state, at mode `0600`. The file is removed when the window goes, so a live-looking token never outlives the port it was for.

It is off. An installation that names no agent and asks for no tools opens no port and writes no token file — a person who never asked for an agent is running a window and nothing besides.

## Settings

`agent.use` names which agent answers, and empty names none. `agent.serve_tools` puts the tools on a port for an agent a person runs themselves, and is off. `agent.claude.*` is how Claude Code is run: `command` starts it, `model` is which of its models answers, `max_steps` is how many times it may go to the model before it is stopped, and `reads_hooks_and_skills` is what this machine holds for it. A section is kept whether it is the one in use or not. See [Settings](settings.md).

Where `command` is empty, the path is asked first, then the folders the command line's installers write to.

## What is not covered

**The bearer token is on the command line.** It is written into the server configuration the agent is started with, so any local process on this machine that can read `/proc` can read it, and it grants what the tools grant. The token file is 0600 and this undoes that.

**An agent whose tools did not arrive answers anyway.** The stream says the server was unreachable, the run continues, and the reason is reported after the answer. The posture intended is fail-closed and what is implemented is answer-then-explain.

**A file the agent leaves in the vault is read by the person's own terminal.** The panel's agent reads the vault's instructions in neither mode. A note it writes stays in the folder, though, and the person's own agent, started in that folder, reads what is there. Where a note may be written is bounded to places a note may live and this application's own folders, so another tool's dot-folder is refused; the vault's root is not, and a file named for what another tool reads can land in it.

**Two hand-maintained lists track somebody else's program.** The nine refused variables and the two named built-in tools are both lists, and the program they filter changes without asking. Each release is a thing to check, and nothing checks it automatically.
