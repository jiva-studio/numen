Feature: Stop agent
  A person stops the agent before it answers.

  Background:
    Given the empty vault "Test"
    And the environment is set to the following:
      | agent | slow |
    And the application "editor" starts
    And the person runs the command "Open agent chat"

  Scenario: A stopped agent does not answer
    Given the person sends the message "Count the apple trees by the gate" to the agent
    When the person stops the agent
    And the environment is set to the following:
      | agent | normal |
    Then the Agent tab titled "Agent" does not have "The agent answered."

  Scenario: A person asks again after the stop
    Given the person sends the message "Count the apple trees by the gate" to the agent
    And the person stops the agent
    When the person sends the message "Count the pear trees by the gate" to the agent
    And the environment is set to the following:
      | agent | normal |
    Then the Agent tab titled "Agent" has "The agent answered."
