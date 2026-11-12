Feature: Delete text
  A person deletes a text of a file, and the text before it comes back where there is one.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The only text of a file is deleted

    Scenario: A transcript is deleted
      Given the vault has the following:
        | path               | sample      |
        | assets/lecture.wav | lecture.wav |
      And the person transcribes the recording at "assets/lecture.wav"
      And the person searches the vault for "Mock transcribed"
      And the search has the result "assets/lecture.wav" in the group "Text"
      And the person presses "Escape"
      When the person deletes the transcript of the file at "assets/lecture.wav"
      Then the button "Transcribe" in the tab at "assets/lecture.wav" is enabled
      And the tab at "assets/lecture.wav" does not have "Mock transcribed"
      And the person searches the vault for "Mock transcribed"
      And the search does not have the result "assets/lecture.wav" in the group "Text"

    Scenario: A recognized text is deleted
      Given the vault has the following:
        | path                            | sample                   |
        | assets/regenerative-braking.pdf | regenerative-braking.pdf |
      And the person recognizes the text of the file at "assets/regenerative-braking.pdf"
      And the person searches the vault for "recovers kinetic energy"
      And the search has the result "assets/regenerative-braking.pdf" in the group "Text"
      And the person presses "Escape"
      When the person deletes the recognized text of the file at "assets/regenerative-braking.pdf"
      And the person searches the vault for "recovers kinetic energy"
      Then the search does not have the result "assets/regenerative-braking.pdf" in the group "Text"

  Rule: A proofread text is deleted

    Scenario: A transcript comes back
      Given the vault has the following:
        | path               | sample      |
        | assets/lecture.wav | lecture.wav |
      And the person transcribes the recording at "assets/lecture.wav"
      And the person proofreads the transcript of the file at "assets/lecture.wav"
      And the tab at "assets/lecture.wav" has "[proofread]"
      When the person deletes the proofread text of the file at "assets/lecture.wav"
      Then the tab at "assets/lecture.wav" has "Mock transcribed"
      And the tab at "assets/lecture.wav" does not have "[proofread]"

    Scenario: A recognized text comes back
      Given the vault has the following:
        | path                            | sample                   |
        | assets/regenerative-braking.pdf | regenerative-braking.pdf |
      And the person recognizes the text of the file at "assets/regenerative-braking.pdf"
      And the person proofreads the recognized text of the file at "assets/regenerative-braking.pdf"
      And the person searches the vault for "proofread"
      And the search has the result "assets/regenerative-braking.pdf" in the group "Text"
      And the person presses "Escape"
      When the person deletes the proofread text of the file at "assets/regenerative-braking.pdf"
      And the person searches the vault for "proofread"
      Then the search does not have the result "assets/regenerative-braking.pdf" in the group "Text"
