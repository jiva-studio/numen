Feature: A note that changes on disk
  A person writes a note. Another program changes the saved note.

  Background:
    Given the note "Shopping list" is open with "tea"

  Scenario: The open note follows a change in the saved note
    When another program writes "milk" in the saved note "Shopping list"
    Then the open note "Shopping list" has "milk"

  Scenario: Unsaved text is kept when the saved note changes
    Given the person writes "bread" in the open note "Shopping list"
    When another program writes "milk" in the saved note "Shopping list"
    Then the open note "Shopping list" warns about "change"
    And the open note "Shopping list" has "bread"

  Scenario: The person keeps their text
    Given the person writes "bread" in the open note "Shopping list"
    And another program writes "milk" in the saved note "Shopping list"
    When the person chooses "keep my text" for the open note "Shopping list"
    Then the saved note "Shopping list" has "bread"
    And the saved note "Shopping list" does not have "milk"

  Scenario: The person takes the text of the saved note
    Given the person writes "bread" in the open note "Shopping list"
    And another program writes "milk" in the saved note "Shopping list"
    When the person chooses "use the saved text" for the open note "Shopping list"
    Then the open note "Shopping list" has "milk"
    And the open note "Shopping list" does not have "bread"

  Scenario: The open note takes the latest change
    Given the person writes "bread" in the open note "Shopping list"
    And another program writes "milk" in the saved note "Shopping list"
    And another program writes "eggs" in the saved note "Shopping list"
    When the person chooses "use the saved text" for the open note "Shopping list"
    Then the open note "Shopping list" has "eggs"

  Scenario: A removed saved note leaves the text in the open note
    When another program removes the saved note "Shopping list"
    Then the open note "Shopping list" warns about "removal"
    And the open note "Shopping list" has "tea"

  Scenario: The person saves a removed note again
    Given another program removes the saved note "Shopping list"
    When the person chooses "save the note again" for the open note "Shopping list"
    Then the saved note "Shopping list" has "tea"
