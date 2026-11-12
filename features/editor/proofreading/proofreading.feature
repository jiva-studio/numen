Feature: Proofreading
  A person proofreads the text of a scan or of a recording.

  As a person who works with scans and recordings
  I want to correct the text of a scan or of a recording
  So that I can find it by the corrected words

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A scan is proofread

    Background:
      Given the vault has the following:
        | path                      | sample             |
        | assets/pumped-storage.pdf | pumped-storage.pdf |
      And the person recognizes the text of the file at "assets/pumped-storage.pdf"
      And the person proofreads the recognized text of the file at "assets/pumped-storage.pdf"

    Scenario: The proofread text is found
      When the person searches the vault for "proofread"
      Then the search has the result "assets/pumped-storage.pdf" in the group "Text"

    Scenario: The text before the correction is still found
      When the person searches the vault for "upper reservoir"
      Then the search has the result "assets/pumped-storage.pdf" in the group "Text"

  Rule: A recording is proofread

    Background:
      Given the vault has the following:
        | path               | sample      |
        | assets/lecture.wav | lecture.wav |
      And the person transcribes the recording at "assets/lecture.wav"
      And the person proofreads the transcript of the file at "assets/lecture.wav"

    Scenario: The transcript has the proofread text
      Then the tab at "assets/lecture.wav" has "[proofread]"

    Scenario: The proofread transcript is found
      When the person searches the vault for "proofread"
      Then the search has the result "assets/lecture.wav" in the group "Text"
