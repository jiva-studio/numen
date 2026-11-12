Feature: Cancel a task
  A person stops work that runs in the background.

  As a person who uses the editor
  I want to see the tasks that run in the background and cancel one
  So that I stop work I do not need

  Background:
    Given the empty vault "Test"
    And the environment is set to the following:
      | embedding model | slow |
    And the vault has the following:
      | path                | text                      |
      | notes/Tide clock.md | The clock is by the quay. |
    And the application "editor" starts
    And the person opens the task list

  Scenario: A running task is listed
    Then the task list has the task "notes/Tide clock.md"

  Scenario: A running task is cancelled
    Given the task list has the task "notes/Tide clock.md"
    When the person cancels the task "notes/Tide clock.md"
    Then the task list does not have the task "notes/Tide clock.md"

  Scenario: A finished task leaves the list
    Given the task list has the task "notes/Tide clock.md"
    When the environment is set to the following:
      | embedding model | normal |
    Then the task list does not have the task "notes/Tide clock.md"
