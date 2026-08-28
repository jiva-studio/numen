# Cards

A card is a set of named values a person wrote, laid out by a stencil into a front and a back. This is a specification, not a decision record: every rule here traces to [ADR-0027](adr/0027-the-stencil-and-the-deck.md).

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

**The first field names the card.** Its value is written in the card's heading and nowhere else, so a card has no name apart from its fields. A stencil declaring `Question` first gives a deck whose headings are questions, and the question is typed once.

A stencil declaring no field at all cuts nothing: its cards have no name to stand in a heading. It is a problem against the stencil.

Because the first field is a heading, it holds one line. A field wanting a paragraph, a list or a picture is not the one to put first.

### The first field stays first

The editor does not move the first field, does not move another field above it, and does not remove it. The three are one act under three names: each of them hands the naming of every card to a different field, and every heading in every deck the stencil cuts then belongs to a field it was not written for.

The fields below the first are reordered freely. Their order is the order a person is asked for them and nothing else, and moving them writes to no deck.

**Renaming the first field is free.** It is written in `fields` and in the `{{ }}` of the faces, and in no deck at all, so the rename touches the stencil alone.

A person who edits `fields` by hand and moves the first field is not stopped and cannot be: a heading is a line of text, and it reads as the value of whichever field now stands first. Nothing is lost from the file and every card is labelled by the wrong field.

### Renaming a field

A field renamed in a stencil is renamed in every card that stencil cuts. Each deck of the vault is read, the heading is rewritten in the cards cut by that stencil, and the value under it is left as it was. The rename reaches decks that are not open, so it is one edit to one file and a write to as many as hold cards of it.

A deck that cannot be written — a deck whose frontmatter does not parse, a file the filesystem refuses — keeps the old heading, and that is a problem against the deck.

A rename is the application changing what is in a file, so a deck it reaches carries an `id` afterwards if it carried none ([ADR-0019](adr/0019-a-note-is-identified-by-a-ulid.md)). One rename can therefore write an identifier into a deck a person wrote by hand and never opened here.

### Faces

Each second-level heading is one face, and the heading is what the face is called. Under it, `### Front` and `### Back` hold what is shown before and after the answer. A face missing either is a problem, and it lays out nothing.

A side runs to the next `### Front`, the next `### Back` or the next face, so a third-level heading of any other name stands inside the side it is written under, and a second `### Front` or `### Back` is text under the first.

A stencil has as many faces as a person writes. One card is shown once through each.

### Placeholders

`{{Field}}` stands where a value goes, and the name inside is a field's name written exactly. The first field is placed the same way, by its own name, and what it lays out is the card's heading.

A placeholder naming a field the stencil does not declare is a problem against the stencil. A placeholder whose card leaves that field empty lays out as nothing.

Everything around a placeholder is markdown and is drawn as markdown.

`{{` is not escaped. Wherever the two characters stand in a face, what runs to the next `}}` is read as a placeholder, and a face that wants those characters as text has no way to write them. This is a known limit of the format.

## The deck

```markdown
---
type: deck
---

## Llama

[[Animal]]

### Height

about 45" (shoulder)

### Life span

about 20 years

## Leaf mould

[[Term]]

### Meaning

Compost made of fallen leaves alone, left two winters

### Source

[[The compost heap]]
```

### Cards

Each second-level heading is one card, and the heading holds the value of the stencil's first field. That value is what the card is called. A card runs to the next second-level heading or to the end of the file.

A card writing its first field as a `###` heading of its own carries that field twice. The heading above stands, and the third-level one is kept in the file and shown by no face. It is a problem against the deck.

A heading with no text leaves the first field empty, so the card has no name. It is a problem against the deck, and the card is read and shown marked.

Two cards of one name are a problem against the deck. The first stands as that name; the rest are read and shown marked, so nothing a person wrote goes missing from the screen.

### The preamble and the tail

Text above the first card is the deck's preamble. It is kept verbatim and it is no card.

A value is read without the blank lines standing at either end of it, so the bytes that separate one field from the next belong to neither. What the file ends with once the last value has been read is the tail, and it is kept verbatim the same way. A deck with no second-level heading in it is a deck of no cards, and its whole body is preamble.

### The stencil a card is cut by

A lone `[[wikilink]]` paragraph directly beneath the card's heading names the stencil. It is an ordinary link and resolves the ordinary way ([Links](links.md)).

A card whose first paragraph is not a lone wikilink has no stencil, and so does a card whose wikilink resolves to a note that is not one. Both are problems against the deck. The values are read either way, and nothing is lost from the file.

Text between that wikilink and the card's first field is kept verbatim and is no field's value. Nothing lays it out.

One deck carries cards of as many stencils as a person likes.

### Values

Each third-level heading is a field, and the value is everything under it until the next heading of either level.

A value is markdown and holds what markdown holds: paragraphs, lists, a table, a quote, an image as `![[llama.jpg]]`. What it cannot hold is a heading at the second or third level, because those two are spent. A value wanting a heading uses a deeper one.

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
| a stencil declaring no field | the stencil | `fields` is absent or empty, so a card cut by it has nothing to be named by. |
| the first field written twice | the deck | a card carries its first field as a `###` heading as well as in its own heading. The heading stands. |
| a face missing a side | the stencil | no `### Front` or no `### Back` under a second-level heading. The face lays out nothing. |
| a placeholder nobody declared | the stencil | `{{Field}}` naming a field outside `fields`. The rest of the face is read as usual. |
| a card with no stencil | the deck | the first paragraph under the card's heading is not a lone wikilink. The values are read. |
| a stencil that is not one | the deck | the wikilink resolves to a note whose `type` is not `stencil`. The values are read. |
| a card with no name | the deck | a second-level heading with no text. The card is shown marked. |
| two cards of one name | the deck | the first stands; the rest are shown marked. |
| two fields of one name | the deck | one card writes a heading twice. The first stands. |
| a deck that could not be written | the deck | a field renamed in a stencil did not reach this deck, so a card holds a heading the stencil no longer declares. |
| a deck over its bound | the deck | eight megabytes, and the file was not read. |

The stencil a card names is an ordinary link, so a wikilink reaching nothing or reaching two notes is `dangling` and `ambiguous` as anywhere else ([Links](links.md)).

A card carrying a field its stencil does not declare is not a problem. Neither is a value holding a heading of a level the format has spent: the card is split at that heading, and the file says what it says.

## What is not in these files

**No schedule.** When a card is due, how long its intervals have grown and how it has been answered are computed from the history of answers, so they are a cache and live in the index ([ADR-0015](adr/0015-a-books-text-is-a-cache-or-an-artifact.md)). A deck is not rewritten because a card was reviewed.

**No identifiers.** A card is named by its deck and its heading. Renaming a heading makes a different card.

## Handling of existing files

A stencil and a deck are notes, so everything in [Note format](note-format.md) holds for them: unknown frontmatter keys are preserved verbatim, line endings are preserved per file, a file that fails to parse is reported and not rewritten, and the application does not rewrite what it did not change.

A deck edited in the editor keeps the order of the file: the cards, and the fields inside them, are written back where they were. Every part of the file the editor did not touch arrives on the other side as the bytes it went in as — the preamble, the tail, the text under a card's wikilink, and a field no stencil declares.
