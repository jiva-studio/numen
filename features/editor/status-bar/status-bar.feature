Feature: Status bar
  A person turns on a bar at the bottom of the window.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The bar can be turned on

    Scenario: The bar is off at the start
      Given the title bar has the button "Open file tree"
      Then the plugin "Status bar" is disabled
      And the window does not have the status bar

    Scenario: The bar shows after the plugin is turned on
      When the person enables the plugin "Status bar"
      Then the plugin "Status bar" is enabled
      And the window has the status bar
