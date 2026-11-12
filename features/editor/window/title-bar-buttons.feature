Feature: Title bar buttons
  A person runs the commands of the plugins with the buttons of the title bar.

  As a person who uses the editor
  I want to run a command with a button of the title bar
  So that I reach the commands I use often with one action

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A button runs its command

    Scenario: The file tree is opened with its button
      Given the Files tab titled "Files" is open
      And the person closes the Files tab titled "Files"
      When the person clicks the button "Open file tree" in the title bar
      Then the Files tab titled "Files" is open

    Scenario: A note is made with its button
      When the person clicks the button "Create note" in the title bar with:
        | Name | Brine log |
      Then the vault contains the note titled "Brine log"
      And the note tab titled "Brine log" is active

  Rule: A button belongs to its plugin

    Scenario: The button of a disabled plugin is removed
      Given the title bar has the button "Open file tree"
      When the person disables the plugin "File Tree Explorer"
      Then the title bar does not have the button "Open file tree"

    Scenario: The button of an enabled plugin comes back
      Given the person disables the plugin "File Tree Explorer"
      When the person enables the plugin "File Tree Explorer"
      Then the title bar has the button "Open file tree"
