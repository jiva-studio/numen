Feature: Use connections
  A plugin uses the connections that a person chooses in its settings.

  As a person who uses a language model service
  I want to choose a connection and a model for a plugin
  So that the plugin works through the service

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The search by meaning uses a connection

    Background:
      Given the vault has the following:
        | path          | text                  |
        | notes/Boat.md | A boat leaves at six. |

    Scenario: A note is found by meaning through a connection
      Given the service "Main service" answers with the status "200"
      And the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person sets the setting "Source" of the plugin "Vector Embeddings" to "Connection"
      And the person adds a row of the connection "Main" with the model "embedder" to the plugin "Vector Embeddings"
      And the question reads "This indexes 1 note through Main. It may cost money."
      When the person answers the question with "Index"
      Then the search has the result "A boat leaves at six" in the group "Meaning"

    Scenario: A service that asks to wait is named in the task list
      Given the service "Busy service" answers with the status "429"
      And the person adds the connection "Busy" of the type "OpenAI-compatible server" for the service "Busy service"
      And the person sets the setting "Source" of the plugin "Vector Embeddings" to "Connection"
      And the person adds a row of the connection "Busy" with the model "embedder" to the plugin "Vector Embeddings"
      When the person answers the question with "Index"
      Then the task list has the failed task "Busy: rate limited"

  Rule: Proofreading uses the rows in order

    Background:
      Given the vault has the following:
        | path               | sample      |
        | assets/lecture.wav | lecture.wav |
      And the person transcribes the recording at "assets/lecture.wav"
      And the service "Busy service" answers with the status "429"
      And the person adds the connection "Busy" of the type "OpenAI-compatible server" for the service "Busy service"
      And the person adds a row of the connection "Busy" with the model "chatter" to the plugin "LLM Proofreading"

    Scenario: A row that asks to wait is named in the task list
      When the person proofreads the transcript of the file at "assets/lecture.wav"
      Then the task list has the failed task "Busy: rate limited"

    Scenario: The next row proofreads when the first row asks to wait
      Given the service "Main service" answers with the status "200"
      And the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person adds a row of the connection "Main" with the model "chatter" to the plugin "LLM Proofreading"
      When the person proofreads the transcript of the file at "assets/lecture.wav"
      Then the file at "assets/lecture.wav" has the proofread text

  Rule: The agent answers through a connection

    Scenario: The agent answers through a connection with a key
      Given the service "Main service" needs the key "invented-key"
      And the person adds the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person gives the connection "Main" the key "invented-key"
      And the person removes the row 1 of the plugin "Numen Agent"
      And the person adds a row of the connection "Main" with the model "chatter" to the plugin "Numen Agent"
      And the person runs the command "Open agent chat"
      When the person sends the message "Summarize thermodynamics" to the agent
      Then the Agent tab titled "Agent" has "echo:"
