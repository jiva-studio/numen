Feature: Numen Sync plan change confirmation

  The page asks before it moves an active subscription to another plan, and changes nothing when the person cancels.

  As a person who uses Numen Sync
  I want to see what a plan change charges before it is made
  So that I do not change my plan by mistake

  Background:
    Given the empty vault "Test"
    And a sync server that sells the plans "Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the plan "Basic"

  Scenario: The page asks before it switches the plan
    When the person chooses the plan "Pro" in Change plan
    Then the page says "Switch to Pro?"
    And the page says "Charged now"

  Scenario: The switch is made once the person confirms it
    Given the person chooses the plan "Pro" in Change plan
    When the person confirms the switch to "Pro"
    Then the subscription shows the plan "Pro" with 250 GB

  Scenario: A cancelled switch keeps the plan
    When the person chooses the plan "Pro" in Change plan
    And the person cancels the switch
    Then the subscription shows the plan "Basic" with 10 GB
