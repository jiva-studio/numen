Feature: Numen Sync plan and the notes in the cloud

  A plan sets how much the cloud holds. A change of the plan never deletes the notes in the cloud.

  As a person who uses Numen Sync
  I want my notes to stay in the cloud when I change my plan
  So that I do not lose notes and do not stop at a full cloud

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

  Rule: A larger plan opens space

    Scenario: A note is sent after a full plan is made larger
      Given the cloud holds 10 GB
      When the person changes the plan to "Plus"
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

  Rule: A smaller plan never deletes notes

    Scenario: A plan smaller than the notes keeps them in the cloud
      Given the person writes the note "Archive" with the text "old" in the vault "Test"
      And the person changes the plan to "Pro"
      And the cloud holds 20 GB
      When the person changes the plan to "Basic"
      Then the person is told "20 GB of 10 GB"
      And the second computer has the note "Archive" with the text "old" in its vault "Copy"

    Scenario: Changes are sent again when the notes fit the plan
      Given the person changes the plan to "Pro"
      And the cloud holds 20 GB
      And the person changes the plan to "Basic"
      When 15 GB are removed from the cloud
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
