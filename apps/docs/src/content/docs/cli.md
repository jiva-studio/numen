---
title: numen-cli
description: The same vault and the same index, used from a terminal and from a script.
---

`numen-cli` is a second way to use a vault: no window, one command at a time, intended for scripts and for jobs that run unattended. It reads the same settings, writes the same index and follows the same rules as the window — a vault scanned here is a vault the window opens already searchable.

It comes with the source rather than with the installers.

You can refer to a vault by its name, its path or its identity:

```sh
numen-cli scan Research
numen-cli scan ~/notes/research
numen-cli search Research "entropy"
```

## What it takes

<!-- BEGIN AUTOGEN -->
| | |
| --- | --- |
| `numen-cli vault add <path> [--name <name>]` | add a folder to the vault list |
| `numen-cli vault list` | show the vaults this installation knows |
| `numen-cli vault rename <vault> <new name>` | change the name of a vault |
| `numen-cli vault remove <vault>` | take a vault off the list, leaving its folder |
| `numen-cli vault erase <vault> [--yes]` | remove it and move its folder to the trash |
| `numen-cli vault open <vault>` | set the vault the next window opens |
| `numen-cli scan <vault> [--rebuild-index]` | update the index of a vault --rebuild-index reads every file again and deletes the vectors of every model except the one in use |
| `numen-cli search <vault> <query>` | full-text search within one vault |
| `numen-cli links <vault> <note>` | what a note points at, and what points at it |
| `numen-cli problems <vault> [<check>...]` | list notes with broken links, invalid frontmatter or ambiguous links |

### Over every command

| | |
| --- | --- |
| `--index <path>` | where the index lives (default: platform cache directory) |
| `--registry <path>` | where the vault list lives (default: platform config directory) |
| `--service-dir <name>` | the folder a vault keeps its identity in (default: .numen) |
<!-- END AUTOGEN -->

## The commands, one at a time

**`vault add`** gives a folder an identity and puts it on the list, the same list the window shows. `--name` calls it something other than the folder's own name; a name another vault has gets a number appended.

**`vault erase`** asks before it moves a folder to the trash. `--yes` confirms for you, which a script needs. `vault remove` never asks for confirmation: the folder stays where it is.

**`vault open`** does not open anything here. It sets which vault the *next window* opens.

**`scan`** brings the index up to date with what is on disk: changed files are read again, and deleted files are removed from the index. `--rebuild-index` reads every file regardless of what the index has stored. Use it when search returns stale results. Vectors are made here too, and unlike the window, which does it in the background, the command waits for them.

**`search`** is the text-search part of what the window does: names and text, in one vault.

**`links`** prints what a note points at and what points at it, which is the plex as text.

**`problems`** prints what numen cannot interpret in the vault. Name the checks you want, or none to run all four:

| | |
| --- | --- |
| `parse` | a link with no target, a link with no role, a role nobody decided on. |
| `frontmatter` | a block between the `---` lines that is not YAML. |
| `ambiguous` | a link that points to more than one note. |
| `dangling` | a link that points to no note. |

## Commands of the plugins

Every other command comes from an enabled plugin, which declares it; `numen-cli help` lists them under "commands of the plugins", each with the arguments it takes. A command over a file takes the file after the vault, and an argument is a flag. The plugins run without a window, and a command waits until its plugin is done:

| | |
| --- | --- |
| `numen-cli proofreading.create <vault> <file>` | correct the live text of a file with the model named in [the settings](/reader/#correcting-what-ocr-read) (`tools-proofreading`) |
| `numen-cli link.import <vault> --address <url>` | keep an address as a link note (`tools-media-importer`) |
| `numen-cli link.downloadMedia <vault> <note>` | download the video of a link note (`tools-media-importer`) |
| `numen-cli link.deleteMedia <vault> <note>` | delete the downloaded media of a link note (`tools-media-importer`) |

Some commands belong to the application and are run by the plugin that handles them; `numen-cli help` lists them under "shared commands", each with the plugins that handle it. Where one plugin handles the file, it runs; where several do, `--plugin <id>` picks one:

| | |
| --- | --- |
| `numen-cli app:file.createText <vault> <file>` | create the text of a file: the text layer of a PDF (`md.numen.readers.pdf`), the text of an EPUB (`md.numen.readers.epub`), the text of a scan, recognized page by page (`md.numen.engines.ocr`), the transcript of a recording (`md.numen.engines.transcription`), the captions or the article of a link note (`md.numen.tools.media.importer`) |
| `numen-cli app:file.createText <vault> <file> --plugin md.numen.engines.ocr` | recognize the text of a PDF that also has a text layer |

What a command creates is a text version of the file. The next `scan` reads it into the index, as the window does, and `search` finds it. A disabled plugin has no commands.

## Where it puts things

The same places the window uses: the index and the vault list where the platform keeps them, and `.numen/` inside each vault. `--index` and `--registry` set either to another location, for example a copy to experiment on, or a check that does not touch any data you use.

`--service-dir` renames the folder a vault keeps its identity in. It has to match what the window is using, or the two will disagree about what your vault contains.
