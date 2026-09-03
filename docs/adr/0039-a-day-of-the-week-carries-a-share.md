# ADR-0039: A day of the week carries a share of the load

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format, and `modules/libs/core` — `flashcards`, `usecase/flashcards`
- **Related:** ADR-0018, ADR-0034, ADR-0036, ADR-0040

## Context

A preset's budgets are one number for every day of the week, and a week is not seven equal days. A person whose Saturday is spoken for wants less of the load on it, and one who keeps a Sunday clear wants none.

## Decision

### `load` is a share, under the day's name

`load` is how much of a day's load each day of the week carries, in per cent, under the first three letters of the day's name. A day the preset does not name carries the whole of it.

That share scales every budget the day keeps: a Saturday at 50 holds half the minutes and half of each card count.

### A day at nothing schedules nothing

It is a pause of that one day, the way a governing budget of zero is a pause of the preset (ADR-0036). The preset schedules on the days around it, and nothing about it has stopped.

### The share is what a day admits

It is read whether or not the days are evened out. What the share says is how much a day takes; moving a card off the day it fell on is another decision and another key (ADR-0040).

## Consequences

- **A quiet day and a day off are one setting.** 50 and 0 are values of the same key, and a person says both in the same place.
- **The share is on every budget at once.** A day at half holds half the minutes, half the new cards and half the reviews, so a light day is a light day whichever goal is steering.

## Alternatives considered

**A list of light days, each carrying a fixed half and shedding the rest onto the day either side.** Rejected: the half was a number nothing decided, and a person who wants a quiet Saturday and no Sunday at all has one word for both. A share says every one of them, and nothing is shed anywhere a person did not ask for.
