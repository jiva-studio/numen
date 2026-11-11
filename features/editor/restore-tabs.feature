Feature: Restore tabs
  A person leaves tabs open. When the window starts again, each tab is restored at the place where the person left it.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Scenario: A note keeps its text
    Given the vault has the following:
      | path                   | text |
      | notes/Shopping list.md | tea  |
    And the person opens the note titled "Shopping list"
    And the person writes "bread" in the note titled "Shopping list"
    And the note titled "Shopping list" on disk has "bread"
    When the application "editor" starts
    Then the note titled "Shopping list" has "bread"

  Scenario: A PDF keeps its page
    Given the vault has the following:
      | path             | sample    |
      | assets/shore.pdf | shore.pdf |
    And the PDF tab at "assets/shore.pdf" is open at page "2"
    When the application "editor" starts
    Then the PDF tab at "assets/shore.pdf" is at page "2"

  Scenario: A book keeps its page
    Given the vault has the following:
      | path               | sample      |
      | assets/ledger.epub | ledger.epub |
    And the book tab at "assets/ledger.epub" is open at page "2"
    When the application "editor" starts
    Then the book tab at "assets/ledger.epub" is at page "2"

  Scenario: A recording keeps its time
    Given the vault has the following:
      | path            | sample   |
      | assets/talk.mp3 | talk.mp3 |
    And the recording tab at "assets/talk.mp3" is open at time "2:00"
    When the application "editor" starts
    Then the recording tab at "assets/talk.mp3" is at time "2:00"

  Scenario: The open tabs and the active tab are restored
    Given the vault has the following:
      | path             | text                  | sample    |
      | notes/Entropy.md | Entropy is a measure. |           |
      | assets/shore.pdf |                       | shore.pdf |
    And the person opens the file at "assets/shore.pdf"
    And the person opens the file at "notes/Entropy.md"
    When the application "editor" starts
    Then the tab at "assets/shore.pdf" is open
    And the tab at "notes/Entropy.md" is active
