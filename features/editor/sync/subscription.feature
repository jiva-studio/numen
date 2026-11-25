Feature: Numen Sync subscription

  One subscription belongs to the identity of a person. It covers every vault and every computer of the identity.

  As a person who uses Numen Sync
  I want to see and buy my subscription in one place
  So that I know why my notes are sent or not

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person creates an identity
    And the application "editor" starts

  Rule: The page "Identity and licenses" shows the subscription

    Scenario: A new identity has no subscription
      When the person opens the page "Identity and licenses"
      Then the page offers to buy the subscription

    Scenario: A payment that is not made leaves the identity without a subscription
      Given the person opens the page "Identity and licenses"
      When the person opens the payment page without paying
      Then the page offers to buy the subscription

  Rule: The subscription needs an identity and a server that answers

    Scenario: A subscription is not offered without an identity
      Given the person deletes the identity from this device
      When the person opens the page "Identity and licenses"
      Then the person is told "Sync your vaults between your devices"
      And the person is told "Needs an identity"
      And the page does not offer to buy the subscription

    Scenario: The page tells when the server is away
      Given the sync server stops
      When the person opens the page "Identity and licenses"
      Then the person is told "Offline"
