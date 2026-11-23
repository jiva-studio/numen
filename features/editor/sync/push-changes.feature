Feature: Receive changes within seconds

  The sync server tells each computer when the account has a change, so a computer does not wait for its next look.

  As a person who uses Numen on more than one computer
  I want a note I write on one computer to reach the other in seconds
  So that I see my work on both computers without waiting

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person connects the vault "Test" as a new synced vault
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"
    And both computers check for changes only every "1 hour"

  Rule: A change reaches the other computer within seconds

    Scenario: A new note appears on the second computer
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" within "5" seconds

    Scenario: A changed note changes on the second computer
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" within "5" seconds
      When the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "bread" within "5" seconds

    Scenario: A note written on the second computer appears on the first
      When the second computer writes the note "Plan" with the text "tea" in its vault "Copy"
      Then the vault "Test" has the note "Plan" with the text "tea" within "5" seconds

  Rule: A computer receives changes again after its connection drops

    Scenario: A note written after the connection drops still arrives
      Given the second computer goes offline for a moment
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" within "15" seconds

  Rule: A computer receives changes again after the server was away

    Scenario: A note written after the server runs again arrives
      Given the sync server stops
      And the sync server runs again
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" within "15" seconds
