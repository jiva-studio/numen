Feature: Numen Sync plans

  A subscription has a plan. The plan sets how much the cloud holds for the identity.

  As a person who uses Numen Sync
  I want to choose a plan and see how much of it I use
  So that I pay for the space I need and know when it runs out

  Background:
    Given the empty vault "Test"
    And a sync server that sells the plans "Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB runs
    And the person has an identity
    And the application "editor" starts

  Rule: The page "Identity and licenses" offers plans and shows the one bought

    Scenario: A new identity is offered every plan with its size
      When the person opens the page "Identity and licenses"
      Then the page offers to buy the plan "Basic" with 10 GB
      And the page offers to buy the plan "Plus" with 100 GB
      And the page offers to buy the plan "Pro" with 250 GB

    Scenario: The bought plan is shown with its size
      Given the person opens the page "Identity and licenses"
      When the person buys the plan "Plus"
      Then the subscription is active
      And the subscription shows the plan "Plus" with 100 GB

    Scenario: A plan bought on one computer is shown on the other
      Given a second computer restores the identity from the recovery words
      And the person opens the page "Identity and licenses"
      When the person buys the plan "Plus"
      Then the subscription shows the plan "Plus" with 100 GB on the second computer

    Scenario: An active subscription is not bought twice
      Given the person opens the page "Identity and licenses"
      When the person buys the plan "Basic"
      Then the page does not offer to buy a plan

    Scenario: An ended subscription still shows its plan
      Given the person opens the page "Identity and licenses"
      And the person buys the plan "Plus"
      When the subscription ended 3 days ago
      And the person opens the page "Identity and licenses"
      Then the person is told "Ended"
      And the subscription shows the plan "Plus" with 100 GB

  Rule: The page shows how much of the plan is used

    Scenario: The used space is shown against the plan
      Given the person opens the page "Identity and licenses"
      And the person buys the plan "Plus"
      When the cloud holds 2 GB
      And the person opens the page "Identity and licenses"
      Then the person is told "2 GB of 100 GB"

  Rule: A full plan stops new changes

    Scenario: A note stays on this computer when the plan is full
      Given the person opens the page "Identity and licenses"
      And the person buys the plan "Basic"
      And the person syncs the vault "Test"
      And a second computer restores the identity from the recovery words
      And the second computer connects its vault "Copy" to the synced vault "Test"
      And the cloud holds 10 GB
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the person is told "Cloud storage is full: changes stay on this device"
      And the second computer does not have the file "Shopping list.md" in its vault "Copy"
