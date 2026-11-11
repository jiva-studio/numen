Feature: Delete note permanently
  A person deletes a note for good after typing its name again.

  Scenario: A note is deleted for good
    Given the empty vault "Test"
    And the vault has the following:
      | path                | text               |
      | notes/Tide table.md | High water at six. |
    And the application "editor" starts
    And the person opens the note titled "Tide table"
    And the tab at "notes/Tide table.md" is open
    When the person runs the command "Delete note permanently" on the file at "notes/Tide table.md" with:
      | Type the note name to confirm | Tide table |
    Then the tab at "notes/Tide table.md" is not open
    And the vault does not contain the file at "notes/Tide table.md"
