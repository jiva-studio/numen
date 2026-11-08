Feature: Import link
  A person imports a web page or a video by its address.

  Background:
    Given the application "editor" starts

  Rule: A link to a web page has the text of the page

    Scenario: The text of a web page is downloaded into its link note
      Given the web address "http://numen.dev/one" has the page text "Tidal mills store water."
      When the person runs the command "Import link"
      And the person enters "http://numen.dev/one"
      And the person runs the command "Download captions"
      Then the tab for the web address "http://numen.dev/one" has "Tidal mills store water."

    Scenario: The text of a web page is found by search
      Given the web address "http://numen.dev/one" has the page text "Tidal mills store water."
      And the person runs the command "Import link"
      And the person enters "http://numen.dev/one"
      And the person runs the command "Download captions"
      When the person searches the vault for "store water"
      Then the search has the result "numen.dev" in the group "Text"

    Scenario: The deleted text of a link is not there after the application starts
      Given the web address "http://numen.dev/two" has the page text "Tidal mills store water."
      And the person runs the command "Import link"
      And the person enters "http://numen.dev/two"
      And the person runs the command "Download captions"
      When the person runs the command "Delete text"
      And the person enters "Article (current)"
      And the application "editor" starts
      Then the tab for the web address "http://numen.dev/two" does not have "Tidal mills store water."

  Rule: A link has one note

    Scenario: A link with another address has a note of its own
      Given the person runs the command "Import link"
      And the person enters "http://numen.dev/two"
      When the person runs the command "Import link"
      And the person enters "http://numen.dev/three"
      Then the number of the notes of the type "link" in the vault "Test" is "2"

    Scenario: A link imported again opens the note it already has
      Given the person runs the command "Import link"
      And the person enters "http://numen.dev/three"
      And the person closes the tab for the web address "http://numen.dev/three"
      When the person runs the command "Import link"
      And the person enters "http://numen.dev/three"
      Then the tab for the web address "http://numen.dev/three" is open
      And the number of the notes of the type "link" in the vault "Test" is "1"

  Rule: A link to a video has a recording that plays

    Scenario: A video is downloaded and plays
      Given the person runs the command "Import link"
      And the person enters "https://youtu.be/mill01"
      And the person runs the command "Download captions"
      And the person runs the command "Download media"
      And the person enters "Video"
      When the person plays the recording in the tab for the web address "https://youtu.be/mill01"
      Then the recording in the tab for the web address "https://youtu.be/mill01" is playing
      And the tab for the web address "https://youtu.be/mill01" has "The mill pond fills behind the gate."

    Scenario: A video note is opened from the file tree
      Given the person runs the command "Import link"
      And the person enters "https://youtu.be/mill02"
      And the person runs the command "Download captions"
      And the person runs the command "Download media"
      And the person enters "Video"
      When the application "editor" starts
      And the person opens the file at "https---youtu.be-mill02.md"
      Then the tab at "https---youtu.be-mill02.md" has "The mill pond fills behind the gate."
