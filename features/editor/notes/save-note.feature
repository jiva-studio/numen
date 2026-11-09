Feature: Save note
  A note is saved while the person writes it.

  Scenario: A note is saved while the person writes
    Given the vault has the following:
      | path                   | text |
      | notes/Shopping list.md | tea  |
    And the person opens the note titled "Shopping list"
    When the person writes "bread" in the note titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"

  Scenario: A note is saved when the person closes its tab
    Given the vault has the following:
      | path                   | text |
      | notes/Shopping list.md | tea  |
    And the person opens the note titled "Shopping list"
    And the person writes "bread" in the note titled "Shopping list"
    When the person closes the tab titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"
