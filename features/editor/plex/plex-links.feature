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

    Scenario: The plex opens on the note of its tab
      Given the person opens the note titled "Quay"
      When the person runs the command "Show in plex" on the note tab titled "Harbour"
      Then the plex shows the focus note titled "Harbour"
