Feature: Tags
  A person sets the tags of notes and sees them in the file tree and in the plex.

  As a person who keeps notes
  I want to set and see the tags of a note where I see the note
  So that I sort my notes without opening them

  Rule: The file tree and the plex show the tags of a note

    Scenario: The file tree shows the tags of its files
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
      Then the file tree shows the following files:
        | file          | tag  |
        | notes/Quay.md | Port |
        | notes/Quay.md | Sea  |
        | notes/Dock.md | Port |
        | notes/Dock.md | Sea  |
        | notes/Dock.md | Quay |
        | notes/Pier.md |      |
