# Cards

A card is a set of named values a person wrote, laid out by a stencil into a front and a back. This is a specification, not a decision record: every rule here traces to an accepted ADR.

Three files are involved and all are ordinary notes. A **stencil** says what fields a card has and how they are shown. A **deck** holds the cards. A **preset** says how the decks pointing at it are scheduled.

## What a note is

The frontmatter key `type` says which of four a note is.

| `type` | What the file is |
| --- | --- |
| `note` | An ordinary note. This is the default, and nearly every note in a vault carries no `type` at all. |
| `stencil` | Fields and faces. |
| `deck` | Cards. |
| `preset` | How the decks pointing at it are scheduled. |

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

`fields` is a list of names, in the order a person is asked for them. A name is text; there are no field types, and a value is HTML.

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

Everything around a placeholder is HTML and is drawn as the markup it is.

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

A section is made at the end of the deck. Taking one away takes away its heading and nothing else: its cards stay where they stand, under whatever heading is above them now, and its own text stays with them.

A card is made at the end of the section it was asked for, and at the end of the cards standing before the first section where it was asked for none. A deck holding no section is a deck of one such run, so a card made in it stands last.

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

A value is HTML and holds what HTML holds: paragraphs, lists, a table, a picture, a rule. What it cannot hold is a heading at the first, second or third level written as a markdown heading, because those three levels are what the file is cut at. A value wanting a heading writes one as `<h4>` or below.

A card is not markdown. What stands under a field's heading is drawn as the markup it is, and a line of prose with no tag around it is a line of prose. It cannot run: the window serves itself under a policy that allows no script it did not serve, and no handler written in an attribute.

A card carrying a field its stencil does not declare keeps it in the file, and no face shows it. A field the stencil declares and the card omits is empty, and so is one whose heading is there with nothing under it.

Two fields of one name in one card are a problem against the deck. The first stands, and the second is kept in the file.

### The order of fields

On the screen a card's fields are in the stencil's order, whatever order the file wrote them in.

In the file they stay where they are. A card the editor did not touch arrives on the other side in the order it went in, and a field is moved only by a person moving it.

## The preset

A preset is a note of `type: preset`, and it says how the decks pointing at it are scheduled. Its settings are frontmatter keys; its body is the person's, written for themselves.

```markdown
---
type: preset
goal: minutes_a_day
minutes_a_day: 20
new_a_day: 8
reviews_a_day: 45
retention: 0.87
load:
  sat: 50
  sun: 0
even_load: true
---

# Sanskrit

Grammar and vocabulary. Three decks point here.
```

A deck names its preset with an entry of its `links:` block carrying `type: preset` — [Links](links.md).

```markdown
---
type: deck
links:
  - to: Sanskrit
    role: ref
    type: preset
---
```

A deck's tab reads which preset schedules it, and offers the defaults and every preset the vault holds. Choosing one repoints that entry; choosing the defaults removes it, and removes the `links:` key with it where the block held nothing else.

| | |
| --- | --- |
| `goal` | which value the one control steers: `minutes_a_day`, `retention` or `by_date`. The value stands under the key it names. |
| `by_date` | the day the material is to be learned by, written `2026-09-30`. What learned means is this preset's own `learned` rule, asked of every card on that day. |
| `minutes_a_day` | how long a day of review runs, spent against the time each answer took. Zero keeps no budget in time. |
| `new_a_day` | how many unseen cards a day holds. 10. |
| `reviews_a_day` | how many returning cards a day holds. 200. |
| `retention` | the share of cards recalled when they come round again. 0.90, and it goes from 0.70 to 0.99. |
| `learned` | what counts as a card learned: `interval`, or `retention`. The value stands under the key it names. `interval`. |
| `interval` | how long a card is sent away for before it is learned, in days. 21, and it goes from 1 to 365. |
| `counts` | what a day's budget is spent on: `cards`, where a card face counts once however often it comes round that day, or `shows`, where every showing spends a slot. `cards`. |
| `backlog` | how much of a day goes to what is overdue before anything new is offered, in per cent. 100 is the overdue pile first and new cards only once it is empty; 0 is new cards first; 50 splits the day between them. 100. |
| `load` | how much of a day's load each day of the week carries, in per cent, under `mon` to `sun`. A day not named carries 100, and a day at 0 schedules nothing. |
| `even_load` | whether days are made to resemble each other. On, and a goal of `by_date` moves no card whatever it holds. |

Several decks pointing at one preset is what sharing it looks like, and scheduling a deck differently is repointing one link. **A deck naming no preset is scheduled by the defaults**, and one naming a note that is not a preset is scheduled by the defaults with a problem against it.

Each preset's budget is spent on the cards of the decks pointing at it, and a sitting over the whole vault is the union of them. Inside one preset, the budget its `goal` names is what closes the day: the minutes under `minutes_a_day`, the two card counts under `retention`, and under `by_date` what has to be got through to have the material learned by that day. A setting the goal does not name keeps its value and takes no part until its own goal is chosen again.

**A preset's day is divided over the decks pointing at it**, and pressing one deck hands over that deck's share of it — the same count the deck stands at on the front door. The shares go by what each deck owes: three decks owing the same take a third each, and a deck owing nine times another's takes nine times the share. A share too small to buy a card, or larger than the deck has cards to spend it on, goes to the decks that can use it, so the day spends what it holds. What a deck was already sat through today comes off that deck's own share, so sitting one deck does not take from another and the decks may be sat in any order. Adding a deck adds nothing to the day's work; it spreads the same work over more decks, and raising the work is raising the budget ([ADR-0034](adr/0034-the-preset.md)).

**A day asks a card as often as it falls due in it.** An answer a card did not come back on sends it away for minutes, so it lands back inside the day it was asked in and that day asks it again. Under `counts: cards` a card face spends a slot the first time the day asks it and comes round again in it for nothing; under `counts: shows` every showing spends one. The minutes go on every showing either way, and the count a person reads beside the control is in cards: a card the day comes back to is the one card, and the showings it takes are what the clock runs out on.

**`counts` is read where a count closes the day, which is `retention` alone.** A goal of `minutes_a_day` is closed by the clock, and the clock is spent on every showing whichever way the preset counts. A goal of `by_date` is closed by the cards it has to begin, and every showing after a card face's first is a review, which that goal holds to nothing. So `counts: shows` shortens a day held to `retention` and changes nothing under the other two.

**A day names every budget that closed it.** Under `retention` the day is held to its new cards and to its reviews at once, and a day that hands over the whole of each has been closed by each. A person told only one of them raises that one and finds the day unchanged, so both are said.

`learned` is what the preset counts as a card learned, and the value it reads stands under the key it names. Under `interval` a card is learned once the interval it is sent away for reaches `interval` days; under `retention` it is learned once the chance of recalling it today is at or above `retention`. The rule not named keeps its value and takes no part until it is chosen again, and a preset naming no rule at all counts by `interval` at 21 days. Learned is a state and not a milestone: both rules are asked of where the card stands now, and a lapse takes a card back out of that standing under either of them. How many cards stand learned today is answered by the rule in force. How long until all of them are is a prediction made under a stated assumption: the soonest the material could be learned, read off the run in which nothing is forgotten. It is what the deck could reach and not a forecast of what will happen. It stands beside a goal of `minutes_a_day` and a goal of `retention`, and is absent under `by_date`, whose day is the answer already; and it is asked only under `learned: interval`, since a chance of recall is a level a deck stands at and names no such day. Over the range of a `retention` goal it falls away and steps up wherever another review is wanted — the day a card passes an interval is the reviews it takes times the space between them, and a review is a whole number. Every value of it is right where it is asked, and a step is that arithmetic and not a fault.

A goal of `by_date` aims at that same rule on the day it names, so the pace begins a card early enough to learn it by then: three weeks before the day under an interval of three weeks, and on the day itself under a chance of recall. Where a card cannot get there whatever the pace — a card added a fortnight before a day it must stand three weeks away from — the picture says how many card faces fall short. The day is not moved and the rule is not bent; the pace shown beside that number gets there every card that can. The day named is the answer, so no second day is offered beside it, and past that day the preset schedules nothing: what a projection shows there is a deck nobody is answering, with the debt climbing and what was learned fading.

`load` gives a day of the week its share of the load, and that share scales every budget the day keeps: a Saturday at 50 holds half the minutes and half of each card count. It is read whether or not the days are evened out, and **a day at 0 schedules nothing** — a pause of that one day.

`even_load` is whether a card is moved off the day it fell on. The day it comes back on is chosen inside the tolerance the scheduler allows around the interval, and a day carrying less of the load, or already holding more cards, is one the card is less likely to be put on. It is pressure and not a promise: no day is stopped from carrying more than its share, and a short interval leaves nowhere to move a card. With `even_load` off a card falls where the scheduler puts it, and a day that cannot show it leaves it standing over for the next one.

A preset whose goal is `by_date` moves no card, whatever `even_load` holds. The pace is what spreads that material over the days to the day named, and the days it has are the days it needs.

**No cards a day is a pause**: the preset schedules nothing, and every deck pointing at it stops. Pausing one deck is a preset of its own. A goal of a day ends the same way — past that day the preset schedules nothing until the day is moved or the deck is pointed elsewhere — and a preset aiming at a day and naming none is paused from the start, because the budget its goal names is the day.

**Why a preset schedules nothing is one answer, worked out where the cards are handed out.** The reasons are a closed list — no minutes a day, no cards a day, a date with no day, a date behind us, a week no day of which carries any of the load — and an interface asking is given one of them and puts it into words. A day of the week at none of the load stands apart from all five: it is a fact about that one day, and a preset with a quiet Sunday has not stopped. A week at nothing is not, because there is no next day for the cards to be picked up on.

The list says why a *preset* schedules nothing. A preset that schedules and has nothing it can schedule is a different thing and is said beside the material: a deck every card face of which nobody has begun, under a preset beginning none a day, is not stopped and never comes round either. The preset window says it in the control's place and the deck screen says it on the row.

The hour a day of review begins at is not here: it is a fact about a person's clock, and it is in [Settings](settings.md).

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
