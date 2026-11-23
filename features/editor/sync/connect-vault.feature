Feature: Connect a vault to Numen Sync

  A person connects a vault to their account as a new synced vault, or to a synced vault that exists.

  As a person who uses Numen on more than one computer
  I want to choose which vault a folder syncs with
  So that the same notes are on every computer

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts

  Rule: A vault is connected as a new synced vault or to an existing one

    Scenario: A new synced vault is listed
      When the person connects the vault "Test" as a new synced vault
      Then the sync settings list the synced vault "Test"

    Scenario: A second computer lists the synced vaults of the account
      Given the person connects the vault "Test" as a new synced vault
      When a second computer restores the identity from the recovery words
      Then the second computer can connect to the synced vault "Test"

    Scenario: A folder joins an existing synced vault
      Given the person connects the vault "Test" as a new synced vault
      And a second computer restores the identity from the recovery words
      When the second computer connects its vault "Copy" to the synced vault "Test"
      Then the sync settings of the second computer show the vault "Copy" as connected

    Scenario: A renamed vault stays connected
      Given the person connects the vault "Test" as a new synced vault
      When the person renames the vault "Test" to "Work"
      Then the sync settings list the synced vault "Work"
      And the vault "Work" is connected

  Rule: A copy of a connected vault is not connected twice

    Scenario: A copy of a connected folder is refused
      Given the person connects the vault "Test" as a new synced vault
      And the person adds a copy of the vault "Test" as the vault "Copy"
      When the person connects the vault "Copy" as a new synced vault
      Then the sync settings warn about "a copy of a connected vault"

  Rule: Numen Sync needs an identity to connect

    Scenario: A vault is not connected without an identity
      Given the person deletes the identity from this device
      When the person opens the sync settings
      Then the sync settings warn about "No identity on this device"
      And the vault "Test" is not connected
