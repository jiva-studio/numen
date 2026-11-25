Feature: Pause the sync when many files are removed

  Numen Sync asks before it sends the removal of many files.

  As a person who removes a folder by mistake
  I want Numen Sync to ask before it removes the files on the other computer
  So that one slip does not empty both computers

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: Numen Sync asks before it sends the removal of many files

    Scenario: Many removed files pause the sync
      Given the person adds 10 files to the folder "Papers" in the vault "Test"
      And the second computer has 10 files in its folder "Papers" in its vault "Copy"
      When the person removes the folder "Papers" in the vault "Test"
      Then the person is told "Sync is paused: 10 of 10 files are removed"
      And the second computer has 10 files in its folder "Papers" in its vault "Copy"

    Scenario: The person confirms the removal of many files
      Given the person adds 10 files to the folder "Papers" in the vault "Test"
      And the second computer has 10 files in its folder "Papers" in its vault "Copy"
      And the person removes the folder "Papers" in the vault "Test"
      When the person chooses "Remove the files"
      Then the second computer has 0 files in its folder "Papers" in its vault "Copy"

    Scenario: The person keeps the files
      Given the person adds 10 files to the folder "Papers" in the vault "Test"
      And the second computer has 10 files in its folder "Papers" in its vault "Copy"
      And the person removes the folder "Papers" in the vault "Test"
      When the person chooses "Keep the files"
      Then the vault "Test" has 10 files in its folder "Papers"
      And the second computer has 10 files in its folder "Papers" in its vault "Copy"

    Scenario: A few removed files do not pause the sync
      Given the person adds 10 files to the folder "Papers" in the vault "Test"
      And the second computer has 10 files in its folder "Papers" in its vault "Copy"
      When the person removes the file "Papers/File 1.bin" in the vault "Test"
      Then the second computer has 9 files in its folder "Papers" in its vault "Copy"
