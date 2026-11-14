Feature: Recordings
  A person plays, transcribes and edits a recording, and finds it by its words.

  As a person who keeps recordings
  I want to play a recording, make its transcript, edit it and search it
  So that I can find a recording by what is said in it

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path               | sample      |
      | assets/lecture.wav | lecture.wav |
    And the application "editor" starts

  Rule: A recording is used from its tab

    Scenario: A recording is played
      Given the person opens the file at "assets/lecture.wav"
      When the person plays the recording in the tab at "assets/lecture.wav"
      Then the recording in the tab at "assets/lecture.wav" is playing

    Scenario: A recording is transcribed
      Given the person opens the file at "assets/lecture.wav"
      When the person clicks the button "Transcribe" in the tab at "assets/lecture.wav"
      Then the tab at "assets/lecture.wav" has "Mock transcribed"

  Rule: A transcript is found

    Background:
      Given the person transcribes the recording at "assets/lecture.wav"

    Scenario: A recording is found by the words of its transcript
      When the person searches the vault for "Mock transcribed"
      Then the search has the result "assets/lecture.wav" in the group "Text"

    Scenario: A found recording opens with its transcript
      Given the person searches the vault for "Mock transcribed"
      When the person chooses the search result "assets/lecture.wav" in the group "Text"
      Then the tab at "assets/lecture.wav" has a recording
      And the tab at "assets/lecture.wav" has "Mock transcribed"

  Rule: A transcript is edited

    Background:
      Given the person transcribes the recording at "assets/lecture.wav"
      And the person writes "Quay" in the transcript of the tab at "assets/lecture.wav"

    Scenario: An edit stays after the application restarts
      When the application "editor" restarts
      And the person opens the file at "assets/lecture.wav"
      Then the tab at "assets/lecture.wav" has "Quay"

    Scenario: An edit is found by search
      When the person searches the vault for "Quay"
      Then the search has the result "assets/lecture.wav" in the group "Text"
