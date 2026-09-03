# ADR-0041: A preset's day is divided over the decks it schedules

- **Status:** Accepted
- **Date:** 2026-09-01
- **Applies to:** `modules/libs/core` — `usecase/flashcards`
- **Related:** ADR-0034, ADR-0036, ADR-0038

## Context

A preset's budget is the scope of one day (ADR-0036), and several decks point at one preset (ADR-0034). What that day comes to in each of those decks is a second question, and two places ask it: the row a deck stands at on the front door, and the sitting opened by pressing that deck.

## Decision

### The day is divided over every deck the preset schedules, once

A preset's day is handed out over the decks it schedules before any of it is spent. A sitting over one deck is that deck's slice of the one division, and the deck's row on the front door is the same slice. The two are one number because they are one arithmetic: the division runs over every deck, and what the sitting was opened over selects from what it left.

Adding a deck does not add to the day's work; it spreads the same work over more decks. Raising the work is raising the budget, and nothing else.

### A deck's share is what it owes of that budget

The shares are proportional: three decks owing the same take a third each, and a deck owing nine times another's takes nine times the share. What the proportions leave over goes by the largest fraction, and decks standing equal take it in the order their paths stand, so a vault divides the same day however its files are walked.

Every budget the goal names is divided this way, each over what the decks owe of that budget: the reviews over what is due in each deck, the new cards over what each has not begun, the minutes over what each deck's cards cost to answer.

What a deck has already answered today comes off its own share, and the whole day is what is divided rather than what is left of it. So a deck sat first spends its share and no other's, and sitting the decks in any order spends the one day.

### What no deck could use goes to the decks that can

A share too small to buy a card, or larger than the deck has cards to spend it on, is not left on the day. What the shares leave over is offered round again to the decks that still hold a card, in the order their paths stand. A card is charged both to its deck's share and to the preset's day, so the day holds however the shares are drawn.

Inside a deck the order is untouched: the debt first, the card waiting longest at the front, then the cards nobody has answered, under the share `backlog` names (ADR-0038).

## Consequences

- **A deck's row is a promise the sitting keeps.** The count beside a deck is the count pressing it hands over, by construction and not by two arithmetics agreeing.
- **Sitting in one deck does not eat another deck's.** A person who spends a morning on one deck finds the others holding what they held.
- **A share follows what a deck owes, so it moves as the deck does.** A deck emptied of its debt hands its share to the decks that still carry theirs, within the same day.

## Alternatives considered

**Equal shares.** Rejected: a deck of five cards beside one of five hundred would hold half the day and spend a tenth of it.

**A budget of its own for each deck.** Rejected: the preset would stop being the scope of a day, and adding a deck would add to the day's work.
