Feature: Cancel a task
  A person stops work that runs in the background.

  Scenario: A running task is cancelled
    Given the empty vault "Test"
    And the vault has the note at "notes/Tide clock.md" that the embedding model holds
    And the application "editor" starts
    When the person opens the task list
    Then the task list has the task "notes/Tide clock.md"
    When the person cancels the task "notes/Tide clock.md"
    Then the task list does not have the task "notes/Tide clock.md"
