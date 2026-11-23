Feature: Numen Sync subscription

  One subscription belongs to the identity of a person. It covers every vault and every computer of the account.

  As a person who uses Numen Sync
  I want to see and buy my subscription in one place
  So that I know why my notes are sent or not

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person creates an identity
    And the application "editor" starts

  Rule: The page "Identity and licenses" shows the subscription of the account

    Scenario: A new account has no subscription
      When the person opens the page "Identity and licenses"
      Then the page offers to buy the subscription

    Scenario: A paid subscription is active
      Given the person opens the page "Identity and licenses"
      When the person buys the subscription
      Then the subscription is active

    Scenario: A subscription bought on one computer is active on the other
      Given a second computer restores the identity from the recovery words
      And the person opens the page "Identity and licenses"
      When the person buys the subscription
      Then the subscription is active on the second computer

    Scenario: An active subscription is not bought twice
      Given the person opens the page "Identity and licenses"
      When the person buys the subscription
      Then the page does not offer to buy the subscription

    Scenario: A payment that is not made leaves the account without a subscription
      Given the person opens the page "Identity and licenses"
      When the person opens the payment page without paying
      Then the page offers to buy the subscription

  Rule: The subscription needs an identity and a server that answers

    Scenario: A subscription is not offered without an identity
      Given the person deletes the identity from this device
      When the person opens the page "Identity and licenses"
      Then the page says "Numen Sync needs an identity to tell your devices apart"
      And the page does not offer to buy the subscription

    Scenario: The page tells when the server is away
      Given the sync server stops
      When the person opens the page "Identity and licenses"
      Then the subscription shows "Offline"
