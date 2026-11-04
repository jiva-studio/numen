# Flashcards

Running the cards a vault holds.

Flashcards is an application of its own, beside the editor and over the same core. Writing cards is occasional and reviewing them is daily, so reviewing is not done through the editor.

## Vault listing

The window opens with the vaults this installation has and the number of cards due in each. The list contains names and vault locations, which the registry returns before any vault is opened, so the whole list is shown at once.

Each vault's count arrives separately and is shown in its row. A vault whose count has not arrived shows a placeholder the size of the number, and cannot be opened until it has a count, by click or by its letter key. A vault counted recently keeps its count while the next count runs, so a list already shown is never cleared and refilled. A vault that could not be read shows the reason where its number would be.

Choosing a vault stops the counting, and so does closing the window: counts that have not arrived are not awaited.

## What is asked

A card is shown once for each template its card model declares ([cards](cards.md)), and each template asks a different question. **What is scheduled is a review card** — one card paired with one template of its card model — and not a card. A card model of two templates gives each of its cards two review cards, and one may be due while the other is a week away.

A card is identified by its card id, so it is the same card after it is moved to another section, another deck or another vault. A template is identified by its name, which is its heading in the card model, and renaming a template restarts that template's schedule.

### What is not asked

- A card whose wikilink names no card model, or names a note that is not one. Nothing can lay it out. It is reported as a problem on the deck, and it is fixed in the editor.
- A template missing a front or a back, which cannot be laid out.
- A deck that could not be read at all — over its bound, or frontmatter that does not parse. One unreadable file is not a reason to withhold the rest of a person's cards.

A card whose first field is empty is shown like any other. Its heading is empty, and the card is displayed using the template, not the heading.

## What is due

A card is due when the current day is the day of its schedule or later. **A card due today is due for the whole day**, whatever hour it is scheduled for, because a person can review at any hour.

A day begins four hours past midnight, and that is the Flashcards plugin's setting `day_start_time`, described in [Settings](settings.md). Reviewing cards at one in the morning finishes the day before, not starting the next, and a boundary at midnight would split one session in two. The hour uses local time, so the review day begins and ends according to the user's schedule.

A card that has no answers yet is due at its first review.

How many due cards a day includes — how many new cards, how many returning ones, and how long the day runs — is set by the preset the card's deck refers to. A deck with no preset uses the defaults — [Cards](cards.md).

## The order

A session runs over **the whole vault** by default, and a deck is a filter inside it. Cards due across the entire vault are reviewed together, regardless of which file each card was written in.

Cards that are due and have been answered before come first, the longest overdue first, because the most overdue card is the closest to being forgotten. Cards never answered come after them, in the order they appear in their decks.

## The four answers

A card is rated by how well it was recalled, not by whether it was:

|           |                                        |
| --------- | -------------------------------------- |
| **Again** | it was not recalled at all             |
| **Hard**  | it was recalled slowly and with effort |
| **Good**  | it was recalled                        |
| **Easy**  | it was recalled with no effort         |

The better a card was recalled, the longer the interval. FSRS computes the intervals, and none are written into a vault: they are calculated from the answers when needed.

### Undoing an answer

A person can press the wrong key. Undoing an answer is a separate line that names the answer it undoes, and both lines stay in the file. The card returns to its state before that answer, because its state is calculated from the answers that were not undone.

## Asking about a card

A person can ask the agent about the current card. Moving the card to the left opens a panel where that card can be asked about; control and `a` do the same, and pressing them again, or escape, closes the panel. The panel is available before the answer is shown and after. The agent answers from this vault — the card, the card model of the card, and the notes and sources the card links to — and names the source of each part of its answer. It can read everything and write only cards, so an answer that turns out to be wrong is corrected where a person found it.

A conversation belongs to one card and ends with it. What the panel can access, and what it cannot, is in [Agents](agents.md).

## Reading the sources of the cards

Moving the card the other way opens the notes the deck is linked to; control and `r` do the same, and close it again. It is available before and after the answer, as the conversation is. The letters are used with control because the bare letters are the answer keys.

It shows the vault's own graph: the notes the deck links to, which is every note its cards name, and the notes that link to the deck. They are listed one below another, each under its title, and space scrolls them. A link written in a card opens that note in this view.

The links belong to the deck and not to one card, so every card of a deck shows the same notes. A name that resolves to nothing is shown as it is written, with a note saying so; a name that matches several notes resolves to the nearest, with a note saying so. An attachment, a book and a note in another vault are left out, because this view cannot open them. The card model of a card is left out too: every card names one, and it defines how a card is laid out rather than what it was written from.

A deck linked to a great many notes has the text of the first thirty read and the rest listed by name, with a line giving the count.

## Where the answers are kept

```
<vault>/.numen/md.numen.views.flashcards.studio/<ulid>.jsonl
```

One session writes one file and nothing ever appends to it again. The whole history is those files combined, so a vault moved between a phone and a desktop has one history and one set of counts, and no file is ever written by two machines.

The answers belong to the vault and move with it. What is calculated from them — when each card is next due — is stored with the application, and deleting it loses only that calculated data.

## Card ids

A card typed by hand has no card id until the application writes its file ([cards](cards.md)), and an answer cannot be recorded for a card with no card id.

**A vault is marked when a review session begins**, and never during count calculation. Decks containing unmarked cards are written once to assign unique card identifiers, and are reloaded before review begins. Due cards are counted from what is already marked, so opening the window writes nothing: only the vault the person chooses is written to.

So a person who never opens the editor can still review their cards, and one write to a deck is the only change review makes in a vault. Nothing else about a deck is changed: not its text, not its order, and never a schedule.

## What is reported

| Problem                          | Against             | What it is                                                                                                                                              |
| -------------------------------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| a line that cannot be read       | the vault's answers | an interrupted run leaves a line without a closing newline, and a line of a version this build does not know is ignored. Both are counted and left out. |
| a deck that could not be written | the deck            | a card in it has no card id, and the file could not be written with one. Its cards are left out of this session and retried at the next.                |

Every other problem found during review is already a problem on a deck or a card model, and [cards](cards.md) lists them.
