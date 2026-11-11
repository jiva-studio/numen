Feature: Import link
  A person imports a web page or a video by its address.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A web page has its text

    Background:
      Given the web address "http://numen.dev/tides" has the page text "Tidal mills store water."
      And the person imports the link "http://numen.dev/tides"
      And the person downloads the text of the link "http://numen.dev/tides"

    Scenario: The text is downloaded
      Then the Media tab for the web address "http://numen.dev/tides" has "Tidal mills store water."

    Scenario: The text is found by search
      When the person searches the vault for "store water"
      Then the search has the result "numen.dev" in the group "Text"

    Scenario: The deleted text stays deleted
      Given the Media tab for the web address "http://numen.dev/tides" has "Tidal mills store water."
      When the person deletes the text of the link "http://numen.dev/tides"
      And the application "editor" restarts
      Then the Media tab for the web address "http://numen.dev/tides" does not have "Tidal mills store water."

  Rule: A link has one note

    Background:
      Given the person imports the link "http://numen.dev/tides"

    Scenario: Another address has its own note
      When the person imports the link "http://numen.dev/weather"
      Then the number of the notes of the type "link" in the vault "Test" is 2

    Scenario: The same address opens its note
      Given the person closes the Media tab for the web address "http://numen.dev/tides"
      When the person imports the link "http://numen.dev/tides"
      Then the Media tab for the web address "http://numen.dev/tides" is open
      And the number of the notes of the type "link" in the vault "Test" is 1

  Rule: An import makes a note or none

    Scenario: The first import opens the tab of the note
      When the person imports the link "http://numen.dev/tides"
      Then the Media tab for the web address "http://numen.dev/tides" is open
      And the number of the notes of the type "link" in the vault "Test" is 1

    Scenario: A bad address makes no note
      Given the person imports the link "http://numen.dev/tides"
      When the person tries to import the link "tides"
      Then the window shows the message "not a web address"
      And the number of the notes of the type "link" in the vault "Test" is 1
