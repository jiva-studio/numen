Feature: Numen Sync after the subscription ends

  Changes are sent for 7 days after the end of the subscription. Then the vaults in the cloud can only be read.
  The cloud deletes them 60 days after the subscription ends.

  As a person who uses Numen Sync
  I want to know when my notes stop leaving this computer, and why
  So that I can buy the subscription again before I lose changes

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the subscription
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: Changes are sent for 7 days after the end

    Scenario: A note is sent after the subscription ended
      Given the subscription ended 3 days ago
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"

    Scenario: The page warns that the subscription ended
      When the subscription ended 3 days ago
      And the person opens the page "Identity and licenses"
      Then the person is told "Ended"
      And the page offers to buy the subscription

  Rule: Changes are not sent after the 7 days

    Scenario: A note stays on this computer
      Given the subscription ended 10 days ago
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the person is told "Subscription ended: changes stay on this device"
      And the second computer does not have the file "Shopping list.md" in its vault "Copy"

    Scenario: The page tells that the vaults can only be read
      When the subscription ended 10 days ago
      And the person opens the page "Identity and licenses"
      Then the person is told "Read only"

    Scenario: The page tells that the cloud deleted the vaults
      When the subscription ended 70 days ago
      And the person opens the page "Identity and licenses"
      Then the person is told "Deleted"

  Rule: A new subscription opens the writes again

    Scenario: A note is sent after the subscription is bought again
      Given the subscription ended 10 days ago
      And the person opens the page "Identity and licenses"
      And the person buys the subscription
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
