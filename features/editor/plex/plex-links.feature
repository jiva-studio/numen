Feature: Plex links
  A person sees the notes joined to a note by their links.

  As a person who keeps notes
  I want to see the notes joined to a note
  So that I can follow from one note to the next

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path                | text         |
      | notes/Harbour.md    | # Harbour    |
      | notes/Coast.md      | # Coast      |
      | notes/Cape.md       | # Cape       |
      | notes/Quay.md       | # Quay       |
      | notes/Jetty.md      | # Jetty      |
      | notes/Lighthouse.md | # Lighthouse |
    And the notes have the following links:
      | from    | to         | role  |
      | Coast   | Harbour    | child |
      | Coast   | Cape       | child |
      | Harbour | Quay       | child |
      | Harbour | Jetty      | child |
      | Harbour | Lighthouse | jump  |
    And the application "editor" starts
    And the person opens the note titled "Harbour"
    And the person runs the command "Show in plex" on the note tab titled "Harbour"

  Rule: The plex shows notes by their links

    Scenario: The notes joined to a note are shown
      Then the plex shows the following notes:
        | relation | title      |
        | focus    | Harbour    |
        | parent   | Coast      |
        | child    | Quay       |
        | child    | Jetty      |
        | jump     | Lighthouse |
        | sibling  | Cape       |

    Scenario: A click moves the focus
      When the person clicks the note titled "Quay" in the plex
      Then the plex shows the following notes:
        | relation | title   |
        | focus    | Quay    |
        | parent   | Harbour |
        | sibling  | Jetty   |
      And the plex does not show the following notes:
        | relation | title      |
        | parent   | Coast      |
        | sibling  | Cape       |
        | jump     | Lighthouse |

    Scenario: The plex opens on the note of its tab
      Given the person opens the note titled "Quay"
      And the person clicks the note titled "Quay" in the plex
      And the plex shows the focus note titled "Quay"
      When the person runs the command "Show in plex" on the note tab titled "Harbour"
      Then the plex shows the focus note titled "Harbour"

  Rule: Each relation has its side of the focus note

    Scenario: The notes are placed by their relation
      Then the plex places the following notes on the side of the focus note:
        | title      | side       |
        | Coast      | above      |
        | Quay       | below      |
        | Jetty      | below      |
        | Lighthouse | below left |
        | Cape       | right      |
