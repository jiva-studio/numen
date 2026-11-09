---
title: Vaults
description: A vault is an ordinary folder of markdown files. How to open one, keep several, and take one off the list.
---

A vault is a folder. There is no import, no library and no database of your writing: markdown
files in folders you arranged, opened where they already are.

Any folder qualifies. A folder your other editor has been writing into for years is a vault the
moment you open it, and every file in it is a note from the start. Nothing is registered,
converted or rewritten.

## Opening one

**The first launch needs no setup.** numen does not create a folder for you. The window opens showing its name, the vaults it knows (none yet), and the
one thing to do, **Create vault**.

From then on it opens the vault you had last.

**Create vault** creates a folder, **Open vault** uses one you already have; either way your
machine's own folder picker is what chooses it. Both are in the commands —
<kbd>Ctrl</kbd>/<kbd>⌘</kbd> <kbd>P</kbd> — once a vault is open.

The first scan runs in the background: names and words are searchable almost at once, and larger
vaults continue to be indexed while you work. What is still being done is shown in the title bar of the
window.

## What is not a note

Two places are skipped whole:

- `.numen/`, which is where the vault keeps its own identity and what each plugin creates from its
  files, in a folder named after the plugin — the text read from scanned
  [documents](/documents/), the transcript of a recording;
- any folder whose name begins with a dot, which is where other tools keep their state.

Files removed from the vault are moved into `.trash/` inside it, which is a dot-folder and so is
out of the index as well. Nothing there is deleted until the command **Empty trash** is run.

## What a folder of yours becomes

|                    |                                                                                                                                                                 |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `.md` files        | notes: read, searched, linked and shown in the plex.                                                                                                            |
| `.pdf` and `.epub` | [documents](/documents/): their text is searched and a result opens the book at the page.                                                                       |
| everything else    | left exactly as it is. Images, spreadsheets, whatever else you keep there — numen does not touch them, and a note can [point at one](/links/) as an attachment. |

Nothing is moved into a structure of numen's own. The folders you arranged are the folders you
keep.

## Several vaults

An installation keeps a list of vaults and shows one at a time. Moving to another happens in the
window you already have: **Vaults** in the commands lists them, and choosing one opens it.

Before the swap, every open tab saves its unsaved changes. A tab with an unresolved choice (a note whose file changed outside numen) cancels the switch and stays open,
so nothing is left half-saved.

Rules for the list:

|                             |                                                                                                            |
| --------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Names are unique            | a folder added under a name another vault has gets a number appended, and you rename it afterwards.        |
| Vaults do not overlap       | a folder inside a vault you already have, or one that holds a vault, fails. One file belongs to one vault. |
| A whole disk is not a vault | neither is your home folder itself.                                                                        |

## Moving the folder

Move it, rename it, put it on another disk. The vault stores its identity inside its own
folder, so numen recognises it and updates the list with its new location.

Copying a vault is the one case that fails: two folders with the same identity are not
two vaults. Delete `.numen/` from the copy and add it again — it becomes a vault of its own,
with its own identity, and its notes are its own from then on.

## Taking one off the list

**Remove** removes the vault from the list and from the search index. The folder stays exactly
where it is, and adding it again brings back the same vault.

**Erase** removes it and moves the folder to wherever your machine puts deleted things, so you
can still restore it from there.

Neither can remove the vault you have open, and neither can remove the last one you have;
the window always has a vault open.
