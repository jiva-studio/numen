Feature: Create note
  A person makes a note with a command.

  Scenario: A note is made with a name
    Given the empty vault "Test"
    And the application "editor" starts
    When the person runs the command "Create note"
    And the person names it "Quantum"
    Then the vault contains the note titled "Quantum"
    And the note tab titled "Quantum" is active
