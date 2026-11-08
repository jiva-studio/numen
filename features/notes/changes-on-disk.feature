Feature: A note that changes on disk
  A person writes a note. Another program changes the same file.

  Background:
    Given a note "Shopping list" is open

  Scenario: The note follows a change on disk
    When another program writes "milk" in the note
    Then the note has "milk"

  Scenario: Unsaved text is kept when the file changes
    Given the person writes "bread"
    When another program writes "milk" in the note
    Then the note warns that the file changed
    And the note keeps "bread"

  Scenario: The person keeps their text
    Given the person writes "bread"
    And another program writes "milk" in the note
    When the person keeps their text
    Then the file has "bread"
    And the file does not have "milk"

  Scenario: The person takes the text from the file
    Given the person writes "bread"
    And another program writes "milk" in the note
    When the person takes the text from the file
    Then the note has "milk"
    And the note does not have "bread"

  Scenario: The note takes the latest change
    Given the person writes "bread"
    And another program writes "milk" in the note
    And another program writes "eggs" in the note
    When the person takes the text from the file
    Then the note has "eggs"

  Scenario: A removed file keeps the text in the note
    When another program removes the note
    Then the note warns that the file is gone
    And the note keeps its text

  Scenario: A removed note can come back
    Given another program removed the note
    When the person saves the note again
    Then the file is back with the text
