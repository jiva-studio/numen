# ADR-0036: The goal names the budget that closes the day

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format, and `modules/libs/core` — `flashcards`, `usecase/flashcards`
- **Related:** ADR-0018, ADR-0031, ADR-0034, ADR-0037, ADR-0038, ADR-0039

## Context

A preset holds several numbers a day could end at: the minutes it runs, the unseen cards it holds, the returning cards it holds, and the day the material is to be learned by. Held to all of them at once, a day ends at whichever runs out first, and a person who raises the number in front of them finds the day the length it was.

## Decision

### One key names the budget

`goal` names which of three a preset is steered by — `minutes_a_day`, `retention` or `by_date` — and the value stands under the key it names.

### That budget is the only one that closes the day

Under `minutes_a_day` the day is spent against the time each answer took. Under `retention` the target sets the intervals and the two card counts close the day. Under `by_date` the day holds what has to be got through to have the material learned by that day, and neither the minutes nor the counts cut it short. What being learned means under a date is the preset's own rule (ADR-0037).

A preset steered by minutes turns them into a count from its own answer times; from there everything is counts.

### A setting the goal does not name keeps its value

It takes no part while another goal is in force, and is in force again the moment its own goal is chosen. It is neither zeroed nor removed: a person who set a card limit last month finds it where they left it.

### A budget of zero is a pause

A preset whose governing budget is zero schedules nothing, and every deck pointing at it stops. Pausing one deck is a preset of its own. A budget the goal does not name is not a pause, whatever it holds.

A goal of a date ends the same way: past the date the preset schedules nothing until the date is moved or the deck is pointed elsewhere. A preset aiming at a day and naming none is paused from the start — the budget its goal names is the day, and a goal that cannot read its own budget schedules nothing.

## Consequences

- **The number a person is shown is the number that closes the day.** One control moves it, and the day answers.
- **A pause is a value.** Nothing is written anywhere to say a preset has stopped, and nothing is cleared to start it again.
- **Three sets of settings stand in one preset.** Only one of them is in force, and the other two are a person's work kept for the day they choose that goal.

## Alternatives considered

**Every budget binding at once.** Rejected: a day would end at whichever number ran out first, and which one that was is not what a person is changing.

**Zeroing or removing a setting the goal does not name.** Rejected: choosing a goal would spend the settings of the other two.
