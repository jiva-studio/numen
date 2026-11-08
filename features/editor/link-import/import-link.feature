Feature: Import link
  A person imports a web page or a video by its address.

  Background:
    Given the application "editor" starts

  Rule: A link to a web page has the text of the page

    Scenario: The text of a web page is downloaded into its link note
      Given the web address "http://tidal-mills.numen.test/history/rance" has the page text "A tidal mill stored seawater behind a sluice gate."
      When the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance"
      And the person runs the command "Download captions"
      Then the tab titled "rance" has "A tidal mill stored seawater behind a sluice gate."

    Scenario: The text of a web page is found by search
      Given the web address "http://tidal-mills.numen.test/history/rance" has the page text "A tidal mill stored seawater behind a sluice gate."
      And the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance"
      And the person runs the command "Download captions"
      When the person searches the vault for "sluice gate"
      Then the search has the result "tidal-mills.numen.test" in the group "Text"

    Scenario: The deleted text of a link is not there after the application starts
      Given the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance?part=2"
      And the person runs the command "Download captions"
      When the person runs the command "Delete text"
      And the person enters "Article (current)"
      And the application "editor" starts
      Then the tab titled "rance-part=2" does not have "A tidal mill stored seawater behind a sluice gate."

  Rule: A link has one note

    Scenario: A link with another address has a note of its own
      Given the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance?part=2"
      When the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance?part=3"
      Then the number of the notes of the type "link" in the vault "Test" is "2"

    Scenario: A link imported again opens the note it already has
      Given the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance?part=3"
      And the person closes the tab titled "rance-part=3"
      When the person runs the command "Import link"
      And the person enters "http://tidal-mills.numen.test/history/rance?part=3"
      Then the tab titled "rance-part=3" is open
      And the number of the notes of the type "link" in the vault "Test" is "1"

  Rule: A link to a video has a recording that plays

    Scenario: A video is downloaded and plays
      Given the person runs the command "Import link"
      And the person enters "https://www.youtube.com/watch?v=tidalmill01"
      And the person runs the command "Download captions"
      And the person runs the command "Download media"
      And the person enters "Video"
      When the person plays the recording in the tab titled "tidalmill01"
      Then the recording in the tab titled "tidalmill01" is playing
      And the tab titled "tidalmill01" has "The mill pond fills behind the gate."

    Scenario: A video note is opened from the file tree
      Given the person runs the command "Import link"
      And the person enters "https://www.youtube.com/watch?v=tidalmill02"
      And the person runs the command "Download captions"
      And the person runs the command "Download media"
      And the person enters "Video"
      When the application "editor" starts
      And the person opens the file at "https---www.youtube.com-watch-v=tidalmill02.md"
      Then the tab at "https---www.youtube.com-watch-v=tidalmill02.md" has "The mill pond fills behind the gate."
