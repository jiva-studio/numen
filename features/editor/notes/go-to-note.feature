Feature: Go to note
  A person finds a note by its title and sees where it stands among the other notes.

  As a person who keeps notes
  I want to go to a note by its title and to see its place in the plex
  So that I find my notes fast and I see how they link

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path                 | text                                                                         |
      | notes/Entropy.md     | Entropy is a measure of uncertainty.                                         |
      | Thermodynamics.md    | ---\nlinks:\n  - to: "[[notes/Entropy]]"\n    role: child\n---\n\nHeat and work. |
    And the application "editor" starts

  Scenario: The note is opened by its title
    When the person runs the command "Go to note" with:
      | Search notes | Entropy |
    Then the note tab at "notes/Entropy.md" is active

  Scenario: The note is shown in the plex with its parent
    Given the person runs the command "Go to note" with:
      | Search notes | Entropy |
    When the person runs the command "Show in plex"
    Then the plex has the focus note titled "Entropy"
    And the plex has the parent note titled "Thermodynamics"
