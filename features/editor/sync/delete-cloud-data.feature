Feature: Delete cloud data

  The person can delete everything Numen Sync holds of the account, apart from deleting the identity on this computer. The server keeps
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
      And the question says "Type the account fingerprint"

    Scenario: A text that is not the account fingerprint deletes nothing
      When the person asks to delete the cloud data and types "somebody else"
      Then the page says "This is not the account fingerprint."
      And the cloud data is not being deleted

  Rule: The cloud data is kept for 7 days

    Scenario: The card shows that the cloud data is being deleted
      When the person deletes the cloud data
      Then the cloud data is being deleted
      And the page says "Restore account"

    Scenario: Restoring the account stops the deletion
      Given the person deletes the cloud data
      When the person restores the account
      Then the cloud data is not being deleted

    Scenario: Deleting the cloud data cancels the subscription
      Given the person buys the subscription
      And the person deletes the cloud data
      When the person restores the account
      Then the page offers to buy the subscription

  Rule: The person can see what the server holds

    Scenario: The account data is saved as a file
      When the person exports the account data
      Then the saved file names the account of the person
