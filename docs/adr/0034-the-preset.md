# ADR-0034: A preset says how a deck is scheduled

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format — every application that reads or writes one
- **Amends:** ADR-0027 (a fourth value of `type`)
- **Related:** ADR-0017, ADR-0018, ADR-0026, ADR-0027, ADR-0031, ADR-0036, ADR-0037, ADR-0038, ADR-0039

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

**How loaded each day is, is one table.** A day is one day whatever presets fall on it, so spreading a card reads what every preset has already put there and applies its own shares and its own willingness to move a card. The projection behind the control does this over the cards of one preset; the table across every preset arrives with the scheduling that reads it.

### One rule says which day a card lands on

**A card's day is chosen in one function, and the sitting and the picture both go through it.** The scheduler works out an interval; the day inside the tolerance around that interval is chosen by weight, where a day's weight is the share of the load its day of the week carries over what already falls on it. A day at nothing weighs nothing and takes no card; a day already carrying more takes fewer. Two implementations kept in step by tests are two answers to one question, and the fault they produce is a picture promising a load the sitting never delivers.

It is pressure and not a promise. No day is forbidden to carry more than its share, and nothing is solved over the collection: the tolerance is empty on short intervals and closed on long ones, so a card with nowhere to go stands where it fell.

**The picture draws the next sitting, not today.** The control's curve reads the first day of the run the preset admits. A day at none of the load is no sitting at all, so a person moving a control on such a day reads what the setting buys them on the day they will next sit down, rather than a row of noughts. It is one real day of the projection, worked out by the arithmetic the sitting runs, so the count on the curve is the count that sitting hands them. Days the preset does not admit take no part in any summary over the run.

**A preset aiming at a day evens no load.** The pace is what spreads a date's material over its days, and the days it has are the days it needs. A window of a placement holds nothing beyond the front of a run, so its lightest day is its last, and a card put there is a card asked for later than the pace was told it would be.

**An even load off is no placement at all.** The card lands where the scheduler put it. If the day it lands on does not admit it, it is not shown that day: it stands overdue, the next day picks it up, and that day is larger by the share the light day shed. Nothing is written anywhere — there is no schedule in the vault to write to.

## Consequences

- **A preset travels with the vault.** It is text the person wrote, in their history, syncing with everything else.
- **There is no vault-wide scheduling scope.** The scopes are the installation, a preset, and the deck that points at one.
- **A number is enforced where it was typed.** Decks are files and nothing contains anything, so no limit is displaced onto a parent.
- **The application writes to a note it did not create.** Settings written into the vault are ADR-0017's write path, and a preset is the first note the application edits key by key rather than whole.
- **A card is scheduled at its own preset's target.** The answers are replayed under the scheduler of the preset the card's deck points at, and the schedule cache carries a mark of which cards stood under which target. A mark that does not match is a cache thrown away whole.
- **The mark carries the placement too.** A preset's shares, its even load, the goal that decides whether that load is evened at all, and the hour a day of review begins at all decide which day a card lands on, so all of them stand in the mark the cache is filed under.
- **A day is a whole number.** The days are counted from the day the clock is counted from, so the same answers name the same days in every process and a schedule worked out again is the schedule that was worked out.
- **The settings a person can change stand in two files.** `numen.json` is the installation's and is written down in [Settings](../settings.md); a preset is the vault's and is written down in [Cards](../cards.md), beside the deck it schedules.

## Alternatives considered

**Scheduling settings in `numen.json`.** Rejected: the file is one installation's, and a person studying two subjects on one machine has one set of limits for both.

**A scheduling scope inside the vault, one for all its decks.** Rejected: it is the same fault one level down, and a preset already gives a vault as many as it wants.

**Settings written in the preset's body**, as a deck's cards and a stencil's faces are. Rejected: the body is where a value may hold a heading or a paragraph. A limit is a scalar, and frontmatter is where this format keeps scalars.

**Keys in the language of the interface.** Rejected: the vault format is one spelling, and `type`, `fields` and `links` are already written in it.

**A deck carrying its own limits.** Rejected: two decks that should agree would have no way to say so, and the settings would be copied by hand into every deck of a subject.

**A list of light days, each carrying a fixed half and shedding the rest onto the day either side.** Rejected: the half was a number nothing decided, and a person who wants a quiet Saturday and no Sunday at all has one word for both. A share says every one of them, and nothing is shed anywhere a person did not ask for.

**A day chosen once for the picture and again for the sitting.** Rejected: the two were kept in step by tests, and drifted the moment one of them was touched. The control drew an even load the scheduler never delivered.
