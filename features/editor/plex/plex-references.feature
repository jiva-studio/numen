Feature: Plex references
  A person sees the notes that a note links to in its text, and the notes that link to it.

  As a person who keeps notes
  I want to see the notes named in the text of a note and the notes that name it
  So that I can follow a mention to its note

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path           | text           |
      | notes/Quay.md  | See [[Jetty]]. |
      | notes/Jetty.md | # Jetty        |
      | notes/Dock.md  | See [[Quay]].  |
    And the application "editor" starts
    And the person opens the note titled "Quay"
    And the person runs the command "Show in plex" on the note tab titled "Quay"

  Rule: The setting "References" chooses the reference notes

    Scenario: Both kinds of reference are shown
      Then the plex shows the following notes:
        | relation  | title |
        | reference | Jetty |
        | reference | Dock  |

    Scenario: References are hidden
      Given the plex shows the following notes:
        | relation  | title |
        | reference | Jetty |
        | reference | Dock  |
      When the person sets the setting "References" of the plugin "Plex Graph" to "Off"
      Then the plex does not show the following notes:
        | relation  | title |
        | reference | Jetty |
        | reference | Dock  |

    Scenario Outline: Only the references of one direction are shown
      Given the plex shows the reference note titled "<hidden>"
      When the person sets the setting "References" of the plugin "Plex Graph" to "<setting>"
      Then the plex shows the reference note titled "<shown>"
      And the plex does not show the reference note titled "<hidden>"
      And the plex draws the line from "Quay" to "<shown>" with an arrow to "<arrow>"

      Examples:
        | setting  | shown | hidden | arrow |
        | Outgoing | Jetty | Dock   | Jetty |
        | Incoming | Dock  | Jetty  | Quay  |

    Scenario: Both directions are shown
      Given the person sets the setting "References" of the plugin "Plex Graph" to "Outgoing"
      And the plex shows the reference note titled "Jetty"
      And the plex does not show the reference note titled "Dock"
      When the person sets the setting "References" of the plugin "Plex Graph" to "Both"
      Then the plex shows the following notes:
        | relation  | title |
        | reference | Jetty |
        | reference | Dock  |
