Feature: Connect a folder that holds notes

  Numen Sync joins the notes of a folder with the notes of a synced vault and keeps every note.

  As a person who uses Numen on more than one computer
  I want to connect a folder that already holds notes to a synced vault
  So that I do not lose notes or get the same note twice

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person has an identity
    And the application "editor" starts
    And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words

  Rule: A note with the same text on both sides stays one note

    Scenario: A note with the same text is not copied
      Given the second computer writes the note "Shopping list" with the text "milk" in its vault "Copy"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the second computer has no conflict copy of the note "Shopping list" in its vault "Copy"

    Scenario: A note with the same text asks nothing
      Given the second computer writes the note "Shopping list" with the text "milk" in its vault "Copy"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      Then the connection of the second computer needs no confirmation

  Rule: A note on one side only goes to the other side

    Scenario: A note of the folder goes to the synced vault
      Given the second computer writes the note "Plan" with the text "tea" in its vault "Copy"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      Then the vault "Test" has the note "Plan" with the text "tea"

    Scenario: A note of the synced vault comes to the folder
      Given the second computer writes the note "Plan" with the text "tea" in its vault "Copy"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

  Rule: A note with another text on each side is kept in both texts

    Scenario: Numen Sync asks before it joins notes with other texts
      Given the second computer writes the note "Shopping list" with the text "bread" in its vault "Copy"
      And the second computer writes the note "Plan" with the text "tea" in its vault "Copy"
      And the person writes the note "Plan" with the text "eggs" in the vault "Test"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      Then the second computer is asked to confirm "2 notes differ from the vault in the cloud. Get it anyway?"

    Scenario: A confirmed note keeps both texts
      Given the second computer writes the note "Shopping list" with the text "bread" in its vault "Copy"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      And the second computer confirms the connection
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      And the second computer has a conflict copy of the note "Shopping list" with the text "bread" in its vault "Copy"

    Scenario: A refused connection changes nothing
      Given the second computer writes the note "Shopping list" with the text "bread" in its vault "Copy"
      When the second computer connects its vault "Copy" to the synced vault "Test"
      And the second computer cancels the connection
      Then the second computer has the note "Shopping list" with the text "bread" in its vault "Copy"
      And the sync settings of the second computer do not list the vault "Copy"
