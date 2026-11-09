Feature: Restore tabs
  A person leaves tabs open. When the window starts again, each tab is restored at the place where the person left it.

  Scenario: A note tab is restored with the text written in it
    Given the vault has the following:
      | path                   | text |
      | notes/Shopping list.md | tea  |
    And the person opens the note titled "Shopping list"
    And the person writes "bread" in the note titled "Shopping list"
    And the note titled "Shopping list" on disk has "bread"
    When the application "editor" starts
    Then the note titled "Shopping list" has "bread"

  Scenario: A PDF tab is restored on the same page
    Given the tab at "assets/shore.pdf" is open at page "2"
    When the application "editor" starts
    Then the tab at "assets/shore.pdf" is at page "2"

  Scenario: A book tab is restored on the same page
    Given the tab at "assets/ledger.epub" is open at page "2"
    When the application "editor" starts
    Then the tab at "assets/ledger.epub" is at page "2"

  Scenario: A recording tab is restored at the same time
    Given the tab at "assets/talk.mp3" is open at time "2:00"
    When the application "editor" starts
    Then the tab at "assets/talk.mp3" is at time "2:00"
