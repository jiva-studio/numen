# ADR-0034: A preset says how a deck is scheduled

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the vault format — every application that reads or writes one
- **Amends:** ADR-0027 (a fourth value of `type`)
- **Related:** ADR-0017, ADR-0018, ADR-0026, ADR-0027, ADR-0031

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

Several decks pointing at one preset is what sharing it looks like. Scheduling a deck differently is repointing one link. **A deck naming no preset is scheduled by the defaults**, and a deck naming a note that is not a preset is a problem against the deck.

### The goal names the budget, and nothing else closes the day

`goal` names which of three a preset is steered by — `minutes_a_day`, `retention` or `by_date` — and the value stands under the key it names.

That budget is the only one that closes the day. Under `minutes_a_day` the day is spent against the time each answer took. Under `retention` the target sets the intervals and the two card counts close the day. Under `by_date` the day holds what has to be got through to have the material learned by that day, and neither the minutes nor the counts cut it short.

A setting the goal does not name keeps its value, takes no part while another goal is in force, and is in force again the moment its own goal is chosen. It is neither zeroed nor removed: a person who set a card limit last month finds it where they left it.

### What a preset settles, and what it does not

Everything about how a deck is scheduled is the preset's. This record settles ten of them: how many new cards and how many reviews a day, whether a budget is spent on a card or on a showing, how long a day runs, the retention target, what counts as learned, how much of a day goes to what is overdue, the share of the load each day of the week carries, an even load, and the goal that steers them. The order cards arrive in, what is done about a card that will not stick and what is done about two faces of one card belong here too, and each arrives with the code that reads it.

`numen.json` keeps the hour a day begins at. It is a fact about a person's clock rather than about a subject.

### A preset says what counts as learned

`learned` names the rule a preset counts by — `interval` or `retention` — and the value stands under the key it names, as `goal` does.

Under `learned: interval` a card is learned once the interval it is sent away for reaches `interval` days: it is learned when it is being asked for at long range. Under `learned: retention` it is learned once the chance of recalling it today stands at or above `retention`: it is learned while it is still in the head. The rule the preset does not name keeps its value and takes no part, exactly as a budget the goal does not name does.

**The rule is the person's.** A hundred words of vocabulary and a hundred ślokas are not learned at the same interval, and a person carrying a subject to an examination means something else by the word than one keeping a language alive. It is a fact about a subject and the person studying it, so it stands in the preset beside everything else about how a deck is scheduled.

**One question, one function.** How much of a preset's material stands learned is read from the rule in one place, so the number on a screen and the number in a projection are one number. A preset that names no rule counts by the default rule at its default value: a key nobody wrote leaves the default in force, and never a threshold every card passes.

**Learned is a state and not a milestone.** A card face stands learned while its reviews are far enough apart, or while it is likely enough to be recalled, and a lapse takes it back out of that standing: the interval collapses and the chance of recall with it. Both rules are asked of where the card stands now.

**A day the whole material is learned is a prediction made under a stated assumption.** A projection takes what it assumes about recall as an input. The figures beside this one are read off the run that follows each card down the middle of what it may do; this one is read off the run in which nothing is forgotten, and it is the soonest the material could be learned. It says what the material could reach, not what will happen.

**It is drawn under a goal of minutes and under a goal of retention, and it is absent under a date.** A preset aiming at a day names its day already, and what qualifies that day is the count no pace reaches. The figure is asked only where the rule is an interval: an interval is passed once and stays passed, while a chance of recall is a level a material stands at and reaches no such day.

**The shape steps.** The day a card passes an interval is the reviews it takes to get there times the space between them, a review is a whole number, and a retention target moves both — it shortens every interval and it decides how many reviews the card wants. So over that control's range the figure falls away and steps up wherever another review is wanted. Every value of it is right where it is asked, and a step is the world's arithmetic and not a fault to be smoothed. A prediction may be drawn across one, because what it claims is the day its assumption reaches and that claim holds at a step as everywhere else. A measurement could not: it would be read as the day the run will meet, and no such day stands steady under a control that moves the scheduler.

The scheduler's own reckoning is another thing and carries another word: a card it has stopped sending minutes away and begun sending days away is **spaced**, and retention is measured over the answers to spaced cards. No setting reaches that.

### A date aims at that rule, tested on the day it names

A person who names a day means they will know the material by it. `by_date` is that day, and what knowing means is the `learned` rule of the same preset, asked of every card face on the day itself.

**The pace follows from the rule.** A card face has to be begun early enough to be learned by then, so the material is spread over the days on which beginning one still leaves it time: under `learned: interval` a card that must be sent away for three weeks is begun weeks before the day, and under `learned: retention` one that has only to be recalled on the day may be begun on it. The same material and the same date are two paces, and each is worked out from the same rule the deck screen counts by.

**What no pace can reach is counted and said.** A card face added a fortnight before a day it must stand three weeks away from cannot get there, whatever a person does: the arithmetic forbids it, and no pace is a remedy. The projection carries how many card faces the day leaves short, and the picture says the number. The day is not moved, the rule is not bent, and the pace beside that number is the one that gets there every card face that can.

**The date is the answer, and nothing beside it answers again.** A preset aiming at a day names no other day the material is learned on: the day it names is that day, and what qualifies it is the count no pace reaches. Past that day the preset schedules nothing, so what a projection shows past it is a material nobody is answering: the debt climbs and what was learned fades. Those days belong to the pause and are drawn as no part of the choice.

### Budgets add up

A preset's budget is spent on the cards of the decks pointing at it, and a sitting over the whole vault is the union of them. Ten minutes on one preset and twenty on another is thirty minutes.

**Inside one preset, the budget its goal names is what closes the day.** A preset steered by minutes turns them into a count from its own answer times; from there everything is counts.

**How loaded each day is, is one table.** A day is one day whatever presets fall on it, so spreading a card reads what every preset has already put there and applies its own shares and its own willingness to move a card. The projection behind the control does this over the cards of one preset; the table across every preset arrives with the scheduling that reads it.

### A day of the week carries a share of the load

`load` is how much of a day's load each day of the week carries, in per cent, under the first three letters of the day's name. A day the preset does not name carries the whole of it.

That share scales every budget the day keeps: a Saturday at 50 holds half the minutes and half of each card count. **A day at nothing schedules nothing**, the way a governing budget of zero is a pause, and it is a pause of that one day.

The share is read whether or not the days are evened out. It is what a day admits, and evening the days out is what moves a card off one.

### One rule says which day a card lands on

**A card's day is chosen in one function, and the sitting and the picture both go through it.** The scheduler works out an interval; the day inside the tolerance around that interval is chosen by weight, where a day's weight is the share of the load its day of the week carries over what already falls on it. A day at nothing weighs nothing and takes no card; a day already carrying more takes fewer. Two implementations kept in step by tests are two answers to one question, and the fault they produce is a picture promising a load the sitting never delivers.

It is pressure and not a promise. No day is forbidden to carry more than its share, and nothing is solved over the collection: the tolerance is empty on short intervals and closed on long ones, so a card with nowhere to go stands where it fell.

**The picture draws the next sitting, not today.** The control's curve reads the first day of the run the preset admits. A day at none of the load is no sitting at all, so a person moving a control on such a day reads what the setting buys them on the day they will next sit down, rather than a row of noughts. It is one real day of the projection, worked out by the arithmetic the sitting runs, so the count on the curve is the count that sitting hands them. Days the preset does not admit take no part in any summary over the run.

**A preset aiming at a day evens no load.** The pace is what spreads a date's material over its days, and the days it has are the days it needs. A window of a placement holds nothing beyond the front of a run, so its lightest day is its last, and a card put there is a card asked for later than the pace was told it would be.

**An even load off is no placement at all.** The card lands where the scheduler put it. If the day it lands on does not admit it, it is not shown that day: it stands overdue, the next day picks it up, and that day is larger by the share the light day shed. Nothing is written anywhere — there is no schedule in the vault to write to.

### How a day is spent between the overdue and the new

`backlog` is how much of a day goes to what is overdue before anything new is offered, in per cent: 100 is the overdue pile first, 0 is new cards first, and the values between split the day. A side that runs short leaves the rest of the day to the other, so a day is never left unspent.

It is not a budget and it does not close a day; it says what the day the goal admits is spent on. A goal of a date does not use it — the material is to be through by that day, all of it, so the order decides nothing that matters.

### A budget counts cards, and may be told to count showings

`counts: cards` is the default: a card face counts against the day's budget the first time it is answered that day, and every further showing of it that day is free. A hundred a day is a hundred cards, whatever it takes to settle each of them.

`counts: shows` spends a slot on every showing. A subject where a card either comes back or does not is studied differently from one whose cards take four steps to settle, and the preset is where that is said.

The time budget is unaffected: minutes are spent as they are spent, on every answer.

### A budget of zero is a pause

A preset whose governing budget is zero schedules nothing, and every deck pointing at it stops. Pausing one deck is a preset of its own. A budget the goal does not name is not a pause, whatever it holds.

A goal of a date is a budget that ends the same way: past the date, the preset schedules nothing until the date is moved or the deck is pointed elsewhere. A preset aiming at a day and naming none is paused from the start: the budget its goal names is the day, and a goal that cannot read its own budget schedules nothing.

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
