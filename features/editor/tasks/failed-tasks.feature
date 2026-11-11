Feature: Failed tasks
  A thing that fails in the background shows as a failed task in the task list.

  Scenario: A text that cannot be fetched is a failed task
    Given the empty vault "Test"
    And the application "editor" starts
    And the person imports the link "http://unreachable.numen.test/tide-tables"
    When the person downloads the text of the link "http://unreachable.numen.test/tide-tables"
    Then the task list has the failed task "tide-tables answered 404 Not Found"
    And the task list has the failed task "Fetching the text of a link"
