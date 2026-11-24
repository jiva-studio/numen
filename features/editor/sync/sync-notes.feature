Feature: Sync notes between computers

  Numen Sync sends the notes of a connected vault to the identity and brings them to every computer.

  As a person who uses Numen on more than one computer
  I want a note I write on one computer to be on the other
  So that I can continue my work anywhere

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: A note written on one computer appears on the other

    Scenario: A new note appears on the second computer
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

    Scenario: A changed note changes on the second computer
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      When the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "bread" in its vault "Copy"

    Scenario: A note in a folder keeps its folder
      When the person writes the note "Food/Shopping list" with the text "eggs" in the vault "Test"
      Then the second computer has the note "Food/Shopping list" with the text "eggs" in its vault "Copy"

    Scenario: A note written on the second computer appears on the first
      When the second computer writes the note "Plan" with the text "tea" in its vault "Copy"
      Then the vault "Test" has the note "Plan" with the text "tea"

  Rule: The server holds no readable note

    Scenario: The sync server cannot read a note
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      Then the sync server holds no stored text "milk"
      And the sync server holds no stored text "Shopping list"

  Rule: Only notes are synced

    Scenario: A file that is not a note stays on its computer
      When the person writes the file "data.bin" in the vault "Test"
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the second computer does not have the file "data.bin" in its vault "Copy"
