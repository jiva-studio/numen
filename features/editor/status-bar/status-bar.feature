Feature: Status bar
  A person turns on a bar at the bottom of the window.

  As a person who uses the editor
  I want to turn on the status bar with its plugin
  So that the bar shows what other plugins place in it, and is not there when I do not want it

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
