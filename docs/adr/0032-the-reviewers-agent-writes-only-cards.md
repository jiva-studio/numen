# ADR-0032: The reviewer's agent writes only cards

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** `modules/libs/core` — `adapter/mcp`, `adapter/index`, `adapter/flashcardsui`, `container`; `modules/apps/desktop` — `internal/agents`, `cmd/numen-flashcards`
- **Related:** ADR-0002, ADR-0004, ADR-0020, ADR-0021, ADR-0022, ADR-0025, ADR-0030

## Context

A person is on a card and wants to know more about it. The review window could not be asked anything: it served what is due and what an answer is, and reached no agent.

The vault it is standing in holds the books that card was written from, and a passage link carries the byte range it came from. The one serious objection to a model in a reviewer is a confident wrong answer on material the person took from a book, and that vault is the answer to it.

A person who finds out mid-sitting that a card is wrong is holding the one piece of knowledge that fixes it, at the one moment they will never have again.

## Decision

### The tools are halved by what they do, and each half is named in full

`mcp.NewReading` serves `note_search`, `note_get`, `note_read`, `note_neighbourhood`, `link_list`, `source_list`, `source_read`, `card_stencils`, `card_read` and `vault_get`. Nothing else is on it.

`mcp.NewReviewing` is that surface and four more: `card_add`, `card_edit`, `card_remove` and `card_section_add`. It is what the window a person runs their cards in serves.

The tools themselves are the same tools, halved by what they do rather than written twice: every family registers its reading half and its writing half separately, and the full surface is both halves. So a tool gains a behaviour once and every surface has it.

### The reviewer writes a card and nothing else

A deck and a stencil are what a vault is arranged into, and nothing here makes one. No note, link or document is written from this window either. What a person is doing here is cards, and cards are what can be changed.

This amends ADR-0030, which said a card is read here and answered here and nothing else is done to it. Reading and answering are still all the window itself does; the change is that a person who spots a mistake can say so and have that one card corrected, instead of carrying it to the editor and losing it on the way.

### What holds each surface is a test that names it

A surface is a claim about what is absent, and a test that lists what is present passes with anything extra on it. All three surfaces are therefore asserted as exact sets, and a tool added to a family is a line added to a list (ADR-0025).

### The reviewer opens the index for reading, and the reading half of it is a type of its own

The index answers three more questions here than it did: the passages a search runs over, and the sources a book's text is read from. It is still opened for reading alone, and the handle that answers them cannot write: the read-only sources type is built on the query half and satisfies no repository port. Writing a card goes through the vault's files, not through the index.

A vault whose index has never been built answers these as it answers everything else — as a vault nothing has read yet.

### It answers by words alone

Nothing embeds behind this window, so a query has no vector and the meaning half of a search does not run. A question whose answer is in a book the person paraphrased may be missed here and found in the editor.

### The reviewer serves the tools and announces no port

The address and token an agent a person configured reads name one vault and one window. Two windows writing that file would point that agent at whichever started last, so the reviewer listens on an ephemeral loopback port with a token that lives in memory, writes neither file, and takes no address from the command line.

### The agent follows the vault the person sat down to

The front door counts every vault; a sitting is on one. The agent is told which vault it works when it is started, so it is started when a sitting opens and stopped when a sitting opens on another vault or the window closes. Going back to the decks leaves it standing.

### A conversation belongs to one card

Answering the card ends it. A thread carried across cards would answer the card in front of the person out of the one behind it.

### The panel is there on either side of the card

The swipe, the key and the control are offered whether the answer is showing or not. A person who cannot recall a card is often asking what the front even means, and refusing them until they turn it is refusing them the question they have.

### An answer says where it came from, in words

This window has one page and nothing to open a link with, so an answer names the file and the place in its own text.

## Consequences

- A third binary reaches the vault through tools, and what it may call is a list in one place.
- Two agents can be running on one machine, one to a window. Both read the whole vault; only the editor's writes notes.
- A sitting can now change the vault. It changes cards, which is what the sitting is about, and every other kind of file is as safe here as it was.
- Every tool surface has a list that must be edited when a tool is added, and a build that forgets fails.
- The reviewer finds less than the editor would on the same question, and says nothing about it.
- The read-only index grows a second type answering the same questions as the writing one, and only a test keeps the writing half out of it.

## Alternatives considered

**Giving the reviewer the surface the editor has.** Rejected: an application whose whole decision is that it reviews would be able to rearrange the vault it is reviewing from.

**Letting it read alone.** Rejected: the moment a person knows a card is wrong is the moment they are looking at it, and sending them elsewhere to fix it means it stays wrong.

**Leaving `note_search` and `source_read` off, so the read-only index need not grow.** Rejected: the objection this exists to answer is a confident wrong answer about a book the person owns, and those two tools are how the book is reached.

**Serving the tools on the editor's port and letting the reviewer borrow them.** Rejected: the reviewer would then need the editor running, which is the arrangement ADR-0030 exists to avoid.

**Keeping the conversation across the cards of a sitting.** Rejected: the context grows through a sitting and the card behind starts answering for the card in front.

**Offering the panel only once the answer is showing.** Rejected: it reads as a rule against looking things up, and the card a person is stuck on is the card they most need to ask about.

**Pre-generating an explanation while the front is up.** Rejected: it starts an agent on every card, including the ones a person knows, to save a wait on the few they do not.
