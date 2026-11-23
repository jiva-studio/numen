Feature: Numen Sync after the server is restored from a backup

  The operator can restore the sync server to an earlier backup. The computers keep their notes and send them again.

  As a person who uses Numen Sync
  I want my notes to stay as they are on my computers when the server loses its newest data
  So that a restore of the server never takes my work back

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the subscription
    And the person connects the vault "Test" as a new synced vault
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: A restored server gets the notes of the computers back

    Scenario: A note written after the backup returns to the server
      Given the operator makes a backup of the sync server
      And the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      When the operator restores the sync server from the backup
      And the operator announces the restore
      Then the synced vault "Test" holds the note "Shopping list" with the text "milk"

    Scenario: A restored server does not bring back an old text
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the operator makes a backup of the sync server
      And the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "bread" in its vault "Copy"
      When the operator restores the sync server from the backup
      And the operator announces the restore
      Then the vault "Test" has the note "Shopping list" with the text "bread"
      And the second computer has the note "Shopping list" with the text "bread" in its vault "Copy"
      And the synced vault "Test" holds the note "Shopping list" with the text "bread"

    Scenario: A note removed after the backup stays removed
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the operator makes a backup of the sync server
      And the person removes the note "Shopping list" in the vault "Test"
      And the second computer does not have the file "Shopping list.md" in its vault "Copy"
      When the operator restores the sync server from the backup
      And the operator announces the restore
      Then the synced vault "Test" does not hold the note "Shopping list"
      And the second computer does not have the file "Shopping list.md" in its vault "Copy"

    Scenario: A computer that was away sends its note after the restore
      Given the operator makes a backup of the sync server
      And the second computer goes offline
      And the second computer writes the note "Shopping list" with the text "eggs" in its vault "Copy"
      And the operator restores the sync server from the backup
      And the operator announces the restore
      When the second computer goes online
      Then the synced vault "Test" holds the note "Shopping list" with the text "eggs"

  Rule: The list of synced vaults comes back

    Scenario: A vault connected after the backup is listed again
      Given the operator makes a backup of the sync server
      And the person connects the vault "Work" as a new synced vault
      When the operator restores the sync server from the backup
      And the operator announces the restore
      Then the second computer can connect to the synced vault "Work"

  Rule: The subscription stays after a restore

    Scenario: A subscription bought after the backup is active
      Given the operator makes a backup of the sync server
      And the person opens the page "Identity and licenses"
      And the person buys the subscription again
      When the operator restores the sync server from the backup
      And the operator announces the restore
      Then the subscription shows "Active"

  Rule: A restore that is not announced changes nothing on the computers

    Scenario: A computer waits for the announcement
      Given the person writes the note "Shopping list" with the text "bread" in the vault "Test"
      And the operator makes a backup of the sync server
      And the person writes the note "Shopping list" with the text "eggs" in the vault "Test"
      When the operator restores the sync server from the backup
      Then the plugin "Numen Sync" warns about "The sync server went back to an earlier state"
      And the vault "Test" has the note "Shopping list" with the text "eggs"
