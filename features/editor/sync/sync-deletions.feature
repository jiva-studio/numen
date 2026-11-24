Feature: Sync deleted files between computers

  Numen Sync removes a file on every computer when the person removes it on one.

  As a person who removes files I do not need
  I want a removed file to be gone from the other computer
  So that both computers hold the same files

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And a second computer restores the identity from the recovery words
    And the second computer connects its vault "Copy" to the synced vault "Test"

  Rule: A removed file is removed on the other computer

    Scenario: A removed PDF is removed on the second computer
      Given the person writes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      When the person removes the file "Papers/Report.pdf" in the vault "Test"
      Then the second computer does not have the file "Papers/Report.pdf" in its vault "Copy"

    Scenario: A removed note is removed on the second computer
      Given the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      And the second computer has the note "Shopping list" with the text "milk" in its vault "Copy"
      When the person removes the file "Shopping list.md" in the vault "Test"
      Then the second computer does not have the file "Shopping list.md" in its vault "Copy"

    Scenario: A removed folder is removed on the second computer
      Given the person writes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      When the person removes the folder "Papers" in the vault "Test"
      Then the second computer does not have the file "Papers/Report.pdf" in its vault "Copy"

    Scenario: A removed file can be written again at its path
      Given the person writes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the person removes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer does not have the file "Papers/Report.pdf" in its vault "Copy"
      When the person writes the file "Papers/Report.pdf" in the vault "Test"
      Then the second computer has the same file "Papers/Report.pdf" in its vault "Copy"

  Rule: A change wins over a removal

    Scenario: A file changed on one computer and removed on the other stays
      Given the person writes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the sync server stops
      And the person removes the file "Papers/Report.pdf" in the vault "Test"
      And the second computer changes the file "Papers/Report.pdf" in its vault "Copy"
      When the sync server runs again
      Then the second computer has the same file "Papers/Report.pdf" in its vault "Copy"
      And the vault "Test" has the same file "Papers/Report.pdf" as the second computer
