# ADR-0038: What a day's budget is spent on, and in what order

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format, and `modules/libs/core` — `flashcards`, `usecase/flashcards`
- **Related:** ADR-0018, ADR-0034, ADR-0036

## Context

The goal fixes how much a day holds (ADR-0036). Two things it does not fix: what draws that budget down, and which cards reach it first. A hundred cards a day is one number over a subject whose cards either come back or do not, and another over one whose cards take four steps to settle. A person opening a vault after a fortnight away has a pile of overdue cards and a deck of unseen ones, and which of the two the day goes to is theirs to say.

## Decision

### A budget counts cards, and may be told to count showings

`counts: cards` is the default: a card face counts against the day's budget the first time it is answered that day, and every further showing of it that day is free. A hundred a day is a hundred cards, whatever it takes to settle each of them.

`counts: shows` spends a slot on every showing. A subject where a card either comes back or does not is studied differently from one whose cards take four steps to settle, and the preset is where that is said.

The time budget is unaffected: minutes are spent as they are spent, on every answer.

### How a day is spent between the overdue and the new

`backlog` is how much of a day goes to what is overdue before anything new is offered, in per cent: 100 is the overdue pile first, 0 is new cards first, and the values between split the day. A side that runs short leaves the rest of the day to the other, so a day is never left unspent.

It is not a budget and it does not close a day; it says what the day the goal admits is spent on. A goal of a date does not use it — the material is to be through by that day, all of it, so the order decides nothing that matters.

## Consequences

- **What a day holds and what it is spent on are two settings.** Neither moves the other: a preset told to count showings keeps the number that was typed into it, and the day is shorter in cards.
- **A day is never left unspent.** Whichever side runs short, the rest of the day goes to the other.

## Alternatives considered

**A slot on every showing, always.** Rejected: one number would mean two things, and only the person studying the subject knows which of them they meant.

**The overdue share as a budget of its own.** Rejected: a second budget would close the day, and a short pile would end a day with time left in it.
