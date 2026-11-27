Feature: Sync a file changed on two computers

  Numen Sync keeps the latest version of a file that is not a note and does not keep the other version.

  As a person who uses Numen on more than one computer
  I want both computers to hold the same file after two changes
  So that a file is never different on two computers

  Background:
    Given the person has the synced vault "Test" on two computers

  Rule: The latest version of a file replaces the other version

    @stand
    Scenario: A file changed on two computers keeps the latest version
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the sync server stops
      And the person replaces the file "Papers/Report.pdf" with a new version in the vault "Test"
      And the second computer replaces the file "Papers/Report.pdf" with a new version in its vault "Copy"
      And the second computer is the first to reach the sync server
      When the sync server runs again
      Then the vault "Test" has the same file "Papers/Report.pdf" as the second computer

    @stand
    Scenario: A file changed on two computers has no copy
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the sync server stops
      And the person replaces the file "Papers/Report.pdf" with a new version in the vault "Test"
      And the second computer replaces the file "Papers/Report.pdf" with a new version in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has 1 file in its folder "Papers"
      And the second computer has 1 file in its folder "Papers" in its vault "Copy"

  Rule: A rename is not a conflict

    @stand
    Scenario: A file renamed on one computer and changed on the other keeps the change
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the sync server stops
      And the person moves the file "Papers/Report.pdf" to "Papers/Summary.pdf" in the vault "Test"
      And the second computer replaces the file "Papers/Report.pdf" with a new version in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has the same file "Papers/Summary.pdf" as the second computer
      And the second computer does not have the file "Papers/Report.pdf" in its vault "Copy"

  Rule: Two computers that make the same path end with one file

    @stand
    Scenario: The same note made on two computers ends as one note
      Given the sync server stops
      And the person writes the note "Daily/Monday" with the text "tea" in the vault "Test"
      And the second computer writes the note "Daily/Monday" with the text "tea" in its vault "Copy"
      When the sync server runs again
      Then the vault "Test" has the note "Daily/Monday" with the text "tea"
      And the second computer has the note "Daily/Monday" with the text "tea" in its vault "Copy"
      And the vault "Test" has no conflict copy of the note "Daily/Monday"

    @stand
    Scenario: The same path with another text keeps both texts
      Given the sync server stops
      And the person writes the note "Daily/Monday" with the text "tea" in the vault "Test"
      And the second computer writes the note "Daily/Monday" with the text "rice" in its vault "Copy"
      And the second computer is the first to reach the sync server
      When the sync server runs again
      Then the vault "Test" has the note "Daily/Monday" with the text "rice"
      And the vault "Test" has a conflict copy of the note "Daily/Monday" with the text "tea"

  Rule: Two names that differ only in letter case are not both synced

    @live
    Scenario: A file whose name differs only in case is not synced
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      When the second computer adds the file "Papers/report.pdf" to its vault "Copy"
      Then the person is told "1 file is not synced: its name matches another name"
