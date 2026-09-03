# ADR-0027: The stencil, the deck and the card

- **Status:** Accepted
- **Date:** 2026-08-27
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0001, ADR-0015, ADR-0017, ADR-0018, ADR-0031, ADR-0034

## Context

A flashcard is two things that change at different rates. The shape — which fields it has, and how they are laid out on the side a person is shown — is settled once and used a thousand times. The filling is what a person typed, and there is a great deal of it.

Every plain-text system that has tried this has had to choose where the shape is written and how one card is told from the next in a file. The vault format allows the application two additions to a markdown file and no more: frontmatter, and `[[wikilink]]` (ADR-0018). Whatever a card is, it is made of what markdown already renders.

## Decision

### One key says what a note is

The frontmatter key `type` says which of four a note is: `note`, `deck`, `stencil` or `preset`. The list is closed, and a note carrying no `type` is a `note`, which is nearly every note in a vault. What a preset is, is ADR-0034.

One key rather than one per kind is what makes the four exclusive: a file is one of them by the shape of the record, and no rule is needed to say it cannot be two. A value outside the list is a problem against the note, and the note is read as an ordinary note.

### A stencil declares fields and faces

A stencil is a note of `type: stencil`. Its fields are declared in the frontmatter key `fields`, in the order a person is asked for them.

The faces are the body. Each is a second-level heading, and under it a front and a back at the third level. A face is HTML with `{{Field}}` standing where a value goes.

A field is a name and nothing else. **A value is HTML**, and so is the face it is laid into: what a person writes there is drawn as the markup it is. A card is the one thing in a vault that is not markdown — the file around it still is, and the headings that cut it into cards and fields are markdown's.

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
```

A face naming a field the stencil does not declare is a problem against the stencil, and the rest of it is read as usual.

### A deck is a file of cards

A deck is a note of `type: deck`.

Each card is a second-level heading. The stencil it is cut by is a lone `[[wikilink]]` paragraph directly beneath it. Each field is a third-level heading, and the value is everything under that heading until the next one.

### A deck has sections

A first-level heading opens a section. The cards under it are its own until the next one. A section is a name and nothing else: no fields, no stencil, no schedule, no mark.

```markdown
---
type: deck
---

# Roots

## Compost, what is it made of ^k7m2xq9fzp

[[Term]]

### Question

Compost, what is it made of

### Answer

Leaves and peelings, turned and left to rot down
```

A deck need not have sections, and cards may stand before the first one. Two sections may carry one name, and an empty section is kept. A section is not carried from place to place; moving one would mean moving its cards, and nothing has asked for that.

**Text between a section's heading and its first card is the section's, and is kept exactly as it stands** — what a card already has between its stencil's wikilink and its first field, one level up. Nothing lays it out and nothing reads it; it is a person writing about their own deck, and a write puts it back where it was.

**A section is made at the end of the deck, and taking one away takes away its heading and nothing else.** Its cards stay where they stand, under whatever heading is above them now, and its own text stays too: a section is a name, so removing it removes a name. Carrying cards out with it would be a second act wearing one name.

**A card is made at the end of the section it was asked for**, and at the end of the cards standing before the first section where it was asked for none. A person asks by pressing the plus that stands in the run they are looking at, so a card lands where they were looking.

### Every field is written under its own heading

A stencil's fields are third-level headings under the card, **all of them, the first included**, and a field holds whatever a person writes under it over as many lines as they like.

The first field is the stencil's first — `fields[0]` — and no other reading of "first" is meant anywhere in this record. It is what a card's heading is read from, so it stays first: a stencil neither moves it nor takes it away.

A card writing one field twice is a problem against the deck, the first field included, and it is reported once.

A value is HTML and holds what HTML holds: paragraphs, lists, a table, a picture. It cannot run — the window serves itself under a policy that allows no script it did not serve, and no handler written in an attribute. One deck carries cards of as many stencils as the person likes, and a card whose stencil declares a field the card leaves out is a card with that field empty.

### The heading is the first field, read back

A card's heading is the first line of its first field, cut to fit one line. It holds nothing of its own: throw it away and the application writes it again from the field.

- The projection is taken over text whose line endings are normalised, and written back in the endings the file keeps.
- It stops at the first line break, and at a hundred and twenty characters, counted as a person counts them and not as bytes.
- It never cuts inside a `[[wikilink]]`, an embed or a run of emphasis: the cut falls before whichever of those it lands in.
- A first field that is empty, or holds only spaces, projects to a heading of nothing, and the card is drawn by its first field's box like any other.

A person editing the file by hand may leave the two disagreeing. The field is what stands; the next write puts the heading back in step, and nothing is reported, because there is nothing for a person to decide.

**A card whose stencil cannot be read is not reprojected.** Nothing can say which field is first, so the heading is left exactly as it stands. This is the one case where a stale heading survives a write.

### The heading carries the structure

A card's boundary is a heading and a field's name is a heading, so the structure of the file is the structure markdown already has. Headings are parts in the index, which makes a card searchable and addressable by `#` the day it is written, with nothing else built.

The format spends three heading levels: the first opens a section, the second opens a card, the third opens a field. A value's own heading is of the fourth level or below.

### A card carries a mark

At the end of a card's heading stands `^` and ten characters of `0123456789abcdefghjkmnpqrstvwxyz`, and that is what the card is for as long as it exists.

```markdown
## Compost, what is it made of ^k7m2xq9fzp
```

It is written when the card is made. A card typed into a deck by hand carries none until the application next writes that file, which is when it is given one.

**A mark is separated from the heading's text by one space, stands last on the line, and is read as a mark only at that length and in that alphabet.** Anything else at the end of a heading is heading text. A malformed mark is not a mark and the card is given one.

**The mark does not know which deck it is in.** A card moved to another deck, or to another vault, is the same card: the same mark, the same history. Nothing recomputes and nothing is lost.

**Two cards carrying one mark** — a card copied by hand — is a problem against the deck holding them. **Both are read and both are shown, each marked**: nothing a person wrote goes missing from the screen, and which of the two is meant is a thing only they know. Neither is given a new mark, because a machine choosing would be choosing which card keeps the history. Across two decks it is a question for whatever keeps a history, and this record does not answer it.

Ten characters is 1.1 × 10¹⁵ marks. A vault of a hundred thousand cards meets a collision about once in two hundred thousand vaults, which is why the check exists and why it is not a design constraint.

### The mark is not shown

A card's heading reaches the index as a heading and a part, and that text is what a search answers with and what the plex hangs under a deck. The mark is taken off before the heading is recorded: it is written for the file, not for a person reading a list.

### Where a card is made whole

Two of the rules above are the application's to keep, not the caller's: minting a mark for a card that carries none, and putting a stale heading back in step. Neither can be done where a deck's body is composed — one needs a generator, the other needs the card's stencil.

**A deck is made whole once, in the use case that writes it**, before the bytes reach the vault. Every caller — the window, the tools an agent uses — therefore gets the same file, and neither has to know the rules.

### A card may point at a note

A card takes ordinary links, in the frontmatter of no file — the wikilink under its heading names its stencil, and any other wikilink in a value is a `ref` like any other. A card that asks about a note points at that note the way anything else in this vault does, and nothing about a card requires it.

### The file holds no schedule

When a card is next due, how far apart its intervals have grown, and how it has been answered are not written into the deck.

A schedule is computed from the history of answers, deterministically and for free. That makes it a cache (ADR-0015), and it is kept with the application. The answers themselves are the vault's, in its service folder ([ADR-0031](0031-an-answer-is-an-artifact-a-schedule-is-a-cache.md)). The deck holds what a person wrote and nothing a machine worked out.

### A field renamed in a stencil is renamed in every card it cuts

A field's name is written twice over: once in the stencil that declares it, and once as a heading in every card of every deck cut by that stencil. Renaming it in one place alone leaves the values under a heading nothing declares.

So the rename reaches them. Every deck of the vault is read, the heading is rewritten in the cards that stencil cuts, and what stands under it is untouched. A deck the rename could not be written to keeps the old heading, and that is a problem against that deck.

### A deck is bounded larger than a note

A note is read up to a ceiling, and a deck is a file holding what would otherwise be a folder of them. It carries a bound of its own, several times the note's, and the two are separate numbers in [cards](../cards.md).

A deck over its bound is refused, and the refusal says which file and what the bound is. The size is taken from the file before it is opened, so nothing over the bound is read.

## Consequences

- A deck file grows long, and a person editing one by hand scrolls. The editor is where a deck is meant to be edited.
- A deck has a size a person can reach by writing, and reaching it means splitting a file the application asked them to keep as one.
- A field renamed in a stencil is a write to every deck that stencil cuts, so one edit to one file lands as many, and a deck that cannot be written keeps a heading nothing declares.
- A value cannot hold a heading of the three levels the format spends, and a person who writes one splits their card in half without being told.
- `type` and `fields` are ordinary English words taken as owned keys, and a person's own key of either name collides.
- `type` now means two things: what a note is, and what a link is for. The second is a key inside a link's own record and the first is a key about the whole note, so an indent tells them apart, and nothing else does.
- A field is a name in a list in a stencil and a heading in a deck. One concept, written two ways, and a reader has to learn both.
- Reordering a stencil's fields moves which one every card's heading is read from, and every deck it cuts has to be rewritten for it. Renaming the first field costs the same as renaming any other.
- **A card can be reordered, renamed and rewritten, and stays itself.** Nothing in a heading is load-bearing.
- **Two cards of one mark is a fault; two cards of one heading is not.** Their marks tell them apart, and nothing else needed them to differ.
- **A deck with no stencil is fully readable.** Every field stands under its own name, so nothing is lost with the stencil.
- **A first field of many lines makes a short heading.** What an outline shows is its first line, which is what an outline is for.
- **`[[Deck#^k7m2xq9fzp]]` reaches the deck**, as every fragment does today. Reaching the card itself is work for whoever wants it.
- Two files must agree for a card to be drawn, and a deck whose stencil was deleted holds cards that cannot be laid out. The values are still there and still readable.
- A deck copied to another machine arrives with the answers given to its cards, because those are kept beside it in the vault's service folder. What the machine works out from them it works out again ([ADR-0031](0031-an-answer-is-an-artifact-a-schedule-is-a-cache.md)).

## Alternatives considered

**A fenced block of the application's own, one to a card, holding the fields as YAML.** Rejected: a value becomes one line of YAML, so a field cannot hold a paragraph, a list or an image, and the cards on the screen this feature is for are exactly the ones with a picture in them.

**A separator in the line — `Question::Answer`, or a `?` on a line of its own** — as the Obsidian and Logseq plugins do. Rejected: it addresses a card with two halves and has nowhere to put the third, so a stencil with named fields cannot be expressed at all.

**A bold run naming each field — `**Height:** about 45"`.** Rejected: the field's name is then parsed out of the person's prose, and a colon inside a value is indistinguishable from the one that separates them.

**A table to a stencil, columns for fields and a row per card.** Rejected: a cell holds one line, which is the same wall the fenced block runs into, and the format this feature exists to draw has a photograph in it.

**A key of its own for each kind — `deck: true` beside `stencil: …`.** Rejected: two independent keys admit a file carrying both, so a rule has to say what that file is, and the rule is enforced nowhere the format itself can see.

**The fields declared in the body of the stencil, under a heading.** Rejected: the body is already spent on faces, so the fields need a level above them, and the two section names become English words a person may not use as a heading of their own. A collision on a frontmatter key is reported; a collision on a heading is silent.

**The fields taken from whatever `{{Field}}` the faces use.** Rejected: a field could not be made before it was placed, and taking a field off the last face it stood on would drop it from every card in every deck without anybody asking for that.

**A field carrying a stable key of its own, with its name beside it**, so a rename touches one file. Rejected: the key would stand in every card's heading in place of the name, and a deck opened in another editor would read as rows of identifiers.

**Leaving a renamed field's values where they are**, orphaned under the old heading. Rejected: nothing tells the person it happened, and the values are gone from every face while still filling the file.

**A name of the card's own, typed by a person.** Rejected: five hundred words is five hundred names nobody wants to invent, and what a person would write is `Card 1`, `Card 2` — a column of identifiers wearing the costume of names. Neither Anki nor Mochi gives a card a name outside its fields; Anki's first field is what a duplicate is judged on, and Mochi's template names one field as the card's title.

**The card's place in the deck, written in the heading.** Rejected: a number a reorder rewrites reads as an order, and a number a reorder leaves alone reads as a broken one. The order of the cards is the order of the file.

**Keeping the heading as the field, and a first field of more than one line written out below it as well.** Rejected: the value stands in two places with nothing keeping them in step, which is the fault the format already reports as a field written twice.

**Refusing a first field of more than one line**, as Anki's sort field and Mochi's title field do, and asking a stencil to put something short first. Rejected: it is a rule about how a person must think of their own cards, made to suit the file.

**A mark written at first review**, so a deck nobody has begun to learn holds nothing machine-written. Rejected: the editor writes the deck in the first place, so there is no untouched file to keep clean, and a card with no mark would need a second way to be addressed and a rule for when each applies.

**A ULID**, as a note carries (ADR-0019). Rejected: twenty-six characters at the end of every heading, to buy uniqueness across every vault that will ever exist, where the question is whether two cards in one vault collide. Creation order goes with it, and nothing here asks for creation order.

**A namespace on the mark, `^card/…`.** Rejected: it was needed where a card was a line in an ordinary note among a person's own anchors. It buys nothing where a card is a heading in a deck.

**The mark on the line below, beside the stencil's wikilink.** Rejected: a card's mark and a person's anchor to that card are the same want, and markdown puts an anchor at the end of the block's own line.

**One file per card.** Rejected: a deck of a thousand cards is a thousand files in a folder a person opens, and every one of them is a note in the index, in search, and in the plex.

**The schedule in the deck, beside the card it belongs to**, as org-fc and the Obsidian plugin do. Rejected: the file is then rewritten after every answer, so a review session is a diff over the whole deck, and what the person wrote and what the machine counted sit in one file with one modification time.
