Feature: Sync notes when the computers are apart

  Numen Sync keeps every note when the server is away or when two computers change the same note.

  As a person who uses Numen on more than one computer
  I want no change of mine to be lost when the computers cannot agree at once
  So that I can trust the sync

  Background:
    Given the person has the synced vault "Test" on two computers

  Rule: Numen Sync keeps working when the server is away

    Scenario: A note written while the server is away arrives later
      Given the sync server stops
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      When the sync server runs again
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

    Scenario: Numen Sync warns while the server is away
      When the sync server stops
      Then the person is told "Could not reach the sync server"
