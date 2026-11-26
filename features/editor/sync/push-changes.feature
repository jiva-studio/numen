@sync-server
Feature: Receive changes within seconds

  A change reaches the other computer within seconds, also when the computers do not look for changes by themselves.

  As a person who uses Numen on more than one computer
  I want a note I write on one computer to reach the other in seconds
  So that I see my work on both computers without waiting

  Background:
    Given the person has the synced vault "Test" on two computers
    And the computers do not look for changes by themselves

  Rule: A change reaches the other computer within seconds

    Scenario: A new note appears on the second computer
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" within "5" seconds

  Rule: A computer receives changes again after its connection drops

    @own-server
    Scenario: A note written after the connection drops still arrives
      Given the second computer goes offline for a moment
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" within "15" seconds

  Rule: A computer receives changes again after the server was away

    @own-server
    Scenario: A note written after the server runs again arrives
      Given the sync server stops
      And the sync server runs again
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" within "15" seconds
