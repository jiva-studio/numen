Feature: Sync files between computers

  Numen Sync sends every file of a connected vault to the identity and brings it to every computer.

  As a person who keeps PDF files and images in a vault
  I want a file I add on one computer to be on the other
  So that my documents are with me on every computer

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: A file of any kind appears on the other computer

    Scenario: A PDF appears on the second computer
      When the person adds the file "Papers/Report.pdf" to the vault "Test"
      Then the second computer has the same file "Papers/Report.pdf" in its vault "Copy"

    Scenario: An image appears on the second computer
      When the person adds the file "Pictures/Tree.png" to the vault "Test"
      Then the second computer has the same file "Pictures/Tree.png" in its vault "Copy"

    Scenario: A changed file changes on the second computer
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      When the person replaces the file "Papers/Report.pdf" with a new version in the vault "Test"
      Then the second computer has the same file "Papers/Report.pdf" in its vault "Copy"

    Scenario: A file written on the second computer appears on the first
      When the second computer adds the file "Papers/Plan.pdf" to its vault "Copy"
      Then the vault "Test" has the same file "Papers/Plan.pdf" as the second computer

  Rule: A moved file keeps its content

    Scenario: A renamed file is renamed on the second computer
      Given the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      When the person moves the file "Papers/Report.pdf" to "Papers/Old report.pdf" in the vault "Test"
      Then the second computer has the same file "Papers/Old report.pdf" in its vault "Copy"
      And the second computer does not have the file "Papers/Report.pdf" in its vault "Copy"

  Rule: The server holds no readable file

    Scenario: The sync server cannot read a file
      When the person adds the file "Papers/Report.pdf" to the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      Then the sync server holds no stored text "Report.pdf"
      And the sync server holds no stored text "Papers"
