---
title: When something is wrong
description: What numen says when it will not do something, what it means, and what to do about it.
---

numen says what it cannot do, where you are, in the words below. Nothing is written into a file
it could not read, and nothing is guessed at on your behalf. This page is that vocabulary.

## The mark on a tab

A note's tab carries one word beside its title when it is not simply saved.

| | |
| --- | --- |
| `unsaved` | what you see has not reached the file yet. It will in a moment; <kbd>Ctrl</kbd>/<kbd>⌘</kbd> <kbd>S</kbd> makes it now. |
| `overtaken` | the file changed underneath the tab, so saving stopped. Answer with **keep mine** or **take the file's**, and it starts again. Nothing you typed is lost while it waits. |
| `gone` | the file is no longer at the name this tab stands at. What is on screen is still yours: **keep mine** writes the note back into existence there. |
| `stuck` | this note cannot be read or written, and the tab says which of the reasons below it is. |

*These notes stopped saving because their files changed. The window waits.* — the same thing, said
about more than one tab at once, usually after a sync client has been through the vault.

## About a note

| | |
| --- | --- |
| *that note is not in the vault* | the name does not reach a note. It may have been moved or removed outside numen. |
| *that file is not a note* | its extension is not one this vault reads as notes. |
| *that file is not text* | not valid UTF-8, so it is not opened here. |
| *that note is longer than this writes* | over a megabyte. Split it, or edit it elsewhere. |
| *the frontmatter of that note cannot be read* | the block between the `---` lines is not valid YAML. numen will not repair it, because repairing means guessing at what you wrote. Fix the block and everything works again. |
| *that text cannot be written into a note* | what was handed over as the body of a note begins with a frontmatter block, and a body is the prose below one. |
| *a note cannot be called that* | the title leaves nothing a file could be named after, or carries a line break. |
| *a note of that name is filed there, so the note was renamed and its file was not* | the new title took, but something already sits where the file would go. Free the name and rename again. |

## About the vault

| | |
| --- | --- |
| *the vault is still being read* | the first scan has not finished. Searching works throughout; a few commands wait for it. |
| *nothing in front of you is a note* | the command acts on a note, and the tab in front of you holds a plex, a document or an agent. |
| *not following the vault* | numen has stopped noticing changes made to the files by other programs. Reopening the vault sets the watch up again. |
| *the vault could not be read* | the folder is gone, or the disk refuses it. Nothing has been done to your files. |
| *that folder is not there, or cannot be read* | the folder you chose cannot be opened. |
| *that folder is a copy of a vault this installation already holds* | two folders carrying one identity are not two vaults. Delete `.numen/` from the copy and add it again — see [vaults](/vault/#moving-the-folder). |
| *that folder is inside a vault already added, or holds one* | vaults do not overlap. One file belongs to one vault. |
| *a vault is already called that* | names on the list are unique. |
| *that is the vault in front of you* / *that is the only vault this installation has* | neither can be forgotten or erased; the window always stands on something. |
| *this machine has nowhere to put what is deleted* | erasing moves the folder to the machine's own place for deleted things, and this machine has none. Nothing was moved. |
| *a tab is holding text you have to answer for, so the window stayed where it was* | a vault swap was called off by an `overtaken` tab. Answer it, then switch. |

## About links

Reported against the note you would open to settle them:

| | |
| --- | --- |
| **dangling** | the link reaches nothing. Writing a note by that name mends it — no repair step, no rebuild. |
| **ambiguous** | more than one note has that name. The link reaches the nearest one, which is a fact about where the files currently sit, so it changes meaning when either of them moves. |
| *these notes link by a name that means another note now* | after a rename, links that now reach a different note. Which was meant is yours to settle. |
| *these notes link to nothing now* | after a removal, links whose target has gone. |

## About search

| | |
| --- | --- |
| *searching by words only — no model set* | search by meaning has nothing to read your notes with. The other two bands work as they always do. See [finding](/finding/). |
| *this vault has not been read for meaning yet* | there is a model, and it has not got to this vault. It is working behind the window. |
| *the vault could not answer* | the search reached the vault and got nothing back. |

## About the agent

| | |
| --- | --- |
| *the agent could not be reached* | the program numen starts to answer with is not installed, is not where it was expected, or would not start. See [the agent](/agent/#setting-it-up). |
| *the agent finished without saying anything* | it ran and produced no answer. |
| *the agent stopped here* | you stopped it, and what it had done up to that point stands. |

## About themes

| | |
| --- | --- |
| *the themes could not be listed* | the `themes/` folder is not there or cannot be read. What ships is still available. |
| *that theme could not be read, so it is not worn* | the file named in the settings is missing or unreadable. numen falls back to its own palette and says which name it could not find; putting the file back is all it takes. |
