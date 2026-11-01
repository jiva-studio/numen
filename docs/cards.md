# Cards

A card is a set of named values a person wrote. A card model lays them out as a front and a back.

Three kinds of file are involved, and all are ordinary notes. A **card model** defines the fields of a card and how they are shown. A **deck** contains the cards. A **preset** defines how the decks that point at it are scheduled.

## What a note is

The frontmatter key `type` sets which of four kinds a note is.

| `type`       | What the file is                                                                              |
| ------------ | --------------------------------------------------------------------------------------------- |
| `note`       | An ordinary note. This is the default, and nearly every note in a vault has no `type` at all. |
| `card-model` | Fields and templates.                                                                         |
| `deck`       | Cards.                                                                                        |
| `preset`     | How the decks that point at it are scheduled.                                                 |

The list is closed. A value outside it is reported as a problem against the note, and the note is read as an ordinary note.

## The card model

```markdown
---
type: card-model
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

<b>Height:</b> {{Height}}
<b>Weight:</b> {{Weight}}
<b>Life span:</b> {{Life span}}

## Name it

### Front

Which animal is {{Height}} at the shoulder and lives {{Life span}}?

### Back

{{Name}}
```

### Fields

`fields` is a list of names, in the order in which the person is prompted for them. A name is text; there are no field types, and a value is HTML.

A name is compared as written. Two fields with one name are a problem against the card model, and the first is used.

**A card's heading is read from the first field.** A card model that declares `Question` first gives a deck whose headings are questions. The question is typed once, under its own heading like every other field.

A card model that declares no field has no fields to fill in its cards. It is a problem against the card model.

A field contains whatever a person writes under it, over any number of lines. The first field is no exception: a field that needs a paragraph, a list or a picture may be at any position in the order, and the heading takes one line of it.

### The first field stays first

The editor does not move the first field, does not move another field above it, and does not remove it. Each of the three makes a different field the source of every card's heading, and every heading in every deck that uses the card model is rewritten from a field it was not written for.

The fields below the first can be reordered freely. Their order is the order of the prompts and nothing else, and moving them writes to no deck.

Nothing prevents a person who edits `fields` by hand from moving the first field. Nothing is lost from the file: every value is under its own heading, and from the next write of a deck on, its cards get their headings from the field that is now first.

### Renaming a field

A field renamed in a card model is renamed in every card that uses the card model. Each deck of the vault is read, the heading is rewritten in the cards that use the card model, and the value under it is not changed. The rename applies to decks that are not open, so it is one edit to the card model and a write to every deck that has cards of it.

A deck that cannot be written (its frontmatter does not parse, or the filesystem refuses the write) keeps the old heading, and that is a problem against the deck.

A rename is the application changing the content of a file, so a deck it changes gets an `id` if it had none. One rename can therefore write an identifier into a deck a person wrote by hand and never opened here.

### Templates

Each second-level heading is one template, and the heading is the name of the template. Under it, `### Front` and `### Back` contain what is shown before and after the answer. A template missing either is a problem, and it produces no output.

A side ends at the next `### Front`, the next `### Back` or the next template. A third-level heading with any other name is part of the side it is written under, and a second `### Front` or `### Back` is text under the first.

A card model has as many templates as a person writes. Each template shows each card once.

### Placeholders

`{{Field}}` marks where a value is inserted, and the name inside is the exact name of a field. The first field is inserted the same way, by its own name, and the whole value under that field's heading is inserted, not the one line of it used as the card's heading.

A placeholder naming a field the card model does not declare is a problem against the card model. A placeholder whose card has an empty value for that field produces no output.

Everything around a placeholder is HTML and is rendered as HTML.

`{{` cannot be escaped, and a code span does not exempt it: wherever the two characters appear in a template with no brace between them and the next `}}`, the text between the pairs is read as a placeholder. A template that needs those characters as text can only use a brace in the middle to prevent the match. This is a known limit of the format.

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

Each first-level heading starts a section, and the cards under it belong to it until the next first-level heading. A section is a name and nothing else: it has no fields, no card model, no schedule and no card id.

A deck need not have sections, and cards may be placed before the first one. Two sections may have one name, and a section with no card is kept.

Text between a section's heading and its first card belongs to the section and is kept unchanged. It is the section-level equivalent of the text a card has between its card model's wikilink and its first field. Nothing renders it and nothing reads it.

A section is created at the end of the deck. Deleting a section deletes its heading and nothing else: its cards stay in place, under whichever heading is above them then, and its own text stays with them.

A card is created at the end of the section it is created in. When it is created in no section, it is added after the cards that precede the first section. A deck with no section is a deck of one such run, so a card created in it is last.

### Cards

Each second-level heading is one card. A card ends at the next second-level heading, at the next first-level heading, or at the end of the file.

A card's heading is the first line of its first field, truncated to one line. It stores no data of its own: if it is deleted, it is written again from the field. The heading ends at the first line break or at 120 characters, counted as characters a person sees, and it never ends inside a `[[wikilink]]`, an embed or a run of emphasis. When the limit falls inside one of these, the heading ends before it.

A card whose first field is empty or contains only spaces has an empty heading. The card is read and shown like any other, and its first field is shown as an empty input.

Two cards may have the same heading, as when two cards begin alike. Their card ids distinguish them.

A person editing the file by hand may leave a heading and its first field inconsistent. The field is used. The next write of that deck rewrites the heading from the field, and nothing is reported, because the person has nothing to decide. A card whose card model cannot be read is the exception: the first field is unknown, so the heading is left unchanged.

### The card id

A card's heading ends with `^` and ten characters of `0123456789abcdefghjkmnpqrstvwxyz`. This is the identifier of the card for as long as it exists.

```markdown
## Compost, what is it made of ^k7m2xq9fzp
```

A card id is separated from the heading's text by one space and is last on the line. It is read as a card id only with that length and alphabet. Anything else at the end of a heading is heading text, and a malformed mark is not a card id: the card is given a new one.

A card id is written when the card is created. A card typed into a deck by hand has none until the application next writes that file, which gives it one.

The card id does not depend on the deck. A card moved to another deck, or to another vault, is the same card, with the same card id and the same history. Nothing is recomputed and nothing is lost.

Two cards with one card id in one deck are a problem against that deck.

### The preamble and the tail

Text above the first heading of the body is the deck's preamble. It is kept verbatim and is not a card.

A value is read without the blank lines at either end of it, so the bytes that separate one field from the next belong to neither. The text that remains at the end of the file after the last value is the tail, and it is kept verbatim the same way. A deck with no heading in its body is a deck of no cards, and its whole body is preamble.

### The card model a card is cut by

A paragraph that contains only a `[[wikilink]]`, directly below the card's heading, names the card model. It is an ordinary link and is resolved like any other link ([Links](links.md)).

A card whose first paragraph is not a lone wikilink has no card model, and neither has a card whose wikilink resolves to a note that is not a card model. Both are problems against the deck. The values are read in both cases, and nothing is lost from the file.

Text between that wikilink and the card's first field is kept verbatim and is not the value of any field. Nothing renders it.

One deck can contain cards of any number of card models.

### Values

Each third-level heading is a field, and its value is everything under it until the next heading of the first, second or third level. Every field of the card model is written this way, including the first.

A value is HTML and can contain paragraphs, lists, a table, a picture and a rule. It cannot contain a markdown heading at the first, second or third level, because the file is split at those three levels. A value that needs a heading uses `<h4>` or below.

A card is not markdown. What is under a field's heading is rendered as HTML, and a line of text with no tag around it is rendered as text. A card cannot run scripts: the window sets a policy that allows only scripts the window serves itself and no event handler written in an attribute.

A card that has a field its card model does not declare keeps it in the file, and no template shows it. A field that the card model declares and the card omits is empty, and so is a field whose heading has nothing under it.

Two fields with one name in one card are a problem against the deck. The first is used, and the second is kept in the file.

### The order of fields

On the screen a card's fields are in the card model's order, whatever order they have in the file.

In the file they keep their order. A card the editor did not change is written back in the order it was read in, and a field is moved only when a person moves it.

## The preset

A preset is a note of `type: preset`, and it defines how the decks that point at it are scheduled. Its settings are frontmatter keys. Its body is free text for the person.

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

A deck's tab shows which preset schedules it, and lists the defaults and every preset in the vault. Choosing a preset changes the target of that entry. Choosing the defaults removes the entry, and also removes the `links:` key if the block contained nothing else.

|                 |                                                                                                                                                                                                                        |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `goal`          | which value the one control sets: `minutes_a_day`, `retention` or `by_date`. The value is under the key it names.                                                                                                      |
| `by_date`       | the day the material is to be learned by, written `2026-09-30`. "Learned" is defined by this preset's `learned` rule, applied to every card on that day.                                                               |
| `minutes_a_day` | how long a day of review runs, counted against the time each answer took. Zero under this goal is a pause. Under the other two goals the value is not read.                                                            |
| `new_a_day`     | how many unseen cards a day contains. 10.                                                                                                                                                                              |
| `reviews_a_day` | how many returning cards a day contains. 200.                                                                                                                                                                          |
| `retention`     | the share of cards recalled when they are shown again. 0.90, in the range 0.70 to 0.99.                                                                                                                                |
| `learned`       | what counts as a card learned: `interval`, or `retention`. The value is under the key it names. `interval`.                                                                                                            |
| `interval`      | the interval, in days, that a card must reach to count as learned. 21, in the range 1 to 365.                                                                                                                          |
| `counts`        | what a day's budget is spent on: `cards`, where a review card counts once however often it is shown that day, or `shows`, where every showing spends a slot. `cards`.                                                  |
| `backlog`       | how much of a day goes to overdue cards before any new card is offered, in per cent. 100 means overdue cards first and new cards only once none is overdue; 0 is new cards first; 50 splits the day between them. 100. |
| `load`          | how much of the load each day of the week takes, in per cent, under `mon` to `sun`. A day not named takes 100, and a day at 0 schedules nothing.                                                                       |
| `even_load`     | whether the scheduler evens out the load between days. With a goal of `by_date`, no card is moved whatever this holds.                                                                                                 |

Several decks that point at one preset share it, and scheduling a deck differently means changing the target of one link. **A deck naming no preset is scheduled by the defaults**, and a deck that names a note that is not a preset is scheduled by the defaults, with a problem reported against the deck.

Each preset's budget is spent on the cards of the decks that point at it, and a session over the whole vault is the union of them. Inside one preset, the budget named by its `goal` ends the day: the minutes under `minutes_a_day`, the two card counts under `retention`, and under `by_date` the number of cards that must be done to have the material learned by that day. A setting the goal does not name keeps its value and is not used until its own goal is chosen again.

**A preset's day is divided over the decks that point at it**, and selecting one deck allocates that deck's share of it, which is the same count the deck shows in its listing. The shares are proportional to the number of cards due in each deck: three decks with the same count due get a third each, and a deck with nine times another's count due gets nine times the share. A share too small for one card, or larger than the number of cards the deck can use, is given to the decks that can use it, so the whole budget of the day is spent. What a deck has already done today is subtracted from that deck's own share, so a session on one deck does not reduce another deck's share and the decks may be studied in any order. Adding a deck does not add work to the day; it spreads the same work over more decks. To increase the work, increase the budget.

**A day shows a card as often as it falls due in that day.** An answer that does not pass reschedules the card for a few minutes later, so it falls due again inside the same day and the day shows it again. Under `counts: cards` a review card spends a slot the first time the day shows it and costs nothing when it is shown again that day. Under `counts: shows` every showing spends a slot. The minutes are spent on every showing either way, and the count next to the control is in cards: a card shown again is still one card, and its showings are what use up the time.

**`counts` is read only where a count ends the day, which is the `retention` goal.** A goal of `minutes_a_day` ends the day by the clock, and the clock is spent on every showing whichever way the preset counts. A goal of `by_date` ends the day by the cards it has to start, and every showing after a review card's first is a review, which that goal sets to zero. So `counts: shows` shortens a day limited by `retention` and changes nothing under the other two.

**A day reports every budget that ended it.** Under `retention` the day is limited by both its new cards and its reviews, and a day that completes all of each was ended by each. If only one were reported, a person might increase it and find the day unchanged, so both are reported.

`learned` defines when the preset counts a card as learned, and the value it reads is under the key it names. Under `interval` a card is learned once its scheduled interval reaches `interval` days. Under `retention` it is learned once the probability of recalling it today is at or above `retention`. The rule not named keeps its value and is not used until it is chosen again, and a preset that names no rule counts by `interval` at 21 days. Learned is a state and not a milestone: both rules are evaluated on the current state of the card, and a lapse makes a card not learned under either rule. The number of cards that are learned today is computed by the rule in force. The time until all cards are learned is a prediction made under a stated assumption: the soonest the material could be learned, computed from the run in which nothing is forgotten. It is what the deck could reach and not a forecast of what will happen. It is shown for a goal of `minutes_a_day` and a goal of `retention`, and is absent under `by_date`, whose day is already the answer. It is computed only under `learned: interval`, because a probability of recall is a level a deck is at and does not define such a day. Over the range of a `retention` goal it decreases and jumps up wherever another review is needed: the day a card passes an interval equals the number of reviews it takes times the space between them, and the number of reviews is a whole number. Every value is correct for its input, and a jump is a result of that arithmetic and not a fault.

A goal of `by_date` applies the same rule on the day it names, so the pace starts a card early enough for it to be learned by then: three weeks before the day under an interval of three weeks, and on the day itself under a probability of recall. When a card cannot be learned by the day whatever the pace (for example, a card added two weeks before a day by which it must have a three-week interval), the chart shows how many review cards fall short. The day is not moved and the rule is not relaxed. The pace shown next to that number learns every card that can be learned. The day named is the answer, so no second day is shown, and after that day the preset schedules nothing. A projection past that day shows a deck that is not being studied, with the overdue count rising and the learned count falling.

`load` sets the share of the load for a day of the week, and that share scales every budget of the day: a Saturday at 50 has half the minutes and half of each card count. It is used whether or not the load is evened out, and **a day at 0 schedules nothing**, which pauses that one day.

`even_load` sets whether a card is moved off the day it falls due. The day it is shown again is chosen within the tolerance the scheduler allows around the interval. A card is less likely to be placed on a day with a smaller share of the load or with more cards already. This is a preference and not a guarantee: no day is prevented from taking more than its share, and a short interval leaves no room to move a card. With `even_load` off, a card is placed on the day the scheduler chooses, and a day that cannot show it leaves it for the next day.

A preset whose goal is `by_date` moves no card, whatever `even_load` is set to. The pace spreads the material over the days up to the day named, and the days it has are the days it needs.

**No cards a day is a pause**: the preset schedules nothing, and every deck that points at it stops. Pausing one deck requires a preset of its own. A goal of a date ends the same way: after that date the preset schedules nothing until the date is changed or the deck uses another preset. A preset with the goal `by_date` and no date is paused from the start, because the budget its goal names is the date.

**The reason a preset schedules nothing is one value, computed where the cards are selected.** The reasons are a closed list: no minutes a day, no cards a day, a date with no day, a date in the past, and a week in which no day has any of the load. An interface that asks for the reason gets one of them and turns it into text. A day of the week with none of the load is not one of these five: it is a fact about that one day, and a preset with a quiet Sunday has not stopped. A week with no load is on the list, because there is no next day on which the cards can be scheduled.

The list gives the reason a _preset_ schedules nothing. A preset that schedules but has nothing it can schedule is a different case and is shown next to the material. Example: a deck whose review cards have not been started, under a preset that starts no cards a day, is not stopped, but it also never schedules anything. The preset window shows this in place of the control, and the deck screen shows it on the deck's row.

The hour at which a day of review begins is not set here. It depends on the person's clock and is set in [Settings](settings.md).

## Size

A note is read up to a megabyte ([Editing](editing.md)). A deck is a file of many cards, so it has a bound of its own: eight megabytes. A card model is a note and has the note's.

A deck over its bound is not read, and the interface reports it: it shows which file it was and what the bound is, and the deck must be opened elsewhere or split. The size is checked before the file is opened, so a deck over the bound is rejected with none of its bytes read.

## What is reported

A problem is something that could not be processed and was not guessed. It is reported against the note a person would open to fix it. For everything below, that note is the card model or the deck that contains the text, and never the other file.

| Problem                            | Against        | What it is                                                                                                            |
| ---------------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------- |
| a `type` outside the list          | the note       | `type` is not `note`, `deck`, `card-model` or `preset`. The note is read as an ordinary note.                         |
| two fields of one name             | the card model | `fields` declares a name twice. The first is used.                                                                    |
| a card model declaring no field    | the card model | `fields` is absent or empty, so a card of this card model has no fields to fill.                                      |
| a template missing a side          | the card model | no `### Front` or no `### Back` under a second-level heading. The template produces no output.                        |
| a placeholder that is not declared | the card model | `{{Field}}` naming a field outside `fields`. The rest of the template is read as usual.                               |
| a card with no card model          | the deck       | the first paragraph under the card's heading is not a lone wikilink. The values are read.                             |
| a card model that is not one       | the deck       | the wikilink resolves to a note whose `type` is not `card-model`. The values are read.                                |
| two cards of one card id           | the deck       | two cards of one deck have the same `^` and ten characters. Both are read and shown.                                  |
| two fields of one name             | the deck       | one card has the same field heading twice. The first is used.                                                         |
| a deck that could not be written   | the deck       | a field renamed in a card model did not reach this deck, so a card holds a heading the card model no longer declares. |
| a deck over its bound              | the deck       | eight megabytes, and the file was not read.                                                                           |

The card model a card names is an ordinary link, so a wikilink that resolves to no note or to two notes is `dangling` or `ambiguous` as anywhere else ([Links](links.md)).

A card that has a field its card model does not declare is not a problem. A value that contains a heading of a level the format uses is not a problem either: the file is split at that heading. A card with no card id is not a problem, because it is given one. A heading that differs from its first field is not a problem, because it is written again from the field.

## What is not in these files

**No schedule.** When a card is due, its current interval and how it has been answered are computed from the history of answers. They are a cache and are stored in the index. A deck is not rewritten when a card is reviewed.

## What is searched

A deck and a card model are not split into chunks, and nothing in them is embedded. Neither the full-text search nor the semantic search searches inside a card. A card is found by its heading, which is its question, and a card model by its title, which is the name of its file.

Of a deck's headings only the ones a person wrote are indexed: its sections and its cards. The third level is the card model's field names repeated under every card, and it is not indexed. A card model has no indexed heading. A card's heading is indexed without its card id, which exists for the file and not for a person reading a list.

In every other way a deck and a card model are notes: they have a title, a type and an identifier, their links are resolved and their backlinks are returned, and each is a node in the plex with its headings as child nodes.

## Handling of existing files

A card model and a deck are notes, so everything in [Note format](note-format.md) holds for them: unknown frontmatter keys are preserved verbatim, line endings are preserved per file, a file that fails to parse is reported and not rewritten, and the application does not rewrite what it did not change.

A deck edited in the editor keeps the order of the file: the sections, the cards inside them, and the fields inside those are written back in their original positions. Every part of the file the editor did not change is written back as the same bytes it was read as: the preamble, the tail, the text under a card's wikilink, and a field no card model declares.

A deck is normalized where it is written, before its bytes are stored in the vault: a card with no card id is given one, and a heading that differs from its first field is written again from it. A window and the tools an agent uses therefore produce the same file.
