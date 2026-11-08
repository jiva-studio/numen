Feature: Restore tabs
  A person leaves tabs open. When the window starts again, each tab is restored at the place where the person left it.

  Scenario: A note tab is restored with its text
    Given the person has the note "Shopping list" with "tea"
    When the window starts again
    Then the note "Shopping list" has "tea"

  Scenario: A PDF tab is restored on the same page
    Given the tab "assets/shore.pdf" is open at page "2"
    When the window starts again
    Then the tab "assets/shore.pdf" is at page "2"

  Scenario: A book tab is restored on the same page
    Given the tab "assets/sample.epub" is open at page "2"
    When the window starts again
    Then the tab "assets/sample.epub" is at page "2"

  Scenario: A recording tab is restored at the same time
    Given the tab "assets/talk.mp3" is open at time "2:00"
    When the window starts again
    Then the tab "assets/talk.mp3" is at time "2:00"
