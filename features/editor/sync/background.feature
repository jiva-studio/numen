Feature: Numen Sync keeps working

  Numen Sync works while it is on, also when nobody uses it.

  As a person who uses Numen Sync
  I want Numen Sync to keep working in the background
  So that my changes can leave this device without my help

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity

  Rule: Numen Sync works while it is on

    Scenario: Numen Sync works after the application starts
      When the application "editor" starts
      Then Numen Sync works

    Scenario: Numen Sync works again after it stops by itself
      Given the application "editor" starts
      When Numen Sync stops by itself
      Then Numen Sync works

    Scenario: A disabled Numen Sync does not work
      Given the application "editor" starts
      When the person disables the plugin "Numen Sync"
      Then Numen Sync does not work

    Scenario: An enabled Numen Sync works again
      Given the application "editor" starts
      And the person disables the plugin "Numen Sync"
      When the person enables the plugin "Numen Sync"
      Then Numen Sync works
