Feature: Import video
  A person imports a video by its address and plays it.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts
    And the person imports the link "https://youtu.be/mill01"
    And the person downloads the text of the link "https://youtu.be/mill01"
    And the person downloads the video of the link "https://youtu.be/mill01"

  Scenario: The video plays
    When the person plays the recording in the tab for the web address "https://youtu.be/mill01"
    Then the recording in the tab for the web address "https://youtu.be/mill01" is playing
    And the tab for the web address "https://youtu.be/mill01" has "The mill pond fills behind the gate."

  Scenario: The video note opens from the file tree
    Given the application "editor" restarts
    When the person opens the note of the link "https://youtu.be/mill01" in the file tree
    Then the Media tab for the web address "https://youtu.be/mill01" has "The mill pond fills behind the gate."

  Scenario: The video is deleted
    Given the Media tab for the web address "https://youtu.be/mill01" has a recording
    When the person deletes the video of the link "https://youtu.be/mill01"
    Then the Media tab for the web address "https://youtu.be/mill01" does not have a recording
    And the tab for the web address "https://youtu.be/mill01" has "The mill pond fills behind the gate."

  Scenario: The video is downloaded again
    Given the person deletes the video of the link "https://youtu.be/mill01"
    And the Media tab for the web address "https://youtu.be/mill01" does not have a recording
    When the person downloads the video of the link "https://youtu.be/mill01"
    Then the Media tab for the web address "https://youtu.be/mill01" has a recording
