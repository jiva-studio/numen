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
    And the person connects the vault "Test" as a new synced vault
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: The page asks before it changes the plan

    Scenario: The page asks before it makes the switch
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

  Rule: A larger plan opens space at once

    Scenario: The page shows the larger plan after the change
      When the person changes the plan to "Pro"
      Then the subscription shows the plan "Pro" with 250 GB

    Scenario: A note is sent after a full plan is made larger
      Given the cloud holds 10 GB of the notes of the person
      When the person changes the plan to "Plus"
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

  Rule: A smaller plan never deletes notes

    Scenario: The page shows the smaller plan after the change
      Given the person changes the plan to "Pro"
      When the person changes the plan to "Plus"
      Then the subscription shows the plan "Plus" with 100 GB

    Scenario: A plan smaller than the notes keeps them in the cloud
      Given the person writes the note "Archive" with the text "old" in the vault "Test"
      And the person changes the plan to "Pro"
      And the cloud holds 20 GB of the notes of the person
      When the person changes the plan to "Basic"
      Then the subscription shows "20 GB of 10 GB"
      And the second computer has the note "Archive" with the text "old" in its vault "Copy"

    Scenario: A plan smaller than the notes stops new changes
      Given the person changes the plan to "Pro"
      And the cloud holds 20 GB of the notes of the person
      And the person changes the plan to "Basic"
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the plugin "Numen Sync" warns about "Cloud storage is full: changes stay on this device"
      And the second computer does not have the file "Shopping list.md" in its vault "Copy"

    Scenario: Changes are sent again when the notes fit the plan
      Given the person changes the plan to "Pro"
      And the cloud holds 20 GB of the notes of the person
      And the person changes the plan to "Basic"
      When the person deletes 15 GB of the notes of the person from the cloud
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
