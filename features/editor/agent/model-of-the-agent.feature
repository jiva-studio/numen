Feature: Model of the agent
  A person sees the model of the agent.

  Scenario: The model is the model of the settings
    Given the empty vault "Test"
    And the application "editor" starts
    And the person runs the command "Open agent chat"
    Then the model of the agent is "Test model"
