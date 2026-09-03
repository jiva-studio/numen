# ADR-0034: A preset is a note, and a deck points at one

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format — every application that reads or writes one
- **Amends:** ADR-0027 (a fourth value of `type`)
- **Related:** ADR-0017, ADR-0018, ADR-0026, ADR-0027, ADR-0031, ADR-0036, ADR-0037, ADR-0038, ADR-0039, ADR-0040

## Context

How much is studied in a day, how long a day runs, how much is asked of memory: none of it is settled anywhere a person can reach. `numen.json` holds six appearance keys and one naming key, and everything about review is a constant in the source.

`numen.json` belongs to the installation — one machine, every vault. That is right for the window and for names. It is wrong for how a subject is studied: two vaults are two appetites, and one vault holds subjects that want different daily loads.

## Decision

### A preset is a note

A preset is a note of `type: preset`. The list of types is closed, and this is its fourth value.

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

Keys are the application's, in the spelling `numen.json` already uses. A key the application does not own is left where it stands, as every other note's frontmatter is (ADR-0018).

### A deck points at a preset

A deck names its preset with an entry of its `links:` block carrying `type: preset`. `type` is an open vocabulary in that block, so nothing about the note format changes.

Several decks pointing at one preset is what sharing it looks like. Scheduling a deck differently is repointing one link, and the deck's own tab is where a person picks the preset from the ones the vault holds. **A deck naming no preset is scheduled by the defaults**, and a deck naming a note that is not a preset is a problem against the deck.

### What a preset settles, and what it does not

Everything about how a deck is scheduled is the preset's: how many new cards and how many reviews a day, whether a budget is spent on a card or on a showing, how long a day runs, the retention target, what counts as learned, how much of a day goes to what is overdue, the share of the load each day of the week carries, an even load, and the goal that steers them. Each of them is settled in a record of its own. The order cards arrive in, what is done about a card that will not stick and what is done about two faces of one card belong here too, and each arrives with the code that reads it.

`numen.json` keeps the hour a day begins at. It is a fact about a person's clock rather than about a subject.

### Budgets add up

A preset's budget is spent on the cards of the decks pointing at it, and a sitting over the whole vault is the union of them. Ten minutes on one preset and twenty on another is thirty minutes.

**Inside one preset, one budget closes the day**, and which of them it is the preset's goal says (ADR-0036).

**A day is one day whatever presets fall on it**, so how loaded each of them is, is one table, and the day a card is put on is read off it (ADR-0040).

## Consequences

- **A preset travels with the vault.** It is text the person wrote, in their history, syncing with everything else.
- **There is no vault-wide scheduling scope.** The scopes are the installation, a preset, and the deck that points at one.
- **A number is enforced where it was typed.** Decks are files and nothing contains anything, so no limit is displaced onto a parent.
- **The application writes to a note it did not create.** Settings written into the vault are ADR-0017's write path, and a preset is the first note the application edits key by key rather than whole.
- **A card is scheduled at its own preset's target.** The answers are replayed under the scheduler of the preset the card's deck points at, and the schedule cache carries a mark of which cards stood under which target. A mark that does not match is a cache thrown away whole.
- **The settings a person can change stand in two files.** `numen.json` is the installation's and is written down in [Settings](../settings.md); a preset is the vault's and is written down in [Cards](../cards.md), beside the deck it schedules.

## Alternatives considered

**Scheduling settings in `numen.json`.** Rejected: the file is one installation's, and a person studying two subjects on one machine has one set of limits for both.

**A scheduling scope inside the vault, one for all its decks.** Rejected: it is the same fault one level down, and a preset already gives a vault as many as it wants.

**Settings written in the preset's body**, as a deck's cards and a stencil's faces are. Rejected: the body is where a value may hold a heading or a paragraph. A limit is a scalar, and frontmatter is where this format keeps scalars.

**Keys in the language of the interface.** Rejected: the vault format is one spelling, and `type`, `fields` and `links` are already written in it.

**A deck carrying its own limits.** Rejected: two decks that should agree would have no way to say so, and the settings would be copied by hand into every deck of a subject.
