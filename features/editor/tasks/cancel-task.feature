Feature: Cancel a task
  A person stops work that runs in the background.

  Background:
    Given the empty vault "Test"
    And the embedding model is paused
    And the vault has the following:
      | path                | text                 |
      | notes/Tide clock.md | The clock is by the quay. |
    And the application "editor" starts
    And the person opens the task list

  Scenario: A running task is listed
    Then the task list has the task "notes/Tide clock.md"

  Scenario: A running task is cancelled
    Given the task list has the task "notes/Tide clock.md"
    When the person cancels the task "notes/Tide clock.md"
    Then the task list does not have the task "notes/Tide clock.md"
