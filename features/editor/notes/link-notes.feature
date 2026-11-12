Feature: Link notes
  A note links to another note by its title.

  As a person who keeps notes
  I want to link a note to another note by its title
  So that I can go from one note to the other

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path            | text                     |
      | Entropy.md      | Entropy is a measure.    |
      | Heat engine.md  | A heat engine does work. |
      | Carnot cycle.md | See [[Entropy]].         |
    And the application "editor" starts

  Scenario: A link is completed
    Given the person opens the note titled "Heat engine"
    When the person writes "See [[Entr" in the note titled "Heat engine"
    And the person chooses the completion "Entropy" in the note titled "Heat engine"
    Then the note titled "Heat engine" on disk has "See [[Entropy]]"
    And the note titled "Heat engine" links to the note at "Entropy.md"

  Scenario: A link opens its note
    Given the person opens the note titled "Carnot cycle"
    When the person opens the link "[[Entropy]]" in the note titled "Carnot cycle"
    Then the note tab at "Entropy.md" is active
    And the note at "Entropy.md" has "Entropy is a measure."
