Feature: Sync notes between computers

  Numen Sync sends the notes of a connected vault to the identity and brings them to every computer.

  As a person who uses Numen on more than one computer
  I want a note I write on one computer to be on the other
  So that I can continue my work anywhere

  Background:
    Given the person has the synced vault "Test" on two computers

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
