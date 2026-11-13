Feature: Plex drag
  A person drags a line from a note to link it to another note.

  As a person who keeps notes
  I want to drag a line from a note
  So that I can link notes without writing

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text      |
      | notes/Harbour.md | # Harbour |
      | notes/Quay.md    | # Quay    |
    And the application "editor" starts
    And the person opens the note titled "Harbour"
    And the person runs the command "Show in plex" on the note tab titled "Harbour"

  Rule: A line dragged from a note links it

    Scenario Outline: A line makes a link of the chosen kind
      When the person links the note titled "Harbour" to the note titled "Quay" as a <relation> in the plex
      Then the vault "Test" has the following links:
        | from    | to   | role       |
        | Harbour | Quay | <relation> |

      Examples:
        | relation |
        | parent   |
        | child    |
        | jump     |

    Scenario: A line makes a new note
      When the person links the note titled "Harbour" to a new note titled "Pier" as a child in the plex
      Then the vault "Test" contains the note titled "Pier"
      And the vault "Test" has the following links:
        | from    | to   | role  |
        | Harbour | Pier | child |
