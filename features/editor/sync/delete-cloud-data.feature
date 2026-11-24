Feature: Delete cloud data

  The person can delete everything Numen Sync holds of the identity, apart from deleting the identity on this computer. The server keeps
  the data for 7 days, so a mistake can be undone, and then deletes it for good. The notes on the computers stay.

  As a person who uses Numen Sync
  I want to delete my cloud data and see what the server holds
  So that I stay in charge of my data

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"

  Rule: Deleting the cloud data asks first

    Scenario: The question says what is deleted and what stays
      When the person asks to delete the cloud data
      Then the question says "7 days"
      And the question says "Your notes stay on your devices"

    Scenario: Cancelling the question deletes nothing
      Given the person asks to delete the cloud data
      When the person cancels the deletion of the cloud data
      Then the cloud data is kept

  Rule: The cloud data is kept for 7 days

    Scenario: The card shows that the cloud data is being deleted
      When the person deletes the cloud data
      Then the cloud data is being deleted
      And the page says "Restore cloud data"

    Scenario: Restoring the cloud data stops the deletion
      Given the person deletes the cloud data
      When the person restores the cloud data
      Then the cloud data is kept

    Scenario: Deleting the cloud data cancels the subscription
      Given the person buys the subscription
      And the person deletes the cloud data
      When the person restores the cloud data
      Then the page offers to buy the subscription

  Rule: The person can see what the server holds

    Scenario: The identity data is saved as a file
      When the person exports the identity data
      Then the saved file names the identity of the person
