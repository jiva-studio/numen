@live
Feature: Numen Sync after the cloud deleted the vaults

  The cloud deletes the vaults 60 days after the subscription ends. The notes on the computers stay.

  As a person who uses Numen Sync
  I want my notes to stay on my computer when the cloud deletes my vault
  So that a new subscription sends them again

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person has an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the subscription
    And the person syncs the vault "Test"
    And the person writes the note "Shopping list" with the text "milk" in the vault "Test"

  Rule: A deleted vault is not a deletion of notes

    @own-server
    Scenario: The notes stay on the computer
      When the subscription ended 70 days ago
      Then the vault "Test" has the note "Shopping list" with the text "milk"

  Rule: A new subscription sends the vault again

    @own-server
    Scenario: The vault is sent again after the subscription is bought again
      Given the subscription ended 70 days ago
      And the person opens the page "Identity and licenses"
      When the person buys the subscription
      Then the synced vault "Test" holds the note "Shopping list" with the text "milk"
