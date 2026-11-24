Feature: Numen Sync on a phone

  A phone syncs while Numen is open and sends what the person wrote when Numen leaves the screen.

  As a person who uses Numen on a phone
  I want my notes to move when I use the phone
  So that the phone and my computer have the same notes

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person connects the vault "Test" as a new synced vault
    And the phone is connected to the synced vault "Test"

  Rule: A phone that is open gets the changes of the vault

    Scenario: A note written on the computer arrives while Numen is open on the phone
      Given Numen is open on the phone
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the phone has the note "Shopping list" with the text "milk"

    Scenario: A note written while Numen is closed arrives when it opens
      Given Numen is closed on the phone
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      When Numen opens on the phone
      Then the phone has the note "Shopping list" with the text "milk"

  Rule: A phone that is closed does not use the network

    Scenario: Numen asks for nothing when it leaves the screen
      Given Numen is open on the phone
      When Numen leaves the screen of the phone
      Then the sync server has no open stream for the phone

    Scenario: Numen asks again when it comes back
      Given Numen leaves the screen of the phone
      When Numen opens on the phone
      Then the sync server has an open stream for the phone

  Rule: A note written on the phone is sent

    Scenario: A note is sent when Numen leaves the screen
      Given Numen is open on the phone
      And the person writes the note "Shopping list" with the text "bread" on the phone
      When Numen leaves the screen of the phone
      Then the vault "Test" has the note "Shopping list" with the text "bread"

    Scenario: A note written without the network is sent later
      Given the sync server stops
      And the person writes the note "Shopping list" with the text "eggs" on the phone
      When the sync server runs again
      Then the vault "Test" has the note "Shopping list" with the text "eggs"
