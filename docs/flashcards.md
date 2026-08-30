# Flashcards

Running the cards a vault holds. This is a specification, not a decision record: every rule here traces to an accepted ADR.

Flashcards is an application of its own, beside the editor and over the same core ([ADR-0030](adr/0030-review-is-an-application-of-its-own.md)). Writing cards is occasional and running them is daily, so the daily one is not reached through the other.

## What is asked

A card is shown once through each face its stencil declares ([cards](cards.md)), and each face asks a different thing. **What is scheduled is a card face** — one card, and one face of the stencil that cuts it — and not a card. A stencil of two faces gives each of its cards two paths, and one may be due while the other is a week away.

A card is asked under the mark it carries, so it is the same card after it is moved to another section, another deck or another vault. A face is asked under its name, which is its heading in the stencil, and renaming a face starts that face's schedule again.

### What is not asked

- A card whose wikilink names no stencil, or names a note that is not one. Nothing can lay it out. It is a problem against the deck, and the editor is where it is settled.
- A face missing a front or a back, which lays out nothing.
- A deck that could not be read at all — over its bound, or frontmatter that does not parse. One unreadable file is not a reason to withhold the rest of a person's cards.

A card whose first field is empty is asked like any other. Its heading is empty, and what it is shown through is the face, not the heading.

## What is due

A card is owed when the day holding now reaches the day its schedule falls on. **A card owed today is owed for the whole of it**, whatever hour it falls at, because a person sits down when they sit down.

A day begins four hours past midnight, and that is a setting. A person answering cards at one in the morning is finishing the day before, not starting the next, and a boundary at midnight would split one sitting in two. The hour is an hour on the clock on the wall, so the day an hour is put into or taken out of begins and ends where a person reads it.

A card nobody has answered is owed the first time it is asked about.

## The order

A session runs over **the whole vault** by default, and a deck is a filter inside it. What a person owes is what they owe, and which file a card was written in is not something they think about while answering.

Cards owed and answered before come first, the one waiting longest at the front, because a card left late is the one closest to being forgotten. Cards nobody has answered come after them, in the order they stand in their decks.

## The four answers

A card is answered by how well it came back, not by whether it did:

| | |
| --- | --- |
| **Again** | it did not come back at all |
| **Hard** | it came back, slowly and with effort |
| **Good** | it came back |
| **Easy** | it came back with none |

The better it came back, the longer it is left. What that comes to is FSRS's ([ADR-0031](adr/0031-an-answer-is-an-artifact-a-schedule-is-a-cache.md)), and nothing about the intervals is written into a vault: they are worked out from the answers whenever they are wanted.

### Taking one back

A person hits the wrong key. Taking an answer back is a line of its own, naming the answer it takes back, and both lines stay in the file. The card returns to where it stood before that answer, because where it stands is worked out from the answers that were not taken back.

## Asking about a card

A person is on a card and wants to know more about it. Taking the card to the left brings in a panel where that card can be asked about; `a` does the same, and escape sends the panel away. The panel is there on either side of the card, before the answer is showing and after. The agent answers from this vault — the card, the stencil that cuts it, and the notes and sources the card points at — and names where each part of its answer came from. It reads everything and writes cards alone, so an answer that turns out to be wrong is corrected where a person found it.

A conversation belongs to one card and ends with it. What the panel can reach, and what it cannot, is in [Agents](agents.md).

## Where the answers are kept

```
<vault>/.numen/flashcards/<ulid>.jsonl
```

One sitting writes one file and nothing ever appends to it again. The whole history is those files put together, so a vault carried between a phone and a desktop comes away with one history and one set of counts, and no file is ever written by two machines.

The answers are the vault's and travel with it. What is worked out from them — when each card comes round — is kept with the application, and deleting it costs a person the working out and nothing else.

## Marks

A card typed by hand carries no mark until the application writes its file ([cards](cards.md)), and a card with no mark has nothing an answer could be recorded against.

**A vault is marked when a person sits down to it**, and never for the counting. Its decks holding such a card are written once, which mints a mark for every card in them, and they are read again before anything is asked. What a person owes is counted from what is already marked, so opening the window writes to nothing: the vault they choose is the vault that is written to.

So a person who never opens the editor can still run their cards, and one write to a deck is the whole of what review changes in a vault. Nothing else about a deck is touched: not its text, not its order, and never a schedule.

## What is reported

| Problem | Against | What it is |
| --- | --- | --- |
| a line that cannot be read | the vault's answers | a run that stopped partway leaves a line no newline closed, and a line of a version this build does not know is one it will not act on. Both are counted and left out. |
| a deck that could not be written | the deck | a card in it carries no mark, and the file could not be given one. Its cards are left out of this session and asked for again at the next. |

Everything else a review meets is already a problem against a deck or a stencil, and [cards](cards.md) is where it is listed.
