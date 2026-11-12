Feature: Empty trash
  A person deletes the notes in the trash for good.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path                | text               |
      | notes/Tide table.md | High water at six. |
    And the application "editor" starts
    And the person opens the note titled "Tide table"
    And the person runs the command "Move to trash" on the file at "notes/Tide table.md"
    And the vault contains the file at ".trash/notes/Tide table.md"

  Scenario: The trash is emptied
    When the person runs the command "Empty trash" with:
      | Confirm | Empty the trash of |
    Then the vault does not contain the file at ".trash/notes/Tide table.md"

  Scenario: The trash is kept
    When the person runs the command "Empty trash" with:
      | Confirm | Keep trash |
    Then the vault contains the file at ".trash/notes/Tide table.md"
