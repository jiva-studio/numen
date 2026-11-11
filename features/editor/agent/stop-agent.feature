Feature: Stop agent
  A person sees that the agent is working and can stop it.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts
    And the person runs the command "Open agent chat"
    And the agent holds its answers

  Scenario: The agent is working
    When the person sends the message "Which pear trees grow in the east orchard?" to the agent
    Then the composer of the agent has the button "Stop the agent"
    And the Agent tab titled "Agent" does not have "The agent answered."

  Scenario: The answer ends the wait
    Given the person sends the message "Which pear trees grow in the east orchard?" to the agent
    When another program lets the agent answer
    Then the Agent tab titled "Agent" has "The agent answered."
    And the composer of the agent does not have the button "Stop the agent"
    And the composer of the agent cannot send

  Scenario: The agent is stopped
    Given the person sends the message "Count the apple trees by the gate" to the agent
    When the person stops the agent
    Then the composer of the agent does not have the button "Stop the agent"
    And the composer of the agent cannot send
