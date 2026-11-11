Feature: Update notice
  A person is told when a newer release of Numen is published.

  Background:
    Given the empty vault "Test"

  Rule: The release feed decides the notice

    Scenario: A newer release is announced at the start
      Given the release feed names the version "2.0.0"
      And the application "editor" starts
      Then the title bar shows the update "v2.0.0"
      When the person opens the update notice
      Then the update notice has the button "Download"

    Scenario: A release published later is announced
      Given the application "editor" starts
      And the title bar does not show an update
      When the release feed names the version "2.0.0"
      And the person sets the setting "Release feed address" of the plugin "Update Notifier" to the address of the release feed
      Then the title bar shows the update "v2.0.0"

    Scenario: A withdrawn release is not announced
      Given the release feed names the version "2.0.0"
      And the application "editor" starts
      And the title bar shows the update "v2.0.0"
      When the release feed names no newer release
      And the person sets the setting "Release feed address" of the plugin "Update Notifier" to the address of the release feed
      Then the title bar does not show an update

    Scenario: A missing feed address is a failed task
      Given the application "editor" starts
      When the person sets the setting "Release feed address" of the plugin "Update Notifier" to ""
      And the person opens the task list
      Then the task list has the task "The Custom channel needs a release feed address. Set it in the settings of Update Notifier."
