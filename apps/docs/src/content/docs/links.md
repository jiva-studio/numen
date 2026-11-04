---
title: Links
description: Links that have a direction (parent, child, jump), how to make them, and how a name is matched to a note.
---

A link in numen has a direction. One note is **under** another, or **over** it, or
**across** from it. This turns a set of unrelated notes into a structure you can navigate.

## The five roles

|                                    |                                                                                                             |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| <span class="up">`parent`</span>   | the note this one is placed under.                                                                          |
| <span class="down">`child`</span>  | the note that is placed under this one.                                                                     |
| <span class="across">`jump`</span> | a shortcut across the hierarchy, between notes that belong to different branches.                           |
| `ref`                              | a plain mention. This is what an ordinary `[[wikilink]]` in your prose becomes.                             |
| `attachment`                       | something that is not a note: a file in the vault, or an address on the web. It stays out of the hierarchy. |

The list is closed. A role numen does not know has no behaviour, and is reported rather than
guessed at.

## Making one

- **Create child note**, **Create parent note** and **Create jump note** in the commands make the note and
  the link in one go, from the note you are on.
- In the [plex](/plex/), drag a node out of another node: you draw the connection and the link is created.
- Searching for a note that does not exist offers to create it, as a child, as a parent or as a
  jump.
- Typing `[[a name]]` in your prose makes a `ref`.

## Following a link

Click a `[[wikilink]]` and the note it points to opens in a tab beside the one you were reading. This works in your prose and in the [agent](/agent/)'s answers. A link to a note that does not exist is shown as unresolved and opens nothing.

Inside the brackets a note can also be named by its identity rather than by its name: `[[note://<identifier>]]`. numen writes that form where it made the link and already knew the identifier, or where a name matched several notes and picked none of them. It is read and followed the same way, it is not changed by any rename, and it finds the note in whichever vault you have added contains it.

## What it looks like in the file

Links that have a role are stored in the frontmatter, so your prose stays prose:

```yaml
---
links:
  - to: '[[Thermodynamics]]'
    role: parent
  - to: '[[Linear algebra]]'
    role: parent
    type: requires
    note: 'only eigenvectors are needed'
  - to: '[[Entropy as disorder]]'
    role: jump
    label: 'argues the other way'
---
Ordinary text with an [[ordinary link]], which is a ref.
```

`label` is the few words you use to describe the relationship. They are shown along the line between the
two notes in the plex. `note` is why the link exists, for you to read later. Both are optional,
and neither is read by anything but you.

## Arrows

`marker: arrow` draws an arrowhead at the end of the line that points at the link's target:

```yaml
links:
  - to: '[[Entropy as disorder]]'
    role: jump
    marker: arrow
```

Without `marker` nothing is drawn at that end. Each note sets its own end, so a line
can have no arrow, one arrow, or an arrow at each end, and the arrow does not change what kind
of link it is. A `marker` numen does not know is reported on the note, as an unknown role is.

## One edge, either end

Writing `parent: B` in note A and writing `child: A` in note B are the same edge. The link is stored
in whichever note you wrote it in, and both notes show it.

**A note can have several parents.** The hierarchy is not a tree. A note that belongs
under two parents is listed under both, and neither entry is a duplicate.

Siblings are not stored. Notes with a parent in common are siblings, and numen works that out
whenever it is needed.

## Attachments

An `attachment` points at something that is not a note: a file in the vault, or an
address on the web. Nothing tries to resolve it as a note, and it takes no part in the
hierarchy — a note with a scan attached to it is not the parent of that scan.

```yaml
---
links:
  - to: 'scans/1897-letter.pdf'
    role: attachment
  - to: 'https://example.org/the-paper'
    role: attachment
    label: 'the paper this argues with'
---
```

## How a name is matched to a note

`[[Entropy]]` is a name, not a path to a file, and it is resolved each time it is needed. In
order:

1. an exact path from the top of the vault;
2. a path relative to the folder the linking note is in;
3. a single file of that name anywhere in the vault;
4. several files of that name — the link is **ambiguous**. It resolves to the nearest one in the
   folder tree, and numen reports it so you can decide which was meant.

Case does not matter: `[[entropy]]` finds `Entropy.md`. A name that matches no note is
**dangling**. Creating a note with that name fixes it, and no repair step is
needed.

Inside the brackets, `|` sets the words shown for the link in your sentence and `#` names a
heading inside the target. Neither is part of the address, and both are kept when the note
changes.

Names resolve inside their own vault. Two vaults may each hold an `Entropy.md`, and a link in one vault never resolves to the other's.

## After a rename or a move

When numen moves the note, it checks every link that pointed to it and repairs only the links
that now point to nothing, writing the name the note has now. Nothing else in anyone's
file changes: your brackets, your spacing, your alias, your heading.

A link that now points to a _different_ note is reported rather than repaired. You decide which note you
meant.

A rename done outside numen, with the window closed, looks to numen like one note deleted and another created, with nothing connecting the two. Links to the old name are left dangling. To fix them, rename the note inside numen or edit the links by hand.

## What gets reported

The vault is checked as it is read, and each problem is reported on the note you would open
to fix it:

|                                               |                                                                |
| --------------------------------------------- | -------------------------------------------------------------- |
| a link with no target, or a role nobody knows | the entry cannot be read as a link.                            |
| frontmatter that is not valid YAML            | the note is still indexed, and nothing can be written into it. |
| ambiguous                                     | the link points to more than one note.                         |
| dangling                                      | the link points to no note.                                    |
