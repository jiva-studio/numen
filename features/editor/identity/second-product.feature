Feature: A second product on the same account

  A person who bought a plan of Numen Sync can buy another product with the same identity. A product that is
  paid once stays with the account for good.

  As a person who uses Numen
  I want to buy another product with the identity I already have
  So that I need no new account, no key and no second sign-in

  Background:
    Given a sync server that sells the plans "Basic" with 10 GB and "Plus" with 100 GB runs
    And the server sells the product "Extra" for a single payment
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"

  Rule: A product paid once is bought once and belongs to the account

    Scenario: The page offers the product next to the plans
      Then the page offers to buy the plan "Basic" with 10 GB
      And the page offers to buy the product "Extra"

    Scenario: A product bought once is shown as owned
      When the person buys the product "Extra"
      Then the page shows that the product "Extra" is owned

    Scenario: A product that is owned is not offered again
      When the person buys the product "Extra"
      Then the page does not offer to buy the product "Extra"

    Scenario: A product bought on one computer is shown on the other
      Given a second computer restores the identity from the recovery words
      When the person buys the product "Extra"
      Then the second computer shows that the product "Extra" is owned

    Scenario: Buying the product leaves the subscription of Numen Sync as it was
      Given the person buys the plan "Plus"
      When the person buys the product "Extra"
      Then the subscription shows the plan "Plus" with 100 GB

    Scenario: A refunded product is offered again
      Given the person buys the product "Extra"
      When the payment of the product "Extra" is refunded
      Then the page offers to buy the product "Extra"

  Rule: A trial shows its end

    Scenario: A subscription that starts with a trial shows the day it ends
      When the person starts a trial of the plan "Plus"
      Then the subscription shows "Trial until" the day the trial ends

    Scenario: A trial that is canceled keeps the access until it ends
      Given the person starts a trial of the plan "Plus"
      When the person cancels the trial
      Then the subscription shows the plan "Plus" with 100 GB
