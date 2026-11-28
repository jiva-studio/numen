Feature: Numen Sync plan change

  A person moves an active subscription to a larger or a smaller plan, and is asked before the change is made.

  As a person who uses Numen Sync
  I want to change my plan when my notes grow or shrink
  So that I do not pay for space I do not use and do not change my plan by mistake

  Background:
    Given the empty vault "Test"
    And a sync server that sells the plans "Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB runs
    And the person has an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the plan "Basic"

  Rule: An active subscription changes its plan

    @live
    Scenario: The subscription shows the smaller plan after the change
      Given the person changes the plan to "Pro"
      When the person changes the plan to "Plus"
      Then the subscription shows the plan "Plus" with 100 GB

    @live
    Scenario: The plan that is bought is shown as current and is not offered
      When the person opens the page "Identity and licenses"
      Then the page offers to change the plan to "Plus"
      And the page shows the plan "Basic" as the current plan
      And the page does not offer to change the plan to "Basic"

    @stand
    Scenario: An ended subscription offers to buy, not to change
      Given the subscription ended 3 days ago
      When the person opens the page "Identity and licenses"
      Then the page offers to buy the plan "Plus" with 100 GB
      And the page does not offer to change the plan to "Plus"

  Rule: The person confirms the change

    @live
    Scenario: The person is asked before the plan changes
      When the person wants to change the plan to "Pro"
      Then the person is told "Switch to Pro?"
      And the person is told "Charged now"

    @live
    Scenario: The plan changes once the person confirms
      Given the person wants to change the plan to "Pro"
      When the person confirms the switch to "Pro"
      Then the subscription shows the plan "Pro" with 250 GB

    @live
    Scenario: A cancelled change keeps the plan
      When the person wants to change the plan to "Pro"
      And the person cancels the switch
      Then the subscription shows the plan "Basic" with 10 GB
