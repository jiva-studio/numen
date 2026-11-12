Feature: Unreadable frontmatter
  A person opens a note whose frontmatter cannot be read.

  As a person who keeps notes
  I want to be told when the frontmatter of a note cannot be read
  So that I know the note has a fault and my text is not changed

  Scenario: A note with unreadable frontmatter says so
    Given the empty vault "Test"
    And the vault has the following:
      | path            | text                                                                       |
      | notes/Broken.md | ---\ntitle: [unclosed bracket\nkey: value:\n---\n# Broken\nSome text.      |
    And the application "editor" starts
    When the person opens the file at "notes/Broken.md"
    Then the note tab at "notes/Broken.md" has "frontmatter unreadable"
