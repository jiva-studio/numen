Feature: Import link
  A person imports a web page or a video by its address.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A web page has its text

    Background:
      Given the web address "http://numen.dev/one" has the page text "Tidal mills store water."
      And the person imports the link "http://numen.dev/one"

    Scenario: The text is downloaded
      When the person downloads the text of the link "http://numen.dev/one"
      Then the tab for the web address "http://numen.dev/one" has "Tidal mills store water."

    Scenario: The text is found by search
      Given the person downloads the text of the link "http://numen.dev/one"
      When the person searches the vault for "store water"
      Then the search has the result "numen.dev" in the group "Text"

    Scenario: The deleted text stays deleted
      Given the person downloads the text of the link "http://numen.dev/one"
      And the tab for the web address "http://numen.dev/one" has "Tidal mills store water."
      When the person deletes the text of the link "http://numen.dev/one"
      And the application "editor" starts
      Then the tab for the web address "http://numen.dev/one" does not have "Tidal mills store water."

  Rule: A link has one note

    Background:
      Given the person imports the link "http://numen.dev/three"

    Scenario: Another address has its own note
      When the person imports the link "http://numen.dev/two"
      Then the number of the notes of the type "link" in the vault "Test" is 2

    Scenario: The same address opens its note
      Given the person closes the Media tab for the web address "http://numen.dev/three"
      When the person imports the link "http://numen.dev/three"
      Then the Media tab for the web address "http://numen.dev/three" is open
      And the number of the notes of the type "link" in the vault "Test" is 1
