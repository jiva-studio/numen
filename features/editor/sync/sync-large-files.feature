Feature: Sync large files between computers

  Numen Sync sends a large file and goes on after a break.

  As a person who keeps big files in a vault
  I want a big file to arrive even when the connection breaks
  So that I do not send it again from the start

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: A large file arrives whole

    Scenario: A large file appears on the second computer
      When the person writes the file "Videos/Trip.bin" of 70 MB in the vault "Test"
      Then the second computer has the same file "Videos/Trip.bin" in its vault "Copy"

    Scenario: A file that is not complete is not in the vault
      Given the connection breaks while the second computer receives the file "Videos/Trip.bin"
      When the person writes the file "Videos/Trip.bin" of 70 MB in the vault "Test"
      Then the second computer does not have the file "Videos/Trip.bin" in its vault "Copy"

  Rule: A break does not lose the file

    Scenario: A break while the file is sent
      Given the connection breaks while the file "Videos/Trip.bin" is sent
      And the person writes the file "Videos/Trip.bin" of 70 MB in the vault "Test"
      When the sync server runs again
      Then the second computer has the same file "Videos/Trip.bin" in its vault "Copy"

    Scenario: A break while the file is received
      Given the person writes the file "Videos/Trip.bin" of 70 MB in the vault "Test"
      And the connection breaks while the second computer receives the file "Videos/Trip.bin"
      When the sync server runs again
      Then the second computer has the same file "Videos/Trip.bin" in its vault "Copy"

  Rule: A file that changes while it goes up is sent again

    Scenario: A file changed after a break is sent whole
      Given the connection breaks while the file "Videos/Trip.bin" is sent
      And the person writes the file "Videos/Trip.bin" of 70 MB in the vault "Test"
      And the person changes the file "Videos/Trip.bin" in the vault "Test"
      When the sync server runs again
      Then the second computer has the same file "Videos/Trip.bin" in its vault "Copy"
