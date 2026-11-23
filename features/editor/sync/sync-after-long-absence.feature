Feature: Sync after a long absence

  Numen Sync brings a computer up to date when it was away longer than the sync server keeps the removals.

  As a person who uses Numen on more than one computer
  I want a computer that was away for a long time to catch up
  So that removed notes do not come back and no change of mine is lost

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person connects the vault "Test" as a new synced vault
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"
    And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
    And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
    And the person writes the note "Plan" with the text "tea" in the vault "Test"
    And the second computer has the note "Plan" with the text "tea" in its vault "Copy"
    And the second computer is offline

  Rule: A note removed while a computer was away stays removed

    Scenario: A removed note does not come back
      Given the person removes the note "Shopping list" in the vault "Test"
      And the sync server forgets which notes were removed
      When the second computer is online
      Then the vault "Test" is without the note "Shopping list"
      And the second computer is without the note "Shopping list" in its vault "Copy"

    Scenario: A note that did not change stays
      Given the person removes the note "Shopping list" in the vault "Test"
      And the sync server forgets which notes were removed
      When the second computer is online
      Then the second computer has the note "Plan" with the text "tea" in its vault "Copy"

  Rule: A change made while a computer was away is kept

    Scenario: A changed note of the away computer comes back
      Given the second computer writes the note "Shopping list" with the text "bread" in its vault "Copy"
      And the person removes the note "Shopping list" in the vault "Test"
      And the sync server forgets which notes were removed
      When the second computer is online
      Then the vault "Test" has the note "Shopping list" with the text "bread"

    Scenario: A new note of the away computer arrives
      Given the second computer writes the note "Ideas" with the text "eggs" in its vault "Copy"
      And the sync server forgets which notes were removed
      When the second computer is online
      Then the vault "Test" has the note "Ideas" with the text "eggs"
