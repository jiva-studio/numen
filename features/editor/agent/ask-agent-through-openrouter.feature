@live @model
Feature: Ask agent through OpenRouter
  A person asks the agent a question and a real model answers it through a connection to OpenRouter.

  As a person who keeps notes
  I want the agent to answer through my OpenRouter key
  So that I can use a model of my choice

  Scenario: A question is answered by a model of OpenRouter
    Given the empty vault "Test"
    And the person has the following keys:
      | key name   | environment variable                  |
      | OpenRouter | NUMEN_ACCEPTANCE_TESTS_OPENROUTER_KEY |
    And the application "editor" starts
    And the person has the following connections:
      | name        | type        | key        |
      | OpenRouter  | OpenRouter  | OpenRouter |
      | Numen agent | Numen agent |            |
    And the person sets the setting "Agents" of the plugin "Agent Chat" to the following rows:
      | connection  | model                                      |
      | Numen agent | OpenRouter · Google: Gemini 2.5 Flash Lite |
    And the person runs the command "Open agent chat"
    When the person sends the message "Reply with one word." to the agent
    Then the agent has answered
