Feature: Move note to trash
  A person removes a note and it leaves the places that show notes.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Scenario: The note leaves the file tree
    Given the vault has the following:
      | path                | text                |
      | notes/Tide table.md | High water at six.  |
    And the person opens the note titled "Tide table"
    And the person opens the folder at "notes"
    And the tab titled "Files" has "notes/Tide table.md"
    When the person runs the command "Move note to trash" on the tab titled "Tide table"
    Then the tab titled "Files" does not have "notes/Tide table.md"

  Scenario: The note leaves the plex
    Given the vault has the following:
      | path                 | text                                                                         |
      | notes/Harbour log.md | ---\nlinks:\n  - to: "[[notes/Tide table]]"\n    role: child\n---\n\nThe tide is low. |
      | notes/Tide table.md  | High water at six.                                                           |
    And the person opens the note titled "Harbour log"
    And the person opens the note titled "Tide table"
    And the person runs the command "Show note in plex" on the tab titled "Harbour log"
    And the plex has the note titled "Tide table" as the "child"
    When the person runs the command "Move note to trash" on the tab titled "Tide table"
    Then the plex has the note titled "Harbour log" as the "focus"
    And the plex does not have the note titled "Tide table" as the "child"
