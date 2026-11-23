Feature: Sync notes while the server is away or a note is in conflict

  Numen Sync keeps the notes of a vault when the server cannot be reached and does not overwrite a note changed on two computers.

  As a person who uses Numen on more than one computer
  I want the notes to survive a lost server and a clash of two changes
  So that I never lose what I wrote

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person connects the vault "Test" as a new synced vault
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: Numen Sync keeps working when the server is away

    Scenario: A note written while the server is away arrives later
      Given the sync server stops
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      When the sync server runs again
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

    Scenario: Numen Sync warns while the server is away
      When the sync server stops
      Then the plugin "Numen Sync" warns about "Could not reach the sync server"

  Rule: A note changed on two computers is not overwritten

    Scenario: A note changed on both computers waits
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      When the sync server runs again
      Then the plugin "Numen Sync" warns about "1 note changed on two computers"
      And the vault "Test" has the note "Shopping list" with the text "bread"
      And the second computer has the note "Shopping list" with the text "eggs" in its vault "Copy"
