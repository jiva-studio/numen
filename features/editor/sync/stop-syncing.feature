Feature: Stop syncing a vault

  A person stops syncing a vault on one device. The notes stay on the device and in the cloud.

  As a person who keeps notes in Numen
  I want to stop syncing a vault on this device
  So that I keep my notes and the vault is not sent or changed any more

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And the person writes the note "Food/Shopping list" with the text "milk" in the vault "Test"

  Rule: The files stay

    Scenario: The vault is not synced any more and keeps its notes
      When the person stops syncing the vault "Test"
      Then the vault "Test" is not synced
      And the person has the note "Shopping list" with the text "milk" in the vault "Test"
