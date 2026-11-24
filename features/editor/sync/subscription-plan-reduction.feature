Feature: Numen Sync smaller plan

  A person moves an active subscription to a smaller plan. The notes in the cloud are never deleted by a change.

  As a person who uses Numen Sync
  I want to move to a smaller plan when my notes shrink
  So that I do not pay for space I do not use and do not lose notes

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
