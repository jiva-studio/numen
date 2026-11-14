Feature: Automatic proofreading
  A person lets Numen proofread every new text of a scan or of a recording.

  As a person who works with scans and recordings
  I want the new texts to be proofread without a request
  So that I find them by the corrected words

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A new text is proofread when the setting is on

    Background:
      Given the person sets the setting "Proofread automatically" of the plugin "LLM Proofreading" to "On"

    Scenario: A new recognized text is proofread
      Given the vault has the following:
        | path                      | sample             |
        | assets/pumped-storage.pdf | pumped-storage.pdf |
      When the person recognizes the text of the file at "assets/pumped-storage.pdf"
      Then the file at "assets/pumped-storage.pdf" has the proofread text

    Scenario: A new transcript is proofread
      Given the vault has the following:
        | path               | sample      |
        | assets/lecture.wav | lecture.wav |
      When the person transcribes the recording at "assets/lecture.wav"
      Then the file at "assets/lecture.wav" has the proofread text

    Scenario: A busy connection fails the proofreading
      Given the service "Busy service" is busy
      And the following connections:
        | name | type                     | service      |
        | Busy | OpenAI-compatible server | Busy service |
      And the person sets the setting "Connections" of the plugin "LLM Proofreading" to the following rows:
        | connection | model   |
        | Busy       | chatter |
      And the vault has the following:
        | path               | sample      |
        | assets/lecture.wav | lecture.wav |
      When the person transcribes the recording at "assets/lecture.wav"
      Then the task list has the failed task "Busy: rate limited"

  Rule: A text is not proofread when the setting is off

    Scenario: A new recognized text stays as it is
      Given the vault has the following:
        | path                      | sample             |
        | assets/pumped-storage.pdf | pumped-storage.pdf |
      When the person recognizes the text of the file at "assets/pumped-storage.pdf"
      Then there is no proofread text of the file at "assets/pumped-storage.pdf"

  Rule: A text that exists is proofread when the setting is switched on

    Scenario: A recognized text is proofread when the setting is switched on
      Given the vault has the following:
        | path                      | sample             |
        | assets/pumped-storage.pdf | pumped-storage.pdf |
      And the person recognizes the text of the file at "assets/pumped-storage.pdf"
      When the person sets the setting "Proofread automatically" of the plugin "LLM Proofreading" to "On"
      Then the file at "assets/pumped-storage.pdf" has the proofread text
