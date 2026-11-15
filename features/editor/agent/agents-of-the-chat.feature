Feature: Agents of the chat
  The chat answers through the agents a person chooses on the page of Agent Chat. An agent program
  installed on this computer is found and listed with no setup.

  As a person who uses the agent
  I want the agents installed on my computer to be ready in the chat
  So that I can ask them without setting them up

  Background:
    Given the empty vault "Test"
    And the program "Claude Code" is installed on this computer
    And the application "editor" starts
    And the person enables the plugin "Claude Code Agent"

  Scenario: An agent found on this computer is listed as a connection
    Then the connections list has the following connections:
      | name        | type        |
      | Claude Code | Claude Code |
    And the connection "Claude Code" shows "Discovered"
    And the connection "Claude Code" cannot be opened

  Scenario: With no agent chosen the chat uses every agent
    When the person removes every row of the setting "Agents" of the plugin "Agent Chat"
    Then the setting "Agents" of the plugin "Agent Chat" shows "Automatic: Numen agent, Claude Code"
