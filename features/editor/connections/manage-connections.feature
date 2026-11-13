Feature: Manage connections
  A person keeps the connections to the services and programs that plugins use.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A connection is added and deleted

    Scenario: An added connection is listed
      When the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      Then the connections list has the following connections:
        | name | type                     |
        | Main | OpenAI-compatible server |

    Scenario: A deleted connection takes its rows with it
      Given the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person sets the setting "Connections" of the plugin "LLM Proofreading" to the following rows:
        | connection | model   |
        | Main       | chatter |
      When the person deletes the connection "Main"
      Then the setting "Connections" of the plugin "LLM Proofreading" has no rows

    Scenario: A connection of a disabled plugin is unavailable
      Given the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      When the person disables the plugin "Language model services"
      Then the connection "Main" shows "Type unavailable"

  Rule: The key stays private

    Scenario: A saved key is not shown again
      Given the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person gives the connection "Main" the key "invented-key"
      When the person opens the connection "Main"
      Then the key field shows that a key is saved
      And the dialog does not show "invented-key"
