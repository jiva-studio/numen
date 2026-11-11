Feature: Unreadable frontmatter
  A person opens a note whose frontmatter cannot be read.

  Scenario: A note with unreadable frontmatter says so
    Given the empty vault "Test"
    And the vault has the following:
      | path            | text                                                                       |
      | notes/Broken.md | ---\ntitle: [unclosed bracket\nkey: value:\n---\n# Broken\nSome text.      |
    And the application "editor" starts
    When the person opens the file at "notes/Broken.md"
    Then the note tab at "notes/Broken.md" has "frontmatter unreadable"
