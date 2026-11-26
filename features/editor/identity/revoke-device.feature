@sync-server
Feature: Revoke a device

  A person removes a lost device from the identity. The removed device loses its access.

  As a person who uses Numen on more than one device
  I want to revoke a device I lost
  So that nobody who has it can read or write my notes

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the second computer is linked to the identity of the person

  Rule: A revoked device leaves the identity

    Scenario: A revoked device leaves the list of devices
      When the person revokes the second computer
      Then the identity has 1 device

    Scenario: A device does not revoke itself
      When the person opens the page "Identity and licenses"
      Then the person cannot revoke this device

    Scenario: A revoked device is no longer let in by the sync server
      When the person revokes the second computer
      Then the sync server no longer lets the second computer in

  Rule: A revoked device says it was revoked and keeps its files

    Scenario: A revoked device says it was revoked
      When the person revokes the second computer
      Then the second computer is told "This device was revoked"

    Scenario: A revoked device keeps its local notes
      Given the second computer writes the note "Eggs" with the text "eggs" in its vault "Copy"
      When the person revokes the second computer
      Then the second computer has the note "Eggs" with the text "eggs" in its vault "Copy"
