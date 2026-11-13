Feature: Plex menu
  A person opens the menu of a note in the plex.

  As a person who keeps notes
  I want a menu on each note of the plex
  So that I can act on a note where I see it

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text      |
      | notes/Harbour.md | # Harbour |
      | notes/Quay.md    | # Quay    |
    And the vault has the following presets:
      | preset   |
      | Sanskrit |
    And the notes have the following links:
      | from    | to       | role  |
      | Harbour | Quay     | child |
      | Harbour | Sanskrit | child |
    And the application "editor" starts
    And the person opens the note titled "Harbour"
    And the person runs the command "Show in plex" on the note tab titled "Harbour"

  Rule: The menu of a note has the actions for it

    Scenario: The menu of a note has its items
      When the person opens the menu of the note titled "Quay" in the plex
      Then the menu contains the following items:
        | item             |
        | Open note        |
        | Copy note path   |
        | Create child note |
        | Rename note      |
        | Move to trash    |

    Scenario: The menu of a preset has the item for a preset
      When the person opens the menu of the note titled "Sanskrit" in the plex
      Then the menu contains the following items:
        | item        |
        | Open preset |

    Scenario: The menu of an ordinary note has no item for a preset
      When the person opens the menu of the note titled "Quay" in the plex
      Then the menu does not contain the following items:
        | item        |
        | Open preset |
