Feature: Model of the agent
  A person sees the model of the agent.

  As a person who uses the agent
  I want to see which model the agent uses
  So that I know what answers me

  Scenario: The agent shows the model of the settings
    Given the empty vault "Test"
    And the application "editor" starts
    When the person runs the command "Open agent chat"
    Then the model of the agent is "stand-in/model"
