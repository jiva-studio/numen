Feature: External changes of a note
  Another program changes the file of a note that is open.

  As a person who keeps notes
  I want to see when another program changes or removes a note that is open
  So that I do not lose my text and I choose which version I keep

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text |
      | Shopping list.md | tea  |
    And the application "editor" starts

  Rule: A note follows its file

    Background:
      Given the person opens the note titled "Shopping list"

    Scenario: A note shows the text written last
      When another program writes "milk" in the note titled "Shopping list"
      And another program writes "eggs" in the note titled "Shopping list"
      Then the note titled "Shopping list" has "eggs"
      And the note titled "Shopping list" does not have "the file changed on disk"

    Scenario: A removed file keeps its text
      When another program removes the note titled "Shopping list" from disk
      Then the note titled "Shopping list" has "the note was removed from the vault"
      And the note titled "Shopping list" has "tea"

    Scenario: The note is recreated
      Given another program removes the note titled "Shopping list" from disk
      When the person chooses "Recreate it" for the note titled "Shopping list"
      Then the note titled "Shopping list" on disk has "tea"

  Rule: Unsaved text stays

    Background:
      Given the person opens the note titled "Shopping list"
      And the person writes "bread" in the note titled "Shopping list"

    Scenario: A change on disk is announced
      When another program writes "milk" in the note titled "Shopping list"
      Then the note titled "Shopping list" has "the file changed on disk"
      And the note titled "Shopping list" has "bread"

    Scenario: The version of the person is kept
      Given another program writes "milk" in the note titled "Shopping list"
      When the person chooses "Keep my version" for the note titled "Shopping list"
      Then the note titled "Shopping list" on disk has "bread"

    Scenario: The disk version is used
      Given another program writes "milk" in the note titled "Shopping list"
      And another program writes "eggs" in the note titled "Shopping list"
      When the person chooses "Use disk version" for the note titled "Shopping list"
      Then the note titled "Shopping list" has "eggs"
      And the note titled "Shopping list" does not have "bread"
