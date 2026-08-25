---
title: Your first vault
description: A vault is an ordinary folder of markdown files. How to open one, keep several, and take one off the list.
---

A vault is a folder. There is no import, no library and no database of your writing: markdown
files in folders you arranged, opened where they already are.

Any folder qualifies. A folder your other editor has been writing into for years is a vault the
moment you open it, and every file in it is a note from the first byte — nothing is registered,
converted or rewritten.

## Opening one

Open the commands with <kbd>Ctrl</kbd>/<kbd>⌘</kbd> <kbd>P</kbd> and choose **New vault** for a
folder numen makes, or **Open vault** for a folder you already have. Your machine's own folder
picker opens.

The first scan runs behind the window: names and words are searchable almost at once, and larger
vaults keep filling in while you work. What is still being done is shown in the corner of the
window.

## What is not a note

Two places are skipped whole:

- `.numen/`, which is where the vault keeps its own identity and anything read out of scanned
  [documents](/documents/);
- any folder whose name begins with a dot, which is where other tools keep their state.

Files removed from the vault are moved into `.trash/` inside it, which is a dot-folder and so is
out of the index as well. Nothing there is destroyed.

Only markdown is read as a note. Everything else in the folder — images, PDFs, spreadsheets —
stays where it is and can be [attached to a note](/links/) or, where numen can read it, opened
as a [document](/documents/).

## Several vaults

An installation keeps a list of vaults and shows one at a time. Moving to another happens in the
window you already have: **Vaults** in the commands lists them, and choosing one puts it in front
of you.

Before the swap, every open tab writes what it owes. A tab holding a question you have not
answered — a note whose file changed underneath it — calls the swap off and stays where it is,
so nothing is left half-written.

A few rules the list keeps:

| | |
| --- | --- |
| Names are unique | a folder added under a name another vault has gets a number appended, and you rename it afterwards. |
| Vaults do not overlap | a folder inside a vault you already have, or one that holds a vault, is refused. One file belongs to one vault. |
| A whole disk is not a vault | neither is your home folder itself. |

## Moving the folder

Move it, rename it, put it on another disk. The vault carries its identity inside its own
folder, so numen recognises it and the list catches up with where it went.

Copying a vault is the one case that is refused: two folders carrying the same identity are not
two vaults. Delete `.numen/` from the copy and add it again — it becomes a vault of its own,
with its own identity, and its notes are its own from then on.

## Taking one off the list

**Forget** takes the vault off the list and out of the search index. The folder stays exactly
where it is, and adding it again brings back the same vault.

**Erase** forgets it and moves the folder to wherever your machine puts deleted things, so you
can still fetch it back from there.

Neither can take away the vault you are looking at, and neither can take the last one you have —
the window always stands on something.
