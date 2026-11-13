Feature: Plex references
  A person sees the notes that a note links to in its text.

  As a person who keeps notes
  I want to see the notes named in the text of a note
  So that I can follow a mention to its note

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path           | text           |
      | notes/Quay.md  | See [[Jetty]]. |
      | notes/Jetty.md | # Jetty        |
    And the application "editor" starts
    And the person opens the note titled "Quay"
    And the person runs the command "Show in plex" on the note tab titled "Quay"

  Scenario: A link in the text is shown
    Then the plex shows the following notes:
      | relation  | title |
      | reference | Jetty |

  Scenario: References are hidden
    When the person runs the command "Plex Graph: References" with:
      | References | Off |
    Then the plex does not show the reference note titled "Jetty"
