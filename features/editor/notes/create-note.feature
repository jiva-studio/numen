Feature: Create note
  A person makes a note with a command.

  As a person who keeps notes
  I want to make a note with a name
  So that I can start to write

  Scenario: A note is made with a name
    Given the empty vault "Test"
    And the application "editor" starts
    When the person runs the command "Create note" with:
      | Name | Quantum |
    Then the vault contains the note titled "Quantum"
    And the note tab titled "Quantum" is active
