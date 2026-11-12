Feature: Stop agent
  A person stops the agent before it answers.

  Scenario: A person asks again after the stop
    Given the empty vault "Test"
    And the environment is set to the following:
      | agent | slow |
    And the application "editor" starts
    And the person runs the command "Open agent chat"
    And the person sends the message "Count the apple trees by the gate" to the agent
    And the person stops the agent
    And the environment is set to the following:
      | agent | normal |
    When the person sends the message "Count the pear trees by the gate" to the agent
    Then the Agent tab titled "Agent" has "The agent answered."
