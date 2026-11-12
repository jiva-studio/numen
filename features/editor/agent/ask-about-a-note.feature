Feature: Ask about a note
  A person starts a question about a note from the note.

  Scenario: The note path is in the composer
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text                                 |
      | notes/Entropy.md | Entropy is a measure of uncertainty. |
    And the application "editor" starts
    And the person opens the note titled "Entropy"
    When the person runs the command "Ask agent about note" on the note tab titled "Entropy"
    Then the Agent tab titled "Agent" is active
    And the composer of the agent has "notes/Entropy.md — "
