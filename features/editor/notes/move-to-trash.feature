Feature: Move to trash
  A person removes a note or a file and it leaves the places that show it.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Scenario: The note leaves the file tree
    Given the vault has the following:
      | path                | text               |
      | notes/Tide table.md | High water at six. |
    And the person opens the note titled "Tide table"
    And the person opens the folder at "notes"
    And the tab titled "Files" has "notes/Tide table.md"
    When the person runs the command "Move to trash" on the file at "notes/Tide table.md"
    Then the tab titled "Files" does not have "notes/Tide table.md"
    And the tab at "notes/Tide table.md" is not open
    And the vault does not contain the file at "notes/Tide table.md"

  Scenario: The note leaves the plex
    Given the vault has the following:
      | path                 | text                                                                         |
      | notes/Harbour log.md | ---\nlinks:\n  - to: "[[notes/Tide table]]"\n    role: child\n---\n\nThe tide is low. |
      | notes/Tide table.md  | High water at six.                                                           |
    And the person opens the note titled "Harbour log"
    And the person opens the note titled "Tide table"
    And the person runs the command "Show in plex" on the tab titled "Harbour log"
    And the plex has the child note titled "Tide table"
    When the person runs the command "Move to trash" on the file at "notes/Tide table.md"
    Then the plex does not have the child note titled "Tide table"

  Scenario: The note leaves the search
    Given the vault has the following:
      | path                | text               |
      | notes/Tide table.md | High water at six. |
    And the person searches the vault for "High water"
    And the search has the result "High water at six" in the group "Text"
    And the person opens the note titled "Tide table"
    When the person runs the command "Move to trash" on the file at "notes/Tide table.md"
    And the person searches the vault for "High water"
    Then the search does not have the result "High water at six" in the group "Text"

  Scenario Outline: A <kind> leaves the file tree and the search
    Given the vault has the following:
      | path   | sample   |
      | <path> | <sample> |
    And the person opens the folder at "assets"
    And the tab titled "Files" has "<path>"
    And the person searches the vault for "<words>"
    And the search has the result "<path>" in the group "Text"
    When the person runs the command "Move to trash" on the file at "<path>"
    And the person searches the vault for "<words>"
    Then the tab titled "Files" does not have "<path>"
    And the search does not have the result "<path>" in the group "Text"

    Examples:
      | kind | path               | sample      | words                  |
      | PDF  | assets/shore.pdf   | shore.pdf   | heron wades            |
      | book | assets/sample.epub | sample.epub | reflowable epub reader |

  Scenario: A recording leaves the file tree and the search
    Given the vault has the following:
      | path               | sample      |
      | assets/lecture.wav | lecture.wav |
    And the person transcribes the recording at "assets/lecture.wav"
    And the person proofreads the transcript of the file at "assets/lecture.wav"
    And the person opens the folder at "assets"
    And the tab titled "Files" has "assets/lecture.wav"
    And the person searches the vault for "proofread"
    And the search has the result "assets/lecture.wav" in the group "Text"
    When the person runs the command "Move to trash" on the file at "assets/lecture.wav"
    And the person searches the vault for "proofread"
    Then the tab titled "Files" does not have "assets/lecture.wav"
    And the search does not have the result "assets/lecture.wav" in the group "Text"

  Scenario: A link to the removed note stays in the text
    Given the vault has the following:
      | path                 | text                          |
      | notes/Harbour log.md | See [[Tide table]] for tides. |
      | notes/Tide table.md  | High water at six.            |
    And the person opens the note titled "Harbour log"
    And the person opens the note titled "Tide table"
    When the person runs the command "Move to trash" on the file at "notes/Tide table.md"
    Then the note titled "Harbour log" on disk has "See [[Tide table]] for tides."
    And the note titled "Harbour log" has "[[Tide table]]"
