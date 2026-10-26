---
title: numen-cli
description: The same vault and the same index, reached from a terminal and from a script.
---

`numen-cli` is the second way into a vault: no window, one command at a time, written for scripts and for the jobs you would rather leave running. It reads the same settings, writes the same index and follows the same rules as the window — a vault scanned here is a vault the window opens already searchable.

It comes with the source rather than with the installers.

A vault is named by whichever you have to hand — its name, its path, or its identity:

```sh
numen-cli scan Research
numen-cli scan ~/notes/research
numen-cli search Research "entropy"
```

## What it takes

<!-- BEGIN AUTOGEN -->
| | |
| --- | --- |
| `numen-cli vault add <path> [--name <name>]` | give a folder an identity and remember it |
| `numen-cli vault list` | show the vaults this installation knows |
| `numen-cli vault rename <vault> <new name>` | call a vault something else |
| `numen-cli vault remove <vault>` | take a vault off the list, leaving its folder |
| `numen-cli vault erase <vault> [--yes]` | remove it, and put its folder in the trash |
| `numen-cli vault open <vault>` | the vault the next window opens |
| `numen-cli scan <vault> [--rebuild-index]` | bring the index up to date with a vault --rebuild-index reads every file again, forgets the vectors of every model but the one in use, and gives their space back |
| `numen-cli search <vault> <query>` | full-text search within one vault |
| `numen-cli links <vault> <note>` | what a note points at, and what points at it |
| `numen-cli problems <vault> [<check>...]` | what the vault holds that was not guessed at |

### Over every command

| | |
| --- | --- |
| `--index <path>` | where the index lives (default: platform cache directory) |
| `--registry <path>` | where the vault list lives (default: platform config directory) |
| `--service-dir <name>` | the folder a vault keeps its identity in (default: .numen) |
<!-- END AUTOGEN -->

## The commands, one at a time

**`vault add`** gives a folder an identity and puts it on the list, the same list the window shows. `--name` calls it something other than the folder's own name; a name another vault has gets a number appended.

**`vault erase`** asks before it moves a folder to the trash. `--yes` answers for you, which is what a script needs. `vault remove` never asks: the folder stays where it is.

**`vault open`** does not open anything here — it says which vault the *next window* will open.

**`scan`** brings the index up to date with what is on disk: what changed is read again, what is gone is dropped. `--rebuild-index` reads every file whatever the index remembers, which is the answer when search is returning something stale. Vectors are made here too, and unlike the window, which does it in the background, the command waits for them.

**`search`** is the words half of what the window does: names and text, in one vault.

**`links`** prints what a note points at and what points at it, which is the plex written out.

**`problems`** prints what the vault holds that numen would not guess at. Name the checks you want, or none for all four:

| | |
| --- | --- |
| `parse` | a link with no target, a link with no role, a role nobody decided on. |
| `frontmatter` | a block between the `---` lines that is not YAML. |
| `ambiguous` | a link that reaches more than one note. |
| `dangling` | a link that reaches nothing. |

## Commands of the plugins

Every other command comes from an enabled plugin, which declares it; `numen-cli help` lists them under "commands of the plugins", each with the arguments it takes. A command over a file takes the file after the vault, and an argument is a flag. The plugins run without a window, and a command waits until its plugin is done:

| | |
| --- | --- |
| `numen-cli transcribe <vault> <file>` | transcribe a recording (`engines-transcription`) |
| `numen-cli recognise <vault> <file>` | recognise the text of a scan, page by page (`engines-ocr`) |
| `numen-cli proofread <vault> <file>` | put the live text of a file right with the model named in [the settings](/reader/#correcting-what-ocr-read) (`tools-proofreading`) |
| `numen-cli import <vault> --address <url>` | keep an address as a link and fetch its text (`tools-media-importer`) |

What a command makes is a text version of the file. The next `scan` reads it into the index, as the window does, and `search` finds it. A plugin turned off takes its commands with it.

## Where it puts things

The same places the window uses: the index and the vault list where the platform keeps them, and `.numen/` inside each vault. `--index` and `--registry` point either somewhere else — a copy to try something on, a check that touches nothing you use.

`--service-dir` renames the folder a vault keeps its identity in. It has to match what the window is using, or the two will disagree about what your vault contains.
