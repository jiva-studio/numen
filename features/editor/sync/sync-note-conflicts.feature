Feature: Sync a note changed on two computers

  Numen Sync keeps the latest version of a note and saves the other version in a conflict copy next to it.

  As a person who uses Numen on more than one computer
  I want no text of mine to be lost when two computers change the same note
  So that I can choose later which text to keep

  Background:
    Given the person has the synced vault "Test" on two computers

  Rule: The latest version stays and the other is kept in a conflict copy

    @stand
    Scenario: A note changed on two computers keeps the latest version
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      And the second computer is the first to reach the sync server
      When the sync server runs again
      Then the vault "Test" has the note "Shopping list" with the text "eggs"
      And the second computer has the note "Shopping list" with the text "eggs" in its vault "Copy"

    @stand
    Scenario: The other version is kept in a conflict copy
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      And the second computer is the first to reach the sync server
      When the sync server runs again
      Then the vault "Test" has a conflict copy of the note "Shopping list" with the text "bread"
      And the second computer has a conflict copy of the note "Shopping list" with the text "bread" in its vault "Copy"

    @stand
    Scenario: Numen Sync warns about a note changed on two computers
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      And the second computer is the first to reach the sync server
      When the sync server runs again
      Then the person is told "1 note changed on two computers"

    @stand
    Scenario: The same text on both computers makes no conflict copy
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "bread" in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has no conflict copy of the note "Shopping list"
      And the second computer has no conflict copy of the note "Shopping list" in its vault "Copy"

    @stand
    Scenario: A note changed on one computer only makes no conflict copy
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      When the sync server runs again
      Then the second computer has the note "Shopping list" with the text "bread" in its vault "Copy"
      And the vault "Test" has no conflict copy of the note "Shopping list"

    @stand
    Scenario: A second conflict keeps the first conflict copy
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      And the second computer is the first to reach the sync server
      And the sync server runs again
      And the vault "Test" has a conflict copy of the note "Shopping list" with the text "bread"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "tea" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "rice" in its vault "Copy"
      And the second computer is the first to reach the sync server
      When the sync server runs again
      Then the vault "Test" has a conflict copy of the note "Shopping list" with the text "bread"
      And the vault "Test" has a conflict copy of the note "Shopping list" with the text "tea"

  Rule: A rename is not a conflict

    @stand
    Scenario: A note renamed on one computer and changed on the other keeps the change
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person moves the file "Shopping list.md" to "Groceries.md" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has the note "Groceries" with the text "eggs"
      And the second computer has the note "Groceries" with the text "eggs" in its vault "Copy"
      And the vault "Test" has no conflict copy of the note "Groceries"

    @live
    Scenario: A note renamed on one computer makes no conflict on the other
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      When the person moves the file "Shopping list.md" to "Groceries.md" in the vault "Test"
      Then the second computer has the note "Groceries" with the text "milk" in its vault "Copy"
      And the second computer has no conflict copy of the note "Groceries" in its vault "Copy"
