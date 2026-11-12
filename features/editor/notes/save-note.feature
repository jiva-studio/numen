Feature: Save note
  A note is saved while the person writes it.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text |
      | Shopping list.md | tea  |
    And the application "editor" starts
    And the person opens the note titled "Shopping list"

  Scenario: The person writes
    When the person writes "bread" in the note titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"

  Scenario: The person closes the tab
    Given the person writes "bread" in the note titled "Shopping list"
    When the person closes the note tab titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"
