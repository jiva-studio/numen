# ADR-0033: What the deck is joined to is read beside the card

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** `modules/libs/core` — `usecase/flashcards`, `adapter/flashcardsui`, `refusal`; `modules/apps/desktop` — `cmd/numen-flashcards`, `flashcards`
- **Related:** ADR-0018, ADR-0025, ADR-0027, ADR-0030, ADR-0032

## Context

A card was written from something. While it is being answered, that something is out of reach: a link inside a card reaches the screen and the window swallows the press on it, because there was one page here and no way back to it.

The vault already knows what is joined to what. A link written in a card's value is a link in the file the card stands in, and the index resolved it when it read that file.

## Decision

### The links are the graph's, and nothing is parsed again

`markdown.Parse` puts a note's body links on it whatever type the note is, so a deck carries every link its cards write. What the panel shows is `note.ShowLinks` over the deck's path — one query, already written, over an index this window already opens.

Nothing pulls `[[wikilinks]]` out of a card's values, and no deck is read again to find them.

### They belong to the deck, not to the card

A link is written in a file, and the file is the deck. There are no links to a card, so the panel shows the same notes on every card of one deck.

The alternative is to keep only the links whose text falls inside the card in front of the person. That is a second resolution of what the index already resolved, to narrow a list a person is reading by choice.

### Both directions

What the deck points at, and what points at the deck. A note with nothing pointing out of it can still be the middle of a vault, and looking at one half of an edge is how a graph gets misread.

### Notes, and only notes

An address may name an attachment or a book, and this window has one page and nothing to open either with. Such a link is left out rather than drawn as a note that will not open, and what decides is the address itself: only a name and an identifier can name a note, so only those two are kept. An address of any other kind is not a note that has gone missing, and drawing it as one says a vault has a question in it where it has none.

**A stencil is left out too.** Every card names the stencil it is cut by, so a deck points at all of its stencils. A stencil is how a card is laid out, not what it was written from, and it would otherwise stand at the head of the reading of every deck.

A link into another vault is left out for the same reason — this window has one vault open, and there is nothing here to read the other with.

### A link that resolves to nothing is still shown

It is named by how it is written and carries no text. A vault where a name has come loose is a vault with a question in it, and hiding the question answers it wrongly. A name several notes answer to is shown resolved to the nearest and said to be ambiguous, because a person reading the wrong note has no other way to find out.

### The tail is names alone

A hub note brings hundreds of backlinks, and a note may be a megabyte. The text of the first thirty is read and the rest come back named, with a line saying how many. A panel that reads a vault into memory to fill a strip nobody scrolled is not reading, it is copying.

### The strip is three, and it rests in the middle

Reading on one side of the card, the conversation on the other, and one movement reaching both. The card is where the strip rests, so it is put there without animation when the window opens and whenever it is resized.

Which of the three is in the window is one thing in one place. Two panels each holding whether they are open would sooner or later both be open, and the strip is at one stop.

### The way in stands on either side of the card

The swipe, the key and the control are offered whether the answer is showing or not, as the way into the conversation already is. What a person looks at before answering is theirs to decide.

Both panels are held with the overlay key — control, or command on a Mac — because the letters on their own are what a card is answered by. The key that brings a panel in takes it away again, and so does the control beside it.

### Space means something else while the reading is in the window

It scrolls the reading rather than turning the card. It is the one key in this window that means two things, and it is the one a person's hand is already on; the answer is still shown by the control that stands under both panes.

## Consequences

- The review window reads notes, not only decks. It still writes nothing but a mark and an answer.
- The list is the same on every card of a deck, and long decks have long lists.
- A person who wants to see the stencil a card is cut by opens it in the editor, as they would to change it.
- `refusalOf` leaves the editor's adapter for `libs/core/refusal`, which both adapters can see: two adapters answering the same question needed one vocabulary, not two. It cannot live under `adapter/`, where an adapter may not reach another adapter.
- A person can read what a card came from without leaving the sitting, and without the editor being open.

## Alternatives considered

**Narrowing the list to the card in front of the person.** Rejected: it re-resolves what the index resolved, to shorten a list a person opened on purpose.

**Showing books and passages too.** Rejected here, not forever: the page reader is a route of the editor's own mux, and reaching it from this window is a piece of work of its own.

**Opening what a link names in a place of its own, and coming back.** Rejected: the sitting is the thing a person must not lose, and a pane beside the card never takes them out of it.

**Reading every joined note's text.** Rejected: a hub note makes that unbounded, and the bound has to be somewhere a person can see.
