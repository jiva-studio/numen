@sync-server
Feature: Sync large files between computers

  Numen Sync sends a large file and goes on after a break.

  As a person who keeps big files in a vault
  I want a big file to arrive even when the connection breaks
  So that I do not send it again from the start

  Background:
    Given the person has the synced vault "Test" on two computers

  Rule: A large file arrives whole

    Scenario: A large file appears on the second computer
      When the person adds the file "Videos/Trip.mp4" of 70 MB to the vault "Test"
      Then the second computer has the same file "Videos/Trip.mp4" in its vault "Copy"

    @own-server
    Scenario: A file that is not complete is not in the vault
      Given the connection breaks while the second computer receives the file "Videos/Trip.mp4"
      When the person adds the file "Videos/Trip.mp4" of 70 MB to the vault "Test"
      Then the second computer does not have the file "Videos/Trip.mp4" in its vault "Copy"

  Rule: A break does not lose the file

    @own-server
    Scenario: A break while the file is sent
      Given the connection breaks while the file "Videos/Trip.mp4" is sent
      And the person adds the file "Videos/Trip.mp4" of 70 MB to the vault "Test"
      When the sync server runs again
      Then the second computer has the same file "Videos/Trip.mp4" in its vault "Copy"

    @own-server
    Scenario: A break while the file is received
      Given the person adds the file "Videos/Trip.mp4" of 70 MB to the vault "Test"
      And the connection breaks while the second computer receives the file "Videos/Trip.mp4"
      When the sync server runs again
      Then the second computer has the same file "Videos/Trip.mp4" in its vault "Copy"

  Rule: A file that changes while it goes up is sent again

    @own-server
    Scenario: A file changed after a break is sent whole
      Given the connection breaks while the file "Videos/Trip.mp4" is sent
      And the person adds the file "Videos/Trip.mp4" of 70 MB to the vault "Test"
      And the person replaces the file "Videos/Trip.mp4" with a new version in the vault "Test"
      When the sync server runs again
      Then the second computer has the same file "Videos/Trip.mp4" in its vault "Copy"
