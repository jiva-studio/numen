---
title: Troubleshooting
description: What numen says when it will not do something, what it means, and what to do about it.
---

numen reports what it cannot do in the words below. Nothing is written into a file
it could not read, and numen does not guess on your behalf. This page lists those messages.

## The mark on a tab

A note's tab shows one word beside its title when it is not simply saved.

|             |                                                                                                                                                                      |
| ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `unsaved`   | what you see has not been saved to the file yet. It will be saved in a moment; <kbd>Ctrl</kbd>/<kbd>⌘</kbd> <kbd>S</kbd> saves it now.                               |
| `overtaken` | the file changed outside the tab, so saving stopped. Choose **keep mine** or **take the file's**, and saving starts again. Nothing you typed is lost while it waits. |
| `gone`      | the file is no longer at the name this tab is at. What is on screen is still yours: **keep mine** creates the file again there.                                      |
| `stuck`     | this note cannot be read or written, and the tab says which of the reasons below it is.                                                                              |

_These notes stopped saving because their files changed. The window waits._ — the same message
shown for more than one tab at once, usually after a sync client has changed files in the vault.

## About a note

|                                                                                    |                                                                                                                                                                              |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| _that note is not in the vault_                                                    | the name matches no note. It may have been moved or removed outside numen.                                                                                                   |
| _that file is not a note_                                                          | its extension is not one this vault reads as notes.                                                                                                                          |
| _that file is not text_                                                            | not valid UTF-8, so it is not opened here.                                                                                                                                   |
| _that note is longer than this writes_                                             | over a megabyte. Split it, or edit it elsewhere.                                                                                                                             |
| _the frontmatter of that note cannot be read_                                      | the block between the `---` lines is not valid YAML. numen will not repair it, because repairing means guessing at what you wrote. Fix the block and everything works again. |
| _that text cannot be written into a note_                                          | what was passed as the body of a note begins with a frontmatter block, and a body is the prose below one.                                                                    |
| _a note cannot be called that_                                                     | the title leaves nothing a file could be named after, or contains a line break.                                                                                              |
| _a note of that name is filed there, so the note was renamed and its file was not_ | the new title took, but a file already exists at that path. Free the name and rename again.                                                                                  |

## About the vault

|                                                                                      |                                                                                                                                                   |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| _the vault is still being read_                                                      | the first scan has not finished. Searching works throughout; a few commands wait for it.                                                          |
| _nothing in front of you is a note_                                                  | the command acts on a note, and the open tab holds a plex, a document or an agent.                                                                |
| _not following the vault_                                                            | numen no longer detects changes made to the files by other programs. Reopening the vault starts watching again.                                   |
| _the vault could not be read_                                                        | the folder is gone, or the disk cannot be read. Nothing has been done to your files.                                                              |
| _that folder is not there, or cannot be read_                                        | the folder you chose cannot be opened.                                                                                                            |
| _that folder is a copy of a vault this installation already holds_                   | two folders with the same identity are not two vaults. Delete `.numen/` from the copy and add it again — see [vaults](/vault/#moving-the-folder). |
| _that folder is inside a vault already added, or holds one_                          | vaults do not overlap. One file belongs to one vault.                                                                                             |
| _a vault is already called that_                                                     | names on the list are unique.                                                                                                                     |
| _that is the vault in front of you_ / _that is the only vault this installation has_ | neither can be removed or erased; the window always has a vault open.                                                                             |
| _this machine has no trash_                                                          | erasing moves the folder to the machine's own trash, and this machine has none. Nothing was moved.                                                |
| _a tab is holding text you have to answer for, so the window stayed where it was_    | a vault switch was cancelled by an `overtaken` tab. Resolve it, then switch.                                                                      |

## About links

Three of these are found whenever the vault is read, and dangling links only when they are asked
for by name. Ask the [agent](/agent/) — _what could this vault not make sense of?_ — and it shows
you the list; [`numen-cli problems`](/cli/) prints the same thing.

Each is reported on the note you would open to fix it:

|                                                          |                                                                                                                                                                                            |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **dangling**                                             | the link points to no note. Creating a note with that name fixes it, with no repair step and no rebuild.                                                                                   |
| **ambiguous**                                            | more than one note has that name. The link resolves to the nearest one. That depends on where the files currently are, so the link can resolve to a different note when either file moves. |
| _these notes link by a name that means another note now_ | after a rename, links that now point to a different note. You decide which was meant.                                                                                                      |
| _these notes link to nothing now_                        | after a removal, links whose target has been removed.                                                                                                                                      |

## About search

|                                                |                                                                                                                        |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| _searching by words only — no model set_       | search by meaning has no model to read your notes. The other two bands work as they always do. See [search](/search/). |
| _this vault has not been read for meaning yet_ | there is a model, and it has not read this vault yet. It is working in the background.                                 |
| _the vault could not answer_                   | the search reached the vault but received no response.                                                                 |

## About the agent

|                                              |                                                                                                                                                       |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| _the agent could not be reached_             | the program numen starts to run the agent is not installed, is not where it was expected, or failed to start. See [the agent](/agent/#setting-it-up). |
| _the agent finished without saying anything_ | it ran and produced no answer.                                                                                                                        |
| _the agent stopped here_                     | you stopped it, and what it had done up to that point is kept.                                                                                        |

## About themes

|                                  |                                                                                                                                                                   |
| -------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| _the themes could not be listed_ | the `themes/` folder is not there or cannot be read. What ships is still available.                                                                               |
| _Theme unreadable_               | the file named in the settings is missing or unreadable. numen falls back to its own palette and shows which name it could not find; restoring the file fixes it. |

## About the settings

|                                     |                                                                                                                                                                                                             |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| _that setting could not be written_ | a choice made in the commands — a theme, a size, whether [a title and a filename are one name](/editor/#renaming) — was not written to `numen.json`. The window keeps its previous setting; check the file. |
