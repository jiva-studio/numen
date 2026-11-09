Feature: Link notes
  A note links to another note by its title.

  Background:
    Given the vault "Test" contains the note at "notes/Entropy.md"

  Scenario: A link is written with a completion
    Given the vault has the following:
      | path                 | text                                   |
      | notes/Heat engine.md | Work from a difference in temperature. |
    And the person opens the note titled "Heat engine"
    When the person writes "See [[Entr" in the note titled "Heat engine"
    And the person chooses the completion "Entropy" in the note titled "Heat engine"
    Then the note titled "Heat engine" on disk has "See [[Entropy]]"
    And the note titled "Heat engine" links to the note at "notes/Entropy.md"

  Scenario: A link opens the note it names
    Given the vault has the following:
      | path                  | text                                       |
      | notes/Carnot cycle.md | The cycle is bounded by [[Entropy]] alone. |
    And the person opens the note titled "Carnot cycle"
    When the person opens the link "[[Entropy]]" in the note titled "Carnot cycle"
    Then the tab at "notes/Entropy.md" is active
    And the note at "notes/Entropy.md" has "A measure of uncertainty"
