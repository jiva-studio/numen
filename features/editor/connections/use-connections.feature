Feature: Use connections
  A plugin works through the connections a person chooses on its page.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The search by meaning works through a connection

    Background:
      Given the vault has the following:
        | path          | text                  |
        | notes/Boat.md | A boat leaves at six. |
      And the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person sets the setting "Source" of the plugin "Vector Embeddings" to "Connection"

    Scenario: A note is found by meaning through a connection
      Given the person sets the setting "Connection" of the plugin "Vector Embeddings" to the following rows:
        | connection | model    |
        | Main       | embedder |
      And the person answers "Index" to the question:
        """
        The vault will be indexed again: 1 note through Main.
        This can take a while and may cost money.
        """
      When the person searches the vault for "A boat leaves at six."
      Then the search has the result "A boat leaves at six" in the group "Meaning"

    Scenario: A busy service is named in the task list
      Given the service "Main service" is busy
      And the person sets the setting "Connection" of the plugin "Vector Embeddings" to the following rows:
        | connection | model    |
        | Main       | embedder |
      When the person answers "Index" to "The vault will be indexed again: 1 note through Main"
      Then the task list has the failed task "Main: rate limited"

    Scenario: The rows are kept when another source is chosen
      Given the person sets the setting "Connection" of the plugin "Vector Embeddings" to the following rows:
        | connection | model    |
        | Main       | embedder |
      And the person answers "Index" to "The vault will be indexed again: 1 note through Main"
      And the person sets the setting "Source" of the plugin "Vector Embeddings" to "Local model"
      When the person sets the setting "Source" of the plugin "Vector Embeddings" to "Connection"
      Then the setting "Connection" of the plugin "Vector Embeddings" has the following rows:
        | connection | model    |
        | Main       | embedder |

  Rule: Proofreading tries the rows in order

    Background:
      Given the transcribed recording "assets/lecture.wav"
      And the service "Busy service" is busy
      And the following connections:
        | name | type                     | service      |
        | Busy | OpenAI-compatible server | Busy service |
        | Main | OpenAI-compatible server | Main service |

    Scenario: The next row proofreads when the first is busy
      Given the person sets the setting "Connections" of the plugin "LLM Proofreading" to the following rows:
        | connection | model   |
        | Busy       | chatter |
        | Main       | chatter |
      When the person proofreads the transcript of the file at "assets/lecture.wav"
      Then the file at "assets/lecture.wav" has the proofread text

    Scenario: A busy only row is named in the task list
      Given the person sets the setting "Connections" of the plugin "LLM Proofreading" to the following rows:
        | connection | model   |
        | Busy       | chatter |
      When the person proofreads the transcript of the file at "assets/lecture.wav"
      Then the task list has the failed task "Busy: rate limited"

  Rule: A service works only with its key

    Background:
      Given the transcribed recording "assets/lecture.wav"
      And the service "Main service" accepts only the key "invented-key"
      And the connection "Main" of the type "OpenAI-compatible server" for the service "Main service"
      And the person sets the setting "Connections" of the plugin "LLM Proofreading" to the following rows:
        | connection | model   |
        | Main       | chatter |

    Scenario: A connection without the key is refused
      When the person proofreads the transcript of the file at "assets/lecture.wav"
      Then the task list has the failed task "Main: key rejected"

    Scenario: A connection with the key works
      Given the person gives the connection "Main" the key "invented-key"
      When the person proofreads the transcript of the file at "assets/lecture.wav"
      Then the file at "assets/lecture.wav" has the proofread text
