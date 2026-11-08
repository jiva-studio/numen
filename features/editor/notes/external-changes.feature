Feature: External changes of a note
  A person writes a note. Another program changes the same note on disk.

  Background:
    Given the person has the note titled "Shopping list" with "tea"

  Scenario: The note follows a change on disk
    When another program writes "milk" in the note titled "Shopping list" on disk
    Then the note titled "Shopping list" has "milk"

  Scenario: Unsaved text is kept when the note changes on disk
    Given the person has unsaved "bread" in the note titled "Shopping list"
    When another program writes "milk" in the note titled "Shopping list" on disk
    Then the note titled "Shopping list" has "the file changed on disk"
    And the note titled "Shopping list" has "bread"

  Scenario: The person keeps their text
    Given the person has unsaved "bread" in the note titled "Shopping list"
    And another program writes "milk" in the note titled "Shopping list" on disk
    When the person chooses "Keep my version" for the note titled "Shopping list"
    Then the note titled "Shopping list" on disk has "bread"

  Scenario: The person takes the text from disk
    Given the person has unsaved "bread" in the note titled "Shopping list"
    And another program writes "milk" in the note titled "Shopping list" on disk
    When the person chooses "Use disk version" for the note titled "Shopping list"
    Then the note titled "Shopping list" has "milk"
    And the note titled "Shopping list" does not have "bread"

  Scenario: The note takes the latest change on disk
    Given the person has unsaved "bread" in the note titled "Shopping list"
    And another program writes "milk" in the note titled "Shopping list" on disk
    And another program writes "eggs" in the note titled "Shopping list" on disk
    When the person chooses "Use disk version" for the note titled "Shopping list"
    Then the note titled "Shopping list" has "eggs"

  Scenario: A note removed from disk keeps its text
    When another program removes the note titled "Shopping list" from disk
    Then the note titled "Shopping list" has "the note was removed from the vault"
    And the note titled "Shopping list" has "tea"

  Scenario: The person saves a removed note again
    Given another program removes the note titled "Shopping list" from disk
    When the person chooses "Recreate it" for the note titled "Shopping list"
    Then the note titled "Shopping list" on disk has "tea"
