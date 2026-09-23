---
title: Links
description: Links that carry a direction — parent, child, jump — how to make them, and how a name finds its note.
---

A link in numen carries a direction. One note hangs **under** another, or **over** it, or
**across** to it, and that is the difference between a pile of notes and a shape you can walk
through.

## The five roles

| | |
| --- | --- |
| <span class="up">`parent`</span> | the note this one hangs under. |
| <span class="down">`child`</span> | the note that hangs under this one. |
| <span class="across">`jump`</span> | a shortcut across the hierarchy, between notes that belong to different branches. |
| `ref` | a plain mention. This is what an ordinary `[[wikilink]]` in your prose becomes. |
| `attachment` | something that is not a note: a file in the vault, or an address on the web. It stays out of the hierarchy. |

The list is closed. A role numen does not know has no behaviour, and is reported rather than
guessed at.

## Making one

- **New child note**, **New parent note** and **New jump note** in the commands make the note and
  the link in one go, from the note you are on.
- In the [plex](/plex/), drag a node out of another node: you draw the connection and it exists.
- Searching for a note that does not exist offers to create it, as a child, as a parent or as a
  jump.
- Typing `[[a name]]` in your prose makes a `ref`.

## Following a link

Press a `[[wikilink]]` and the note it names opens in a tab beside the one you were reading. It works in your prose and in what the [agent](/agent/) answers, and a link that reaches nothing is drawn as reaching nothing and opens nothing.

Inside the brackets a note can also be named by its identity rather than by its name: `[[note://<identifier>]]`. numen writes that form where it made the link and already knew the identifier, or where a name matched several notes and picked none of them. It is read and followed the same way, it is left alone by every rename, and it reaches the note in whichever vault you have added holds it.

## What it looks like in the file

Links that carry a role live in the frontmatter, so your prose stays prose:

```yaml
---
links:
  - to: "[[Thermodynamics]]"
    role: parent
  - to: "[[Linear algebra]]"
    role: parent
    type: requires
    note: "only eigenvectors are needed"
  - to: "[[Entropy as disorder]]"
    role: jump
    label: "argues the other way"
---

Ordinary text with an [[ordinary link]], which is a ref.
```

`label` is the few words you call the relationship — they are drawn along the line between the
two notes in the plex. `note` is why the link exists, for you to read later. Both are optional,
and neither is read by anything but you.

## One edge, either end

Writing `parent: B` in note A and writing `child: A` in note B are the same edge. Whichever end
was convenient at the time is the end it lives in, and both notes show it.

**A note can have several parents.** The hierarchy is not a tree — a note that genuinely belongs
under two headings is filed under both, and neither copy is a duplicate.

Siblings are not stored. Notes with a parent in common are siblings, which is a question asked
whenever it is needed.

## Attachments

An `attachment` points at something that is not a note: a file sitting in the vault, or an
address on the web. Nothing tries to resolve it as a note, and it takes no part in the
hierarchy — a note with a scan attached to it is not the parent of that scan.

```yaml
---
links:
  - to: "scans/1897-letter.pdf"
    role: attachment
  - to: "https://example.org/the-paper"
    role: attachment
    label: "the paper this argues with"
---
```

## How a name finds its note

`[[Entropy]]` is a name, not a path to a file, and it is resolved every time it is asked. In
order:

1. an exact path from the top of the vault;
2. a path relative to the folder the linking note is in;
3. a single file of that name anywhere in the vault;
4. several files of that name — the link is **ambiguous**. It reaches the nearest one in the
   folder tree, and numen reports it so you can settle which was meant.

Case does not matter: `[[entropy]]` finds `Entropy.md`. A name that reaches nothing is
**dangling** — writing a note by that name is all it takes to mend it, and no repair step is
needed anywhere.

Inside the brackets, `|` gives the words the link reads as in your sentence and `#` names a
heading inside the target. Neither is part of the address, and both survive whatever happens to
the note.

Names resolve inside their own vault. Two vaults may each hold an `Entropy.md`, and neither is
the other's answer.

## After a rename or a move

When numen moves the note, it looks at everything that pointed at it, and repairs only the links
that now reach nothing — writing the name the note is filed under now. Nothing else in anyone's
file changes: your brackets, your spacing, your alias, your heading.

A link that now reaches a *different* note is reported rather than repaired. Which note you
meant is yours to settle.

A rename done outside numen, with the window closed, is a note gone from one name and arrived at
another with nothing connecting the two. Links to the old name are left dangling, and mending
them is a rename inside numen or an edit by hand.

## What gets reported

The vault is checked as it is read, and what turns up is filed against the note you would open
to settle it:

| | |
| --- | --- |
| a link with no target, or a role nobody knows | the entry cannot be read as a link. |
| frontmatter that is not valid YAML | the note is still indexed, and nothing can be written into it. |
| ambiguous | the link reaches more than one note. |
| dangling | the link reaches nothing. |
