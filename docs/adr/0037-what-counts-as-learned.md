# ADR-0037: A preset says what counts as learned, and a date aims at it

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format, and `modules/libs/core` — `flashcards`, `usecase/flashcards`
- **Related:** ADR-0018, ADR-0031, ADR-0034, ADR-0036

## Context

Two questions are asked of a deck in these words: how much of it stands learned today, and when the whole of it will. Both want a rule, and the application has none to hand out. A hundred words of vocabulary and a hundred ślokas are not learned at the same interval, and a person carrying a subject to an examination means something else by the word than one keeping a language alive.

The scheduler's own reckoning is another thing and carries another word: a card face it has stopped sending minutes away and begun sending days away is **spaced**, and retention is measured over the answers to spaced cards. That is a fact about how the scheduler is treating a card, and no setting reaches it.

## Decision

### A preset says what counts as learned

`learned` names the rule a preset counts by — `interval` or `retention` — and the value stands under the key it names, as `goal` does (ADR-0036).

Under `learned: interval` a card is learned once the interval it is sent away for reaches `interval` days: it is learned when it is being asked for at long range. Under `learned: retention` it is learned once the chance of recalling it today stands at or above `retention`: it is learned while it is still in the head. The rule the preset does not name keeps its value and takes no part, exactly as a budget the goal does not name does.

**The rule is the person's.** It is a fact about a subject and the person studying it, so it stands in the preset beside everything else about how a deck is scheduled.

**One question, one function.** How much of a preset's material stands learned is read from the rule in one place, so the number on a screen and the number in a projection are one number. A preset that names no rule counts by the default rule at its default value: a key nobody wrote leaves the default in force, and never a threshold every card passes.

**Learned is a state and not a milestone.** A card face stands learned while its reviews are far enough apart, or while it is likely enough to be recalled, and a lapse takes it back out of that standing: the interval collapses and the chance of recall with it. Both rules are asked of where the card stands now.

**A day the whole material is learned is a prediction made under a stated assumption.** A projection takes what it assumes about recall as an input. The figures beside this one are read off the run that follows each card down the middle of what it may do; this one is read off the run in which nothing is forgotten, and it is the soonest the material could be learned. It says what the material could reach, not what will happen.

**It is drawn under a goal of minutes and under a goal of retention, and it is absent under a date.** A preset aiming at a day names its day already, and what qualifies that day is the count no pace reaches. The figure is asked only where the rule is an interval: an interval is passed once and stays passed, while a chance of recall is a level a material stands at and reaches no such day.

**The shape steps.** The day a card passes an interval is the reviews it takes to get there times the space between them, a review is a whole number, and a retention target moves both — it shortens every interval and it decides how many reviews the card wants. So over that control's range the figure falls away and steps up wherever another review is wanted. Every value of it is right where it is asked, and a step is the world's arithmetic and not a fault to be smoothed. A prediction may be drawn across one, because what it claims is the day its assumption reaches and that claim holds at a step as everywhere else.

### A date aims at that rule, tested on the day it names

A person who names a day means they will know the material by it. `by_date` is that day, and what knowing means is the `learned` rule of the same preset, asked of every card face on the day itself.

**The pace follows from the rule.** A card face has to be begun early enough to be learned by then, so the material is spread over the days on which beginning one still leaves it time: under `learned: interval` a card that must be sent away for three weeks is begun weeks before the day, and under `learned: retention` one that has only to be recalled on the day may be begun on it. The same material and the same date are two paces, and each is worked out from the same rule the deck screen counts by.

**What no pace can reach is counted and said.** A card face added a fortnight before a day it must stand three weeks away from cannot get there, whatever a person does: the arithmetic forbids it, and no pace is a remedy. The projection carries how many card faces the day leaves short, and the picture says the number. The day is not moved, the rule is not bent, and the pace beside that number is the one that gets there every card face that can.

**The date is the answer, and nothing beside it answers again.** A preset aiming at a day names no other day the material is learned on: the day it names is that day, and what qualifies it is the count no pace reaches. Past that day the preset schedules nothing (ADR-0036), so what a projection shows past it is a material nobody is answering: the debt climbs and what was learned fades. Those days belong to the pause and are drawn as no part of the choice.

## Consequences

- **The screen and the pace agree.** A date's pace is worked out from the same function the deck screen counts by, so a person is never told a material stands learned on a day the date says it does not.
- **A rule changed is a pace changed.** Under a date the day a card is begun on follows from the rule, so choosing the other one moves every card that has not been begun.
- **The word is spent once.** What the scheduler is doing to a card is `spaced`, and what a person has learned is `learned`. Neither is read from the other.

## Alternatives considered

**One meaning of learned for the application.** Rejected: the word is a fact about a subject and the person studying it, and a number the application chose would be right for one subject in a vault.

**Reading learned off the scheduler's own reckoning.** Rejected: `spaced` says a card is being sent days away rather than minutes, which is a decision the scheduler made about a card and not a claim about the person.

**A default that every card passes.** Rejected: a preset naming no rule would report its material learned the day it was written.

**Drawing the day as a measurement rather than a prediction.** Rejected: a measurement is read as the day the run will meet, and no such day stands steady under a control that moves the scheduler.

**Smoothing the step out of the figure.** Rejected: a review is a whole number, so the step is the arithmetic. A curve drawn through it is right nowhere it is read.

**Moving the day, or bending the rule, for the card faces no pace reaches.** Rejected: the person named the day, and a count of what cannot reach it is an answer where a moved day is a different question.
