# Feature Files (Gherkin)

This rule defines the format of the acceptance scenarios in `features/`. The scenarios are business-facing criteria in domain language; the driver code under `tests/acceptance/` turns them into actions on the app. Sources: Automation Panda, [Gherkin Guidelines for AI](https://github.com/AutomationPanda/gherkin-guidelines-for-ai/blob/main/gherkin-guidelines.md), and the Fowler and Farley view of acceptance tests.

---

## 1. Files

- Path: `features/<area>/<kebab-name>.feature`. The folder is the behaviour area, the file is one part of a feature.
- English only. Lines under 120 characters.
- A handful of scenarios per file. Split the file when it grows past that.

## 2. Header

- One line `Feature: <title>` naming the behaviour area.
- An optional description of one to three lines: a sentence of why, or "As a ... I want ... So that ...".
- No Goals, Non-goals or Invariants sections. A rule the product must keep is a scenario. A `Rule:` block groups the scenarios of one rule: use it when a feature holds more than one rule, and name it with a plain sentence of what must hold. A feature of one or two scenarios needs no Rule.

## 3. Scenarios

- Third person, present tense, declarative. Say what the person wants and sees, not how the interface is operated.
- Domain words only. Never click, press, button, field, panel, palette, menu or similar interface words.
- One behaviour per scenario, in strict Given, When, Then order.
- `Then` is observable and checkable.
- Each scenario stands alone and does not depend on another scenario's state.
- `Scenario Outline` only for the same behaviour with different inputs.
- `Background` only for state shared by several scenarios.
- Double quotes around values.

## 4. Language and data

- Plain English at level A1 to A2: short common words, present tense, no idioms, no figurative speech. The same word for the same thing in every feature (note, file, warns, keeps, takes).
- A step is a short sentence. Two steps that do the same thing are one step: the step checker fails on two definitions with the same body.
- The data is neutral and short, like the data of a real use: a note "Shopping list", the texts "milk", "bread", "eggs". No stories, no invented places, characters or scenery (no lock gates, harbours or dawns), and no sentences that read as prose.
- A scenario title names the rule in a few plain words. Titles and Rule names state a fact the scenario shows, with the verbs of the product and no figures of speech (not follows, gives, takes, lit). No adverbs of time or manner ("at once", "quickly", "silently") unless the rule is that timing.

### Steps are functions

A step is a function with parameters: the sentence is its name, the quoted values and the one word it takes are its arguments. Before a new sentence, look for the step that already does it (`npm run acceptance:steps` in `tests/acceptance/desktop`) and pass it a different value. A choice among several outcomes is one step with the choice as its argument (`the person chooses "keep my text"`, `the note warns about "removal"`), not one sentence for each outcome. One step serves Given, When and Then when it is the same action. Do not rephrase a step to fit one scenario.

## 5. Example

```gherkin
Feature: Delete a note

  As a writer
  I want to remove a note I no longer need
  So that my vault holds only what I keep

  Scenario: A deleted note leaves the vault
    Given the vault holds a note "Groceries"
    When the writer deletes the note "Groceries"
    Then the vault does not hold a note "Groceries"
```

## 6. Ownership

The scenarios belong to the owner. An agent that finds a scenario wrong, or a feature that grew, proposes the change to the feature file in a commit of its own that says so.
