Feature: Save note
  A note is saved while the person writes it.

  Scenario: A note is saved while the person writes
    Given the person has the note titled "Shopping list" with "tea"
    When the person writes "bread" in the note titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"

  Scenario: A note is saved when the person closes its tab
    Given the person has the note titled "Shopping list" with "tea"
    And the person writes "bread" in the note titled "Shopping list"
    When the person closes the tab titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"
