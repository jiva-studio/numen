Feature: Plex drag
  A person drags a line from a note to link it to another note.

  As a person who keeps notes
  I want to drag a line from a note
  So that I can link notes without writing

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A line dragged from a note links it

    Background:
      Given the vault has the following:
        | path             | text      |
        | notes/Harbour.md | # Harbour |
        | notes/Quay.md    | # Quay    |
      And the person opens the note titled "Harbour"
      And the person runs the command "Show in plex" on the note tab titled "Harbour"

    Scenario Outline: A line makes a link of the chosen kind
      When the person drags a line <direction> from the note titled "Harbour" to the note titled "Quay"
      Then the vault "Test" has the following links:
        | from    | to   | role       |
        | Harbour | Quay | <relation> |

      Examples:
        | direction | relation |
        | up        | parent   |
        | down      | child    |
        | left      | jump     |

    Scenario: A line makes a new note
      When the person drags a line down from the note titled "Harbour" to a new note titled "Pier"
      Then the vault "Test" has the following links:
        | from | to      | role   |
        | Pier | Harbour | parent |
      And the plex shows the focus note titled "Pier"
      And the plex shows the parent note titled "Harbour"

    Scenario: A click on the handle links a note as a child
      When the person clicks the handle of the note titled "Harbour" to link the note titled "Quay"
      Then the vault "Test" has the following links:
        | from    | to   | role  |
        | Harbour | Quay | child |

  Rule: A line dropped on a note of the plex links it

    Scenario: A line onto a reference note makes a link
      Given the vault has the following:
        | path             | text                        |
        | notes/Harbour.md | # Harbour\n\nSee [[Jetty]]. |
        | notes/Jetty.md   | # Jetty                     |
      And the person opens the note titled "Harbour"
      And the person runs the command "Show in plex" on the note tab titled "Harbour"
      When the person drops the handle of the note titled "Harbour" on the note titled "Jetty"
      Then the vault "Test" has the following links:
        | from    | to    | role |
        | Harbour | Jetty | jump |
