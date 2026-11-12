Feature: Agents of the chat
  A person chooses the agents of the chat as connection rows on the page of the plugin.

  As a person who uses the agent
  I want to choose the agents of the chat on the page of the plugin
  So that I know which agent answers me

  Scenario: The agent of the chat is a row on the page of the plugin
    Given the empty vault "Test"
    And the application "editor" starts
    Then the plugin "Agent Chat" has 1 row
