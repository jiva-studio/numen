# ADR-0027: The stencil and the deck

- **Status:** Accepted
- **Date:** 2026-08-27
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0001, ADR-0015, ADR-0017, ADR-0018, ADR-0026, ADR-0028

## Context

A flashcard is two things that change at different rates. The shape — which fields it has, and how they are laid out on the side a person is shown — is settled once and used a thousand times. The filling is what a person typed, and there is a great deal of it.

Every plain-text system that has tried this has had to choose where the shape is written and how one card is told from the next in a file. The vault format allows the application two additions to a markdown file and no more: frontmatter, and `[[wikilink]]` (ADR-0018). Whatever a card is, it is made of what markdown already renders.

## Decision

### One key says what a note is

The frontmatter key `type` says which of three a note is: `note`, `deck` or `stencil`. The list is closed, and a note carrying no `type` is a `note`, which is nearly every note in a vault.

One key rather than one per kind is what makes the three exclusive: a file is one of them by the shape of the record, and no rule is needed to say it cannot be two. A value outside the list is a problem against the note, and the note is read as an ordinary note.

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

### The first field is the card's name

**Superseded by [ADR-0028](0028-a-card-is-named-by-what-it-holds.md).** A card's heading is the first line of its first field, read back and holding nothing of its own, and the card is addressed by a mark it carries. Every field is written under its own heading. What follows in this section is the decision as it stood.

A stencil's **first field is written in the heading**, and nowhere else. The heading is that field's value, and it is what the card is called and what it is addressed by.

A card therefore carries no name of its own. There is one place a person types the question, and it is the field called the question.

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
```

The stencil above declares `Name`, so `Llama` is the value of `Name`, `{{Name}}` on a face lays it out, and the card has no `### Name` of its own. A stencil declaring `Question` first gives a deck whose headings are questions.

The first field is a heading, so it holds one line. A field wanting more than that is not the one to put first.

A value is HTML and holds what HTML holds: paragraphs, lists, a table, a picture. It cannot run — the window serves itself under a policy that allows no script it did not serve, and no handler written in an attribute. One deck carries cards of as many stencils as the person likes, and a card whose stencil declares a field the card leaves out is a card with that field empty.

### The heading carries the structure

A card's boundary is a heading and a field's name is a heading, so the structure of the file is the structure markdown already has. Headings are parts in the index, which makes a card searchable and addressable by `#` the day it is written, with nothing else built.

The consequence to accept is that a value cannot itself hold a second-level or third-level heading. A value that wants one uses a deeper level. **[ADR-0028](0028-a-card-is-named-by-what-it-holds.md) spends the first level too**, so a value's own heading is of the fourth level or below.

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

### A card is addressed by its deck and its heading

**Superseded by [ADR-0028](0028-a-card-is-named-by-what-it-holds.md).** A card carries a mark of its own at the end of its heading, and that is what it is addressed by. What follows is the decision as it stood.

A card has no identifier of its own. It is named by the note the deck is and the heading the card is, which is the value of its first field.

## Consequences

- A deck file grows long, and a person editing one by hand scrolls. The editor is where a deck is meant to be edited.
- A deck has a size a person can reach by writing, and reaching it means splitting a file the application asked them to keep as one.
- A field renamed in a stencil is a write to every deck that stencil cuts, so one edit to one file lands as many, and a deck that cannot be written keeps a heading nothing declares.
- A value cannot hold a heading of the two levels the format spends, and a person who writes one splits their card in half without being told.
- `type` and `fields` are ordinary English words taken as owned keys, and a person's own key of either name collides.
- `type` now means two things: what a note is, and what a link is for. The second is a key inside a link's own record and the first is a key about the whole note, so an indent tells them apart, and nothing else does.
- A field is a name in a list in a stencil and a heading in a deck. One concept, written two ways, and a reader has to learn both.
- The first field holds one line, so a stencil whose front is a picture puts the picture second and something short first.
- Reordering a stencil's fields moves which one names its cards, and every deck it cuts has to be rewritten for it.
- Two cards whose first field holds the same value are two cards of one name.
- Renaming a card's heading makes it a different card. Whatever was attached to the old name is attached to nothing, and that includes its schedule. A durable identity for a card is a decision the review makes when it has a reason to; until then a rename costs a card's history.
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

**A name of the card's own, beside its fields.** Rejected: a stencil of a question and an answer would have the question written twice, once in the heading and once in the field, and nothing keeps the two agreeing. Neither Anki nor Mochi gives a card a name outside its fields; Anki's first field is what a duplicate is judged on, and Mochi's template names one field as the card's title.

**A name of the card's own that follows the first field until a person types over it.** Rejected: it is the line above with a rule attached, and the rule is invisible — a person cannot see whether the name is still following or has been detached.

**The stencil naming any field, not necessarily the first.** Rejected: it is one more thing to declare and one more control to draw, for an ordering a person can already choose by putting that field first.

**One file per card.** Rejected: a deck of a thousand cards is a thousand files in a folder a person opens, and every one of them is a note in the index, in search, and in the plex.

**The schedule in the deck, beside the card it belongs to**, as org-fc and the Obsidian plugin do. Rejected: the file is then rewritten after every answer, so a review session is a diff over the whole deck, and what the person wrote and what the machine counted sit in one file with one modification time.
