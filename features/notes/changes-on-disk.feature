Feature: A note that changes on disk
  A person writes a note. Another program changes the same file.

  Background:
    Given the note "Shopping list" is open with "tea"

  Scenario: The note follows a change in the file
    When another program writes "milk" in the file
    Then the note has "milk"

  Scenario: Unsaved text is kept when the file changes
    Given the person writes "bread"
    When another program writes "milk" in the file
    Then the note warns about "change"
    And the note has "bread"

  Scenario: The person keeps their text
    Given the person writes "bread"
    And another program writes "milk" in the file
    When the person chooses "keep my text"
    Then the file has "bread"
    And the file does not have "milk"

  Scenario: The person takes the text from the file
    Given the person writes "bread"
    And another program writes "milk" in the file
    When the person chooses "use the file text"
    Then the note has "milk"
    And the note does not have "bread"

  Scenario: The note takes the latest change
    Given the person writes "bread"
    And another program writes "milk" in the file
    And another program writes "eggs" in the file
    When the person chooses "use the file text"
    Then the note has "eggs"

  Scenario: A removed file keeps the text in the note
    When another program removes the file
    Then the note warns about "removal"
    And the note has "tea"

  Scenario: The person saves a removed note to its file again
    Given another program removes the file
    When the person chooses "save the note again"
    Then the file has "tea"
