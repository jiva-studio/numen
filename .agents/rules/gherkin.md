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
- No Goals, Non-goals or Invariants sections. A rule the product must keep is a scenario. A `Rule:` block may group scenarios.

## 3. Scenarios

- Third person, present tense, declarative. Say what the person wants and sees, not how the interface is operated.
- Domain words only. Never click, press, button, field, panel, palette, menu or similar interface words.
- One behaviour per scenario, at most 5 steps, in strict Given, When, Then order.
- `Then` is observable and checkable.
- Each scenario stands alone and does not depend on another scenario's state.
- `Scenario Outline` only for the same behaviour with different inputs.
- `Background` only for state shared by several scenarios.
- Double quotes around values.

## 4. Example

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

## 5. Ownership

The scenarios belong to the owner. An agent that finds a scenario wrong, or a feature that grew, proposes the change to the feature file in a commit of its own that says so.
