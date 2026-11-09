Feature: Proofreading
  A person proofreads the text of a scan or of a recording.

  Background:
    Given the application "editor" starts

  Rule: The text of a scan is proofread

    Scenario: The proofread text of a scan is found
      Given the vault "Test" contains the file at "assets/pumped-storage.pdf"
      And the person opens the file at "assets/pumped-storage.pdf"
      And the person runs the command "Create text"
      And the person picks "Recognize text"
      When the person runs the command "Proofread text"
      And the person picks "Recognized text (current)"
      And the person searches the vault for "proofread"
      Then the search has the result "assets/pumped-storage.pdf" in the group "Text"

    Scenario: The recognized text of a scan is still found after the proofreading
      Given the vault "Test" contains the file at "assets/pumped-storage.pdf"
      And the person opens the file at "assets/pumped-storage.pdf"
      And the person runs the command "Create text"
      And the person picks "Recognize text"
      When the person runs the command "Proofread text"
      And the person picks "Recognized text (current)"
      And the person searches the vault for "upper reservoir"
      Then the search has the result "assets/pumped-storage.pdf" in the group "Text"

  Rule: The text of a recording is proofread

    Scenario: The proofread transcript of a recording has the proofread text
      Given the vault "Test" contains the file at "assets/lecture.wav"
      And the person opens the file at "assets/lecture.wav"
      And the person runs the command "Transcribe"
      When the person runs the command "Proofread text"
      And the person picks "Transcript (current)"
      Then the tab at "assets/lecture.wav" has "[proofread]"

    Scenario: The proofread transcript of a recording is found by search
      Given the vault "Test" contains the file at "assets/lecture.wav"
      And the person opens the file at "assets/lecture.wav"
      And the person runs the command "Transcribe"
      When the person runs the command "Proofread text"
      And the person picks "Transcript (current)"
      And the person searches the vault for "proofread"
      Then the search has the result "assets/lecture.wav" in the group "Text"
