Feature: Tab of a disabled plugin
  A tab stays in the window when its plugin is disabled.

  As a person who uses plugins
  I want my tab to stay open when I disable its plugin
  So that I do not lose my place when I enable the plugin again

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | sample    |
      | assets/shore.pdf | shore.pdf |
    And the application "editor" starts
    And the tab at "assets/shore.pdf" is open at page "2"

  Scenario: The tab shows that the plugin is disabled
    When the person disables the plugin "PDF Reader"
    Then the tab at "assets/shore.pdf" is open
    And the tab at "assets/shore.pdf" has "Plugin disabled"

  Scenario: The tab returns when the plugin is enabled
    Given the person disables the plugin "PDF Reader"
    When the person enables the plugin "PDF Reader"
    Then the tab at "assets/shore.pdf" is open
    And the tab at "assets/shore.pdf" is at page "2"
