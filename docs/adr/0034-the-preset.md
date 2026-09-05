# A preset is a note, and one arithmetic schedules it

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** [The application writes to the vault](0017-the-application-writes-to-the-vault.md), [The note file](0018-the-note-file.md), [The stencil, the deck and the card](0027-the-stencil-and-the-deck.md), [An answer is an artifact, a schedule is a cache](0031-an-answer-is-an-artifact-a-schedule-is-a-cache.md)

## Context

How much is studied in a day, how long a day runs, how much is asked of memory: none of it is settled anywhere a person can reach. `numen.json` holds six appearance keys and one naming key, and everything about review is a constant in the source.

`numen.json` belongs to the installation — one machine, every vault. That is right for the window and for names. It is wrong for how a subject is studied: two vaults are two appetites, and one vault holds subjects that want different daily loads.

## Decision

### A preset is a note

A preset is a note of `type: preset`, which is the fourth of the closed list of note types.

Its settings are frontmatter keys. The body is the person's: what the preset is for, written for themselves.

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

Keys are the application's, in the spelling `numen.json` already uses. A key the application does not own is left where it stands, as every other note's frontmatter is.

### A deck points at a preset

A deck names its preset with an entry of its `links:` block carrying `type: preset`. `type` is an open vocabulary in that block, so nothing about the note format changes.

Several decks pointing at one preset is what sharing it looks like. Scheduling a deck differently is repointing one link, and the deck's own tab is where a person picks the preset from the ones the vault holds. **A deck naming no preset is scheduled by the defaults**, and a deck naming a note that is not a preset is a problem against the deck.

### What a preset settles, and what it does not

Everything about how a deck is scheduled is the preset's: how many new cards and how many reviews a day, whether a budget is spent on a card or on a showing, how long a day runs, the retention target, what counts as learned, how much of a day goes to what is overdue, the share of the load each day of the week carries, whether a card is moved off the day it fell on, and the goal that names which one of them closes the day. Each key, its default and what it does is [Cards](../cards.md).

`numen.json` keeps the hour a day begins at. It is a fact about a person's clock rather than about a subject.

### The scope of a budget is the preset, and it is divided before it is spent

A preset's budget is spent on the cards of the decks pointing at it, and a sitting over the whole vault is the union of them. Ten minutes on one preset and twenty on another is thirty minutes.

**Inside one preset, one budget closes the day**, and the preset's `goal` names which. A day is one day whatever presets fall on it, so how loaded each day is, is one table.

**A preset's day is divided over the decks it schedules, once, before any of it is spent.** A sitting over one deck is that deck's slice of the one division, and the deck's row on the front door is the same slice: they are one number because they are one arithmetic. The shares are proportional to what each deck owes, what the proportions leave over goes by the largest fraction, and where the fractions stand equal the deck owing more takes it, so a vault divides the same day however its files are named or walked. Decks owing the same are alike in everything the division knows of them, and take what is left over in the order their paths stand. What no deck can use is offered round again.

### One function places a card's day, and everything that asks goes through it

A scheduler works out an interval and leaves a tolerance around it: several days would do. Which of them a card is asked for on is chosen in one function, by weight, where a day's weight is the share of the load its day of the week carries over what already falls on it.

**The sitting that hands the cards out and the picture drawn beside a control both go through that function.** A day chosen once for the picture and again for the sitting is two arithmetics kept in step by tests, and they drift the moment one is touched: the control draws an even load the scheduler never delivers.

It is pressure and not a promise. No day is forbidden to carry more than its share, nothing is solved over the collection, and a card with nowhere to go stands where it fell.

## Consequences

- **A preset travels with the vault.** It is text the person wrote, in their history, syncing with everything else.
- **There is no vault-wide scheduling scope.** The scopes are the installation, a preset, and the deck that points at one.
- **A number is enforced where it was typed.** Decks are files and nothing contains anything, so no limit is displaced onto a parent.
- **The application writes to a note it did not create.** Settings written into the vault go down the application's one write path, and a preset is the first note the application edits key by key rather than whole.
- **A card is scheduled at its own preset's target.** The answers are replayed under the scheduler of the preset the card's deck points at, and the schedule cache carries a mark of which cards stood under which target. A mark that does not match is a cache thrown away whole.
- **The settings a person can change stand in two files.** `numen.json` is the installation's and is written down in [Settings](../settings.md); a preset is the vault's and is written down in [Cards](../cards.md), beside the deck it schedules.
- **A curve a person reads is a promise the sitting keeps.** Both are the same arithmetic over the same day.
- **A day is a whole number**, counted from the day the clock is counted from, so the same answers name the same days in every process.
- **A deck's row is a promise the sitting keeps**, by construction and not by two arithmetics agreeing.

## Alternatives considered

**Every budget binding at once.** Rejected: a day would end at whichever number ran out first, and which one that was is not what a person is changing.

**A day chosen once for the picture and again for the sitting.** Rejected: the two are kept in step by tests, and drift the moment one of them is touched.

**Equal shares over the decks.** Rejected: a deck of five cards beside one of five hundred would hold half the day and spend a tenth of it.

**A budget of its own for each deck.** Rejected: the preset would stop being the scope of a day, and adding a deck would add to the day's work.

**Scheduling settings in `numen.json`.** Rejected: the file is one installation's, and a person studying two subjects on one machine has one set of limits for both.

**A scheduling scope inside the vault, one for all its decks.** Rejected: it is the same fault one level down, and a preset already gives a vault as many as it wants.

**Settings written in the preset's body**, as a deck's cards and a stencil's faces are. Rejected: the body is where a value may hold a heading or a paragraph. A limit is a scalar, and frontmatter is where this format keeps scalars.

**Keys in the language of the interface.** Rejected: the vault format is one spelling, and `type`, `fields` and `links` are already written in it.

**A deck carrying its own limits.** Rejected: two decks that should agree would have no way to say so, and the settings would be copied by hand into every deck of a subject.
