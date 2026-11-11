Feature: Failed tasks
  A thing that fails in the background shows as a failed task in the task list.

  Background:
    Given the empty vault "Test"

  Scenario: A text that cannot be fetched is a failed task
    Given the application "editor" starts
    And the person imports the link "http://unreachable.numen.test/tide-tables"
    When the person downloads the text of the link "http://unreachable.numen.test/tide-tables"
    Then the task list has the failed task "tide-tables answered 404 Not Found"
    And the task list has the failed task "Fetching the text of a link"

  Scenario: A Custom channel with no release feed address is a failed task
    Given the application "editor" starts
    And the person sets the setting "Channel" of the plugin "Update Notifier" to "Custom"
    When the person restarts the application
    Then the task list has the failed task "release feed address"
