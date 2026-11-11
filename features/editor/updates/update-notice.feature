Feature: Update notice
  A person is told when a newer release of Numen is published.

  Background:
    Given the empty vault "Test"

  Rule: The latest release decides the notice

    Scenario: A newer release is announced at the start
      Given the environment is set to the following:
        | latest release | 2.0.0 |
      When the application "editor" starts
      Then the title bar has the update "v2.0.0"

    Scenario: The notice offers the download
      Given the environment is set to the following:
        | latest release | 2.0.0 |
      And the application "editor" starts
      And the title bar has the update "v2.0.0"
      When the person opens the update notice
      Then the update notice has the button "Download"

    Scenario: A release that comes out later is announced
      Given the application "editor" starts
      And the title bar does not have an update
      When the environment is set to the following:
        | latest release | 2.0.0 |
      Then the title bar has the update "v2.0.0"

    Scenario: A withdrawn release is not announced
      Given the environment is set to the following:
        | latest release | 2.0.0 |
      And the application "editor" starts
      And the title bar has the update "v2.0.0"
      When the environment is set to the following:
        | latest release | none |
      Then the title bar does not have an update

    Scenario: An older release is not announced
      Given the environment is set to the following:
        | latest release | 2.0.0 |
      And the application "editor" starts
      And the title bar has the update "v2.0.0"
      When the environment is set to the following:
        | latest release | 0.0.1 |
      Then the title bar does not have an update

    Scenario: A missing feed address is a failed task
      Given the application "editor" starts
      When the person clears the setting "Release feed address" of the plugin "Update Notifier"
      And the person opens the task list
      Then the task list has the task "The Custom channel needs a release feed address."
