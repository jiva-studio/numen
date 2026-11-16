Feature: Tags
  A person sets the tags of notes and sees them in the file tree and in the plex.

  As a person who keeps notes
  I want to set and see the tags of a note where I see the note
  So that I sort my notes without opening them

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path          | text      |
      | notes/Quay.md | The quay. |
      | notes/Pier.md | The pier. |
      | notes/Dock.md | The dock. |
    And the notes have the following tags:
      | note | tag  |
      | Quay | Port |
      | Quay | Sea  |
      | Dock | Port |
      | Dock | Sea  |
      | Dock | Quay |
    And the application "editor" starts

  Rule: A person sets the tags of a note

    @smoke
    Scenario: Several tags are set on a file of the file tree
      When the person runs the command "Set tags" on the file at "notes/Pier.md" with:
        | Tag | Port |
        | Tag | Sea  |
      Then the notes have the following tags:
        | note | tag  |
        | Pier | Port |
        | Pier | Sea  |

    Scenario: One of several tags is cleared and the others stay
      When the person runs the command "Set tags" on the file at "notes/Dock.md" with:
        | Tag | Sea |
      Then the notes have the following tags:
        | note | tag  |
        | Dock | Port |
        | Dock | Quay |

    Scenario: A tag is set and another is cleared together
      When the person runs the command "Set tags" on the file at "notes/Quay.md" with:
        | Tag | Port    |
        | Tag | Harbour |
      Then the notes have the following tags:
        | note | tag     |
        | Quay | Sea     |
        | Quay | Harbour |

    Scenario: A new tag is made from the typed text
      When the person runs the command "Set tags" on the file at "notes/Pier.md"
      And the person types "Harbour" in the palette
      Then the palette has the following rows:
        | list | row            | disabled |
        | Tags | Create Harbour | False    |

    Scenario: Several tags are set on a note of the plex
      Given the person opens the note titled "Pier"
      And the person runs the command "Show in plex" on the note tab titled "Pier"
      When the person chooses the item "Set tags" in the menu of the note titled "Pier" in the plex
      And the person answers what the palette asks with:
        | Tag | Port |
        | Tag | Sea  |
      Then the notes have the following tags:
        | note | tag  |
        | Pier | Port |
        | Pier | Sea  |
