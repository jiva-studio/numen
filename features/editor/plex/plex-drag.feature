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

    Scenario: A line makes a new note with a link
      When the person drags a line down from the note titled "Harbour" to a new note titled "Pier"
      Then the vault "Test" has the following links:
        | from | to      | role   |
        | Pier | Harbour | parent |

    Scenario: A line makes a new note that the plex shows
      When the person drags a line down from the note titled "Harbour" to a new note titled "Pier"
      Then the plex shows the focus note titled "Pier"

    Scenario: A click on the handle links a note as a child
      When the person clicks the handle of the note titled "Harbour" to link the note titled "Quay"
      Then the vault "Test" has the following links:
        | from    | to   | role  |
        | Harbour | Quay | child |

  Rule: A line dropped on a note of the plex links it

    Scenario Outline: A line onto a note makes a link by the place of the note
      Given the vault has the following:
        | path             | text      |
        | notes/Coast.md   | # Coast   |
        | notes/Harbour.md | # Harbour |
        | notes/Quay.md    | # Quay    |
      And the notes have the following links:
        | from    | to      | role  |
        | Coast   | Harbour | child |
        | Harbour | Quay    | child |
      And the person opens the note titled "Harbour"
      And the person runs the command "Show in plex" on the note tab titled "Harbour"
      When the person drops the handle of the note titled "<from>" on the note titled "<to>"
      Then the vault "Test" has the following links:
        | from    | to      | role       |
        | Coast   | Harbour | child      |
        | Harbour | Quay    | child      |
        | <from>  | <to>    | <relation> |

      Examples:
        | from  | to    | relation |
        | Quay  | Coast | parent   |
        | Coast | Quay  | child    |
