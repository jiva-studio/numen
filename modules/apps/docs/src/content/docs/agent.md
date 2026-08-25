---
title: The agent
description: An assistant that has read your vault, answers beside the note you are on, and can write in it when you ask.
---

The panel along the edge of the window is an agent that works inside your vault. It searches
your notes, reads them, writes them, and connects them — the same operations you have, reached
by asking in words.

## Asking

Type into the panel and send. **Ask the agent about this note** in the commands starts a
conversation about the note you are on, and <kbd>Ctrl</kbd>/<kbd>⌘</kbd> <kbd>⇧</kbd>
<kbd>A</kbd> opens a fresh one.

It knows which note you are looking at, so *this note*, *file this under Thermodynamics* and
*what did I write about entropy* all mean what you would expect them to mean.

Each thing it does is shown as it does it — the note it read, the link it added — and **Stop**
ends the run wherever it has got to.

## What it can do

| | |
| --- | --- |
| Notes | search the vault, read notes, write and edit them, rename, move, remove, and put one in front of you. |
| Links | add, change, remove and list the links a note carries. |
| Documents | list the books a vault holds, read a stretch of one, ask for a scanned one to be read, show you a passage. |
| Vaults | tell you what a scan could not make sense of, and list, add, rename, forget and open vaults. |

It also has web search and web fetch, for the questions your notes do not answer.

Anything it writes lands in your files, so it shows up in the plex and in search at once — an
agent that creates a note and then searches for it finds it.

## What it cannot do

**It has no shell, no file reader and no file writer.** It reaches your vault through the
operations above and through nothing else. It cannot run a command on your machine, and it
cannot touch a file outside the places a note may live.

**It reads no instructions carried by the vault.** A vault is a folder that arrives from
somewhere — synced, cloned, shared, restored — and a configuration file inside one names
commands for your machine to run. Those are refused, always, whatever the settings say.

**It cannot erase a vault from your disk.** Taking a folder away is asked for in front of you.

By default it also ignores what is configured for the assistant on this machine — standing
instructions, hooks, skills, plugins. If you want your own configuration used, turn on
`agent.claude.reads_hooks_and_skills` in [the settings](/settings/); a vault's own files stay
refused either way.

## Setting it up

numen answers with [Claude Code](https://claude.com/claude-code), which is a program you install
on your machine and sign in to yourself. Where it is installed in the usual place, numen finds
it and there is nothing to configure.

```json
{
  "agent": {
    "use": "claude",
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
| `use` | which agent answers. Empty means none, and the panel says so. |
| `command` | how to start it. Empty looks on the path, then where its installers put it. Worth naming on a machine carrying several installations. |
| `model` | `opus`, `sonnet`, or a full model name. Empty takes whatever that installation answers with. |
| `max_steps` | how many turns it may take before it is stopped. |

## Where it runs, and what it costs

The agent runs on your machine and answers through your own account with whoever provides the
model. numen has no server, and there is nothing for your notes to be uploaded to: what leaves
the machine is what the model is asked, by the program you installed and signed in to.

[Searching by meaning](/finding/) is separate from this, and by default it runs a small model
locally with no account at all.
