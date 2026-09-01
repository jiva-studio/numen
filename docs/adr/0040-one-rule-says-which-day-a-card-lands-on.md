# ADR-0040: One rule says which day a card lands on

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** `modules/libs/core` — `flashcards`, `usecase/flashcards` — and the picture a client draws beside a preset's controls
- **Related:** ADR-0031, ADR-0034, ADR-0036, ADR-0037, ADR-0039

## Context

A day's share says how much of the load it takes (ADR-0039), and a scheduler that works out an interval has a tolerance around it: several days would do. Which of them a card is asked for on is a question two places want answered — the sitting that hands the cards out, and the picture drawn beside a control while a person moves it.

## Decision

### A card's day is chosen in one function, and the sitting and the picture both go through it

The scheduler works out an interval; the day inside the tolerance around that interval is chosen by weight, where a day's weight is the share of the load its day of the week carries over what already falls on it. A day at nothing weighs nothing and takes no card; a day already carrying more takes fewer.

It is pressure and not a promise. No day is forbidden to carry more than its share, and nothing is solved over the collection: the tolerance is empty on short intervals and closed on long ones, so a card with nowhere to go stands where it fell.

### `even_load` is whether a card is moved off the day it fell on

**An even load off is no placement at all.** The card lands where the scheduler put it. If the day it lands on does not admit it, it is not shown that day: it stands overdue, the next day picks it up, and that day is larger by the share the light day shed. Nothing is written anywhere — there is no schedule in the vault to write to.

**A preset aiming at a day evens no load.** The pace is what spreads a date's material over its days, and the days it has are the days it needs. A window of a placement holds nothing beyond the front of a run, so its lightest day is its last, and a card put there is a card asked for later than the pace was told it would be.

### How loaded each day is, is one table

A day is one day whatever presets fall on it, so spreading a card reads what every preset has already put there and applies its own shares and its own willingness to move a card. The projection behind the control does this over the cards of one preset; the table across every preset arrives with the scheduling that reads it.

### The picture draws the next sitting, not today

The control's curve reads the first day of the run the preset admits. A day at none of the load is no sitting at all, so a person moving a control on such a day reads what the setting buys them on the day they will next sit down, rather than a row of noughts. It is one real day of the projection, worked out by the arithmetic the sitting runs, so the count on the curve is the count that sitting hands them. Days the preset does not admit take no part in any summary over the run.

## Consequences

- **The mark the schedule cache is filed under carries the placement.** A preset's shares, its even load, the goal that decides whether that load is evened at all, and the hour a day of review begins at all decide which day a card lands on, so all of them stand in the mark (ADR-0034).
- **A day is a whole number.** The days are counted from the day the clock is counted from, so the same answers name the same days in every process and a schedule worked out again is the schedule that was worked out.
- **A curve a person reads is a promise the sitting keeps.** Both are the same arithmetic over the same day, so what the control says a setting buys is what the sitting hands over.

## Alternatives considered

**A day chosen once for the picture and again for the sitting.** Rejected: the two were kept in step by tests, and drifted the moment one of them was touched. The control drew an even load the scheduler never delivered.
