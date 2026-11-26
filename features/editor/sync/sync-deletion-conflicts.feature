@sync-server
Feature: Sync a file removed on one computer and changed on the other

  Numen Sync keeps a changed file when the other computer removes it.

  As a person who uses Numen on more than one computer
  I want a removal never to destroy a change I made
  So that I lose a text only when I remove it myself

  Background:
    Given the person has the synced vault "Test" on two computers

  Rule: A change wins over a removal

    @own-server
    Scenario: A note changed on one computer and removed on the other stays
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer removes the file "Shopping list.md" in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has the note "Shopping list" with the text "bread"
      And the second computer has the note "Shopping list" with the text "bread" in its vault "Copy"

    @own-server
    Scenario: A note removed on one computer and changed on the other stays
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the sync server stops
      And the person removes the file "Shopping list.md" in the vault "Test"
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has the note "Shopping list" with the text "eggs"
      And the vault "Test" has no conflict copy of the note "Shopping list"

    @own-server
    Scenario: A file changed on one computer and removed on the other stays
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the sync server stops
      And the person replaces the file "Papers/Report.pdf" with a new version in the vault "Test"
      And the second computer removes the file "Papers/Report.pdf" in its vault "Copy"
      When the sync server runs again
      Then the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the vault "Test" has the same file "Papers/Report.pdf" as the second computer

  Rule: A file removed on both computers stays removed

    @own-server
    Scenario: A file removed on both computers is removed
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the sync server stops
      And the person removes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer removes the file "Papers/Report.pdf" in its vault "Copy"
      When the sync server runs again
      Then the person does not have the file "Papers/Report.pdf" in the vault "Test"
      And the second computer does not have the file "Papers/Report.pdf" in its vault "Copy"
