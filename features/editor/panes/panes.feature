Feature: Panes
  A person opens a note beside the others.

  As a person who keeps notes
  I want to open a note in a pane of its own
  So that I can read two notes side by side

  Scenario: A note is opened beside
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text                  |
      | notes/Entropy.md | Entropy is a measure. |
    And the application "editor" starts
    And the person opens the note titled "Entropy"
    When the person runs the command "Open note beside" on the note tab titled "Entropy"
    Then the note tab titled "Entropy" is in a pane of its own
    And the note tab at "notes/Entropy.md" is open
    And the note tab at "notes/Entropy.md" is active
