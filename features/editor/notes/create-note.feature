Feature: Create note
  A person makes a note with a command.

  Scenario: The command Create note makes a note
    When the person runs the command "Create note"
    And the person enters "Quantum"
    Then the vault "Test" has the note titled "Quantum"
    And the tab titled "Quantum" is active
