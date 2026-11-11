Feature: Appearance and performance settings
  A person chooses how the window looks and how hard it works, and the choice stays.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A choice is kept when the window starts again

    Scenario: The performance profile is kept
      When the person runs the command "Performance profile" with:
        | Performance profile | Eco |
      And the application "editor" restarts
      Then the setting "Performance profile" is "Eco"

    Scenario: Dark is kept
      When the person runs the command "Light or dark" with:
        | Light or dark | Dark |
      And the application "editor" restarts
      Then the setting "Light or dark" is "Dark"

    Scenario: A theme is kept
      When the person runs the command "Theme" with:
        | Theme | Dracula |
      And the application "editor" restarts
      Then the setting "Theme" is "Dracula"

  Rule: A theme belongs to its plugin

    Scenario: A theme is chosen in the settings
      When the person sets the setting "Theme" to "Dracula"
      Then the setting "Theme" is "Dracula"

    Scenario: The theme of a disabled plugin is not offered
      Given the person sets the setting "Theme" to "Dracula"
      When the person disables the plugin "Preset Themes"
      Then the setting "Theme" does not have the option "Dracula"
