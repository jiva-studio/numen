Feature: Manage connections
  A person keeps the connections to the services and programs that plugins use.

  As a person who uses a language model service
  I want to add a connection once, with its key
  So that every plugin can use it and the key stays private

  Background:
    Given the empty vault "Test"
    And the service "Main service" answers with the status "200"
    And the application "editor" starts

  Rule: A connection is added and deleted

    Scenario: An added connection is listed
      When the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      Then the connections list has the connection "Main"

    Scenario: A deleted connection leaves the list
      Given the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      When the person deletes the connection "Main"
      Then the connections list does not have the connection "Main"

    Scenario: A deleted connection takes its rows with it
      Given the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person adds a row of the connection "Main" with the model "chatter" to the plugin "LLM Proofreading"
      And the plugin "LLM Proofreading" has 1 row
      When the person deletes the connection "Main"
      Then the plugin "LLM Proofreading" has 0 rows

    Scenario: A connection of a type that no plugin gives is unavailable
      Given the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      When the person disables the plugin "Language model services"
      Then the connection "Main" shows "Type unavailable"

  Rule: A connection keeps its key

    Scenario: A key is stored
      Given the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      When the person gives the connection "Main" the key "invented-key"
      Then the connection "Main" has a key

    Scenario: A key is removed
      Given the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person gives the connection "Main" the key "invented-key"
      When the person removes the key of the connection "Main"
      Then the connection "Main" does not have a key
