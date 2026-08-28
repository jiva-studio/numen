# Cards

A card is a set of named values a person wrote, laid out by a stencil into a front and a back. This is a specification, not a decision record: every rule here traces to an accepted ADR.

Two files are involved and both are ordinary notes. A **stencil** says what fields a card has and how they are shown. A **deck** holds the cards.

## What a note is

The frontmatter key `type` says which of three a note is.

| `type` | What the file is |
| --- | --- |
| `note` | An ordinary note. This is the default, and nearly every note in a vault carries no `type` at all. |
| `stencil` | Fields and faces. |
| `deck` | Cards. |

The list is closed. A value outside it is a problem against the note, and the note is read as an ordinary note.

## The stencil

```markdown
---
type: stencil
fields:
  - Name
  - Height
  - Weight
  - Life span
---

## Recognise

### Front

{{Name}}

### Back

**Height:** {{Height}}
**Weight:** {{Weight}}
**Life span:** {{Life span}}

## Name it

### Front

Which animal is {{Height}} at the shoulder and lives {{Life span}}?

### Back

{{Name}}
```

### Fields

`fields` is a list of names, in the order a person is asked for them. A name is text; there are no field types, and a value is markdown.

A name is compared as written. Two fields of one name are a problem against the stencil, and the first stands.

**The first field is the one a card's heading is read from.** A stencil declaring `Question` first gives a deck whose headings are questions, and the question is typed once, under its own heading like every other field.

A stencil declaring no field at all cuts nothing: its cards have nothing to be filled with. It is a problem against the stencil.

Every field holds whatever a person writes under it, over as many lines as they like. The first is no exception: a field wanting a paragraph, a list or a picture may stand anywhere in the order, and what reaches the heading is one line of it.

### The first field stays first

The editor does not move the first field, does not move another field above it, and does not remove it. The three are one act under three names: each of them hands the reading of every card's heading to a different field, and every heading in every deck the stencil cuts is rewritten from a field it was not written for.

The fields below the first are reordered freely. Their order is the order a person is asked for them and nothing else, and moving them writes to no deck.

A person who edits `fields` by hand and moves the first field is not stopped and cannot be. Nothing is lost from the file: every value stands under its own heading, and from the next write of a deck on, its cards are headed by the field that now stands first.

### Renaming a field

A field renamed in a stencil is renamed in every card that stencil cuts. Each deck of the vault is read, the heading is rewritten in the cards cut by that stencil, and the value under it is left as it was. The rename reaches decks that are not open, so it is one edit to one file and a write to as many as hold cards of it.

A deck that cannot be written — a deck whose frontmatter does not parse, a file the filesystem refuses — keeps the old heading, and that is a problem against the deck.

A rename is the application changing what is in a file, so a deck it reaches carries an `id` afterwards if it carried none ([ADR-0019](adr/0019-a-note-is-identified-by-a-ulid.md)). One rename can therefore write an identifier into a deck a person wrote by hand and never opened here.

### Faces

Each second-level heading is one face, and the heading is what the face is called. Under it, `### Front` and `### Back` hold what is shown before and after the answer. A face missing either is a problem, and it lays out nothing.

A side runs to the next `### Front`, the next `### Back` or the next face, so a third-level heading of any other name stands inside the side it is written under, and a second `### Front` or `### Back` is text under the first.

A stencil has as many faces as a person writes. One card is shown once through each.

### Placeholders

`{{Field}}` stands where a value goes, and the name inside is a field's name written exactly. The first field is placed the same way, by its own name, and what it lays out is the whole value under that field's heading, not the one line of it the card is headed by.

A placeholder naming a field the stencil does not declare is a problem against the stencil. A placeholder whose card leaves that field empty lays out as nothing.

Everything around a placeholder is markdown and is drawn as markdown.

`{{` is not escaped. Wherever the two characters stand in a face, what runs to the next `}}` is read as a placeholder, and a face that wants those characters as text has no way to write them. This is a known limit of the format.

## The deck

```markdown
---
type: deck
---

# Animals

## Llama ^k7m2xq9fzp

[[Animal]]

### Name

Llama

### Height

about 45" (shoulder)

### Life span

about 20 years

# The garden

## Leaf mould ^3n8vr4tqch

[[Term]]

### Term

Leaf mould

### Meaning

Compost made of fallen leaves alone, left two winters

### Source

[[The compost heap]]
```

### Sections

Each first-level heading opens a section, and the cards under it are its own until the next one. A section is a name and nothing else: no fields, no stencil, no schedule, no mark.

A deck need not have sections, and cards may stand before the first one. Two sections may carry one name, and a section holding no card is kept.

Text between a section's heading and its first card is the section's, and is kept exactly as it stands — what a card has between its stencil's wikilink and its first field, one level up. Nothing lays it out and nothing reads it.

A section is made at the end of the deck, as a card is. Taking one away takes away its heading and nothing else: its cards stay where they stand, under whatever heading is above them now, and its own text stays with them.

### Cards

Each second-level heading is one card. A card runs to the next second-level heading, to the next first-level heading, or to the end of the file.

A card's heading is the first line of its first field, cut to one line. It holds nothing of its own: throw it away and it is written again from the field. The cut stops at the first line break and at a hundred and twenty characters, counted as a person counts them, and it never falls inside a `[[wikilink]]`, an embed or a run of emphasis — where it would, it falls before whichever of those it lands in.

A first field that is empty, or holds only spaces, is headed by nothing. The card is read and shown like any other, and its first field is a box to fill.

Two cards may carry one heading, which is what two cards beginning alike come to. Their marks tell them apart and nothing else needs to.

A person editing the file by hand may leave a heading and its first field disagreeing. The field is what stands; the next write of that deck puts the heading back in step, and nothing is reported, because there is nothing for a person to decide. A card whose stencil cannot be read is the exception: nothing can say which field is first, so the heading is left exactly as it stands.

### The mark

At the end of a card's heading stands `^` and ten characters of `0123456789abcdefghjkmnpqrstvwxyz`. That is what the card is for as long as it exists.

```markdown
## Compost, what is it made of ^k7m2xq9fzp
```

A mark is separated from the heading's text by one space and stands last on the line, and it is read as a mark only at that length and in that alphabet. Anything else at the end of a heading is heading text, and a malformed mark is not a mark: the card is given one.

A mark is written when the card is made. A card typed into a deck by hand carries none until the application next writes that file, which is when it is given one.

The mark does not know which deck it is in. A card moved to another deck, or to another vault, is the same card: the same mark, and the same history behind it. Nothing is recomputed and nothing is lost.

Two cards of one mark in one deck are a problem against that deck.

### The preamble and the tail

Text above the first heading of the body is the deck's preamble. It is kept verbatim and it is no card.

A value is read without the blank lines standing at either end of it, so the bytes that separate one field from the next belong to neither. What the file ends with once the last value has been read is the tail, and it is kept verbatim the same way. A deck with no heading in its body is a deck of no cards, and its whole body is preamble.

### The stencil a card is cut by

A lone `[[wikilink]]` paragraph directly beneath the card's heading names the stencil. It is an ordinary link and resolves the ordinary way ([Links](links.md)).

A card whose first paragraph is not a lone wikilink has no stencil, and so does a card whose wikilink resolves to a note that is not one. Both are problems against the deck. The values are read either way, and nothing is lost from the file.

Text between that wikilink and the card's first field is kept verbatim and is no field's value. Nothing lays it out.

One deck carries cards of as many stencils as a person likes.

### Values

Each third-level heading is a field, and the value is everything under it until the next heading of the first, second or third level. Every field of the stencil is written this way, the first included.

A value is markdown and holds what markdown holds: paragraphs, lists, a table, a quote, an image as `![[llama.jpg]]`. What it cannot hold is a heading at the first, second or third level, because those three are spent. A value wanting a heading uses one of the fourth level or below.

A card carrying a field its stencil does not declare keeps it in the file, and no face shows it. A field the stencil declares and the card omits is empty, and so is one whose heading is there with nothing under it.

Two fields of one name in one card are a problem against the deck. The first stands, and the second is kept in the file.

### The order of fields

On the screen a card's fields are in the stencil's order, whatever order the file wrote them in.

In the file they stay where they are. A card the editor did not touch arrives on the other side in the order it went in, and a field is moved only by a person moving it.

## Size

A note is read up to a megabyte ([Editing](editing.md)). A deck is a file of many cards, so it has a bound of its own: eight megabytes. A stencil is a note and has the note's.

A deck over its bound is not read, and nothing about that is silent: the interface says which file it was and what the bound is, and the deck is opened elsewhere or split. The size is asked of the file before it is opened, so a deck over the bound is refused with none of its bytes read.

## What is reported

A problem is something that could not be acted on and was not guessed at, filed against the note somebody would open to settle it. For everything below that note is the stencil or the deck the text was written in, and never the other file.

| Problem | Against | What it is |
| --- | --- | --- |
| a `type` outside the list | the note | `type` is not `note`, `deck` or `stencil`. The note is read as an ordinary note. |
| two fields of one name | the stencil | `fields` declares a name twice. The first stands. |
| a stencil declaring no field | the stencil | `fields` is absent or empty, so a card cut by it has nothing to be filled with. |
| a face missing a side | the stencil | no `### Front` or no `### Back` under a second-level heading. The face lays out nothing. |
| a placeholder nobody declared | the stencil | `{{Field}}` naming a field outside `fields`. The rest of the face is read as usual. |
| a card with no stencil | the deck | the first paragraph under the card's heading is not a lone wikilink. The values are read. |
| a stencil that is not one | the deck | the wikilink resolves to a note whose `type` is not `stencil`. The values are read. |
| two cards of one mark | the deck | two cards of one deck carry the same `^` and ten characters. Both are read and shown. |
| two fields of one name | the deck | one card writes a heading twice. The first stands. |
| a deck that could not be written | the deck | a field renamed in a stencil did not reach this deck, so a card holds a heading the stencil no longer declares. |
| a deck over its bound | the deck | eight megabytes, and the file was not read. |

The stencil a card names is an ordinary link, so a wikilink reaching nothing or reaching two notes is `dangling` and `ambiguous` as anywhere else ([Links](links.md)).

A card carrying a field its stencil does not declare is not a problem. Neither is a value holding a heading of a level the format has spent: the file is split at that heading, and it says what it says. Neither is a card carrying no mark, which is given one, nor a heading standing out of step with its first field, which is written again from it.

## What is not in these files

**No schedule.** When a card is due, how long its intervals have grown and how it has been answered are computed from the history of answers, so they are a cache and live in the index ([ADR-0015](adr/0015-a-books-text-is-a-cache-or-an-artifact.md)). A deck is not rewritten because a card was reviewed.

## What is searched

A deck and a stencil are not cut into chunks, and nothing in them is embedded. The full search does not reach inside a card and neither does the semantic one: a card is found by its heading, which is its question, and a stencil by its title, which is the name of its file.

Of a deck's headings only the ones a person wrote are kept — its sections and its cards. The third level is the stencil's field names written out under every card, and it is not kept. A stencil keeps no heading at all. A card's heading is kept without its mark, which is written for the file and not for a person reading a list.

In every other way a deck and a stencil are notes: a title, a type, an identifier, their links resolved and their backlinks answered, and a node in the plex with their headings hanging under it.

## Handling of existing files

A stencil and a deck are notes, so everything in [Note format](note-format.md) holds for them: unknown frontmatter keys are preserved verbatim, line endings are preserved per file, a file that fails to parse is reported and not rewritten, and the application does not rewrite what it did not change.

A deck edited in the editor keeps the order of the file: the sections, the cards inside them, and the fields inside those are written back where they were. Every part of the file the editor did not touch arrives on the other side as the bytes it went in as — the preamble, the tail, the text under a card's wikilink, and a field no stencil declares.

A deck is made whole where it is written, before its bytes reach the vault: a card carrying no mark is given one, and a heading standing out of step with its first field is written again from it. A window and the tools an agent uses therefore leave the same file behind.
