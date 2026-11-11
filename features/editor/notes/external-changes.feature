Feature: External changes of a note
  A person writes a note. Another program changes the same note on disk.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text |
      | Shopping list.md | tea  |
    And the application "editor" starts

  Rule: A note shows the text of its file

    Background:
      Given the person opens the note titled "Shopping list"

    Scenario: The file changes
      When another program writes "milk" in the note titled "Shopping list"
      Then the note titled "Shopping list" has "milk"
      And the note titled "Shopping list" does not have "the file changed on disk"

    Scenario: The file changes twice
      When another program writes "milk" in the note titled "Shopping list"
      And another program writes "eggs" in the note titled "Shopping list"
      Then the note titled "Shopping list" has "eggs"

  Rule: Unsaved text stays

    Background:
      Given the person opens the note titled "Shopping list"
      And the person writes "bread" in the note titled "Shopping list"

    Scenario: The file changes under unsaved text
      When another program writes "milk" in the note titled "Shopping list"
      Then the note titled "Shopping list" has "the file changed on disk"
      And the note titled "Shopping list" has "bread"

    Scenario: The person keeps the text
      Given another program writes "milk" in the note titled "Shopping list"
      When the person chooses "Keep my version" for the note titled "Shopping list"
      Then the note titled "Shopping list" on disk has "bread"

    Scenario: The person uses the disk text
      Given another program writes "milk" in the note titled "Shopping list"
      When the person chooses "Use disk version" for the note titled "Shopping list"
      Then the note titled "Shopping list" has "milk"
      And the note titled "Shopping list" does not have "bread"

    Scenario: The disk changes twice
      Given another program writes "milk" in the note titled "Shopping list"
      And another program writes "eggs" in the note titled "Shopping list"
      When the person chooses "Use disk version" for the note titled "Shopping list"
      Then the note titled "Shopping list" has "eggs"

  Rule: A removed note keeps its text

    Background:
      Given the person opens the note titled "Shopping list"

    Scenario: The file is removed
      When another program removes the note titled "Shopping list" from disk
      Then the note titled "Shopping list" has "the note was removed from the vault"
      And the note titled "Shopping list" has "tea"

    Scenario: The person saves the note again
      Given another program removes the note titled "Shopping list" from disk
      When the person chooses "Recreate it" for the note titled "Shopping list"
      Then the note titled "Shopping list" on disk has "tea"
