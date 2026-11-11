Feature: Move note to trash
  A person removes a note or a file and it leaves the places that show it.

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

  Scenario: The note leaves the search
    Given the vault has the following:
      | path                | text               |
      | notes/Tide table.md | High water at six. |
    And the person searches the vault for "High water"
    And the search has the result "High water at six" in the group "Text"
    And the person opens the note titled "Tide table"
    When the person runs the command "Move note to trash" on the tab titled "Tide table"
    And the person searches the vault for "High water"
    Then the search does not have the result "High water at six" in the group "Text"

  Scenario: A PDF leaves the file tree and the search
    Given the vault has the following:
      | path             | sample    |
      | assets/shore.pdf | shore.pdf |
    And the person opens the folder at "assets"
    And the tab titled "Files" has "assets/shore.pdf"
    And the person searches the vault for "heron wades"
    And the search has the result "assets/shore.pdf" in the group "Text"
    When the person runs the command "Move note to trash" on the file at "assets/shore.pdf"
    And the person searches the vault for "heron wades"
    Then the tab titled "Files" does not have "assets/shore.pdf"
    And the search does not have the result "assets/shore.pdf" in the group "Text"

  Scenario: A book leaves the file tree and the search
    Given the vault has the following:
      | path               | sample      |
      | assets/sample.epub | sample.epub |
    And the person opens the folder at "assets"
    And the tab titled "Files" has "assets/sample.epub"
    And the person searches the vault for "reflowable epub reader"
    And the search has the result "assets/sample.epub" in the group "Text"
    When the person runs the command "Move note to trash" on the file at "assets/sample.epub"
    And the person searches the vault for "reflowable epub reader"
    Then the tab titled "Files" does not have "assets/sample.epub"
    And the search does not have the result "assets/sample.epub" in the group "Text"

  Scenario: A link to the removed note stays in the text
    Given the vault has the following:
      | path                 | text                         |
      | notes/Harbour log.md | See [[Tide table]] for tides. |
      | notes/Tide table.md  | High water at six.           |
    And the person opens the note titled "Tide table"
    When the person runs the command "Move note to trash" on the tab titled "Tide table"
    Then the note titled "Harbour log" on disk has "See [[Tide table]] for tides."

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
    When the person runs the command "Move note to trash" on the file at "assets/lecture.wav"
    And the person searches the vault for "proofread"
    Then the tab titled "Files" does not have "assets/lecture.wav"
    And the search does not have the result "assets/lecture.wav" in the group "Text"
