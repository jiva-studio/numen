Feature: Numen Sync keeps running

  The plugin "Numen Sync" runs while it is enabled, also when nobody uses it.

  As a person who uses Numen Sync
  I want the plugin to keep running in the background
  So that my changes can leave this device without my help

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity

  Rule: A plugin that must stay on runs while it is enabled

    Scenario: Numen Sync runs after the application starts
      When the application "editor" starts
      Then the plugin "Numen Sync" runs

    Scenario: Numen Sync runs again after its process ends
      Given the application "editor" starts
      When the process of the plugin "Numen Sync" ends
      Then the plugin "Numen Sync" runs

    Scenario: A disabled Numen Sync does not run
      Given the application "editor" starts
      When the person disables the plugin "Numen Sync"
      Then the plugin "Numen Sync" does not run

    Scenario: An enabled Numen Sync runs again
      Given the application "editor" starts
      And the person disables the plugin "Numen Sync"
      When the person enables the plugin "Numen Sync"
      Then the plugin "Numen Sync" runs
