Feature: Numen Sync plan change

  A person moves an active subscription to a larger or a smaller plan. The notes in the cloud are never deleted by a change.

  As a person who uses Numen Sync
  I want to change my plan when my notes grow or shrink
  So that I do not pay for space I do not use and do not stop at a full cloud

  Background:
    Given the empty vault "Test"
    And a sync server that sells the plans "Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the plan "Basic"
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: A larger plan opens space at once

    Scenario: The page shows the larger plan after the change
      When the person changes the plan to "Pro"
      Then the subscription shows the plan "Pro" with 250 GB

    Scenario: A note is sent after a full plan is made larger
      Given the cloud holds 10 GB of the notes of the person
      When the person changes the plan to "Plus"
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

  Rule: Only an active subscription changes its plan

    Scenario: The page shows the plan that is bought as current and does not offer it
      When the person opens the page "Identity and licenses"
      Then the page offers to change the plan to "Plus"
      And the page shows the plan "Basic" as the current plan
      And the page does not offer to change the plan to "Basic"

    Scenario: An ended subscription offers to buy, not to change
      Given the subscription ended 3 days ago
      When the person opens the page "Identity and licenses"
      Then the page offers to buy the plan "Plus" with 100 GB
      And the page does not offer to change the plan to "Plus"
